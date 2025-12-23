package frame

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	myCommon "gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/musae/gamelib/sensitive"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/utils"
	"gitee.com/aniwar2/musae/utils/net/httpreq"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	PASS               = "pass"
	REVIEW             = "review"
	BLOCK              = "block"
	SIGN_VERSION       = "2020-06-19"
	LETTERS            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	BIZ_TYPE_NICKNAME  = "10083_text_global_playername"
	BIZ_TYPE_SQUADNAME = "10083_text_global_squadname"
)

// 动态屏蔽词
var dynamicWordMgr = sensitive.New()

// InitDynamicWord 初始化
func (s *ActorServer) InitDynamicWord() {
	data, err := s.GetFromConfigCenter(db.KeyCfgGlobalDirtyWord)
	if err != nil {
		dynamicWordMgr.AddWord("")
		logger.Warn(err)
		return
	}
	if data != "" {
		words := strings.Split(data, "|")
		dynamicWordMgr.AddWord(words...)
	}
}

type UGCResult struct {
	Result int    `json:"result"` // 请求结果 0=成功
	Msg    string `json:"msg"`    // 提示内容
	Data   Data   `json:"data"`   // 详细数据
}

type Data struct {
	Replaced   string      `json:"replaced"`   // 文字内容，发现敏感词替换内容
	Suggestion string      `json:"suggestion"` // pass,review,block，通过，建议审核，拦截
	Label      string      `json:"label"`      // 内容安全标签
	Score      int         `json:"score"`      // 置信度 范围：[0,100]
	Details    []WordMatch `json:"details"`    // 命中敏感词是有输出
	TaskId     string      `json:"task_Id"`    // 任务ID
	Manual     bool        `json:"manual"`     // 是否有人工审核的流程，如果为true，那么就等待着回调吧
	Async      bool        `json:"async"`      // 2021-03-24 新增，如果是异步审核，比如视频，那么就为true,那就等待回调吧
}

// WordMatch 敏感词
type WordMatch struct {
	Word  string `json:"word"`  // 命中敏感词
	Start int    `json:"start"` // 开始位置
	End   int    `json:"end"`   // 结束位置
}

// CheckSpecialLetters
//
//	@Description: 特殊字符检查
//	@receiver s
//	@param str 给定的校验字符串
//	@param ignoreSpace 是否忽略空白符
//	@return bool 存在特殊字符则返回true
func (s *ActorServer) CheckSpecialLetters(str string, ignoreSpace bool) bool {
	for _, v := range str {
		// 不可显示字符
		if !unicode.IsGraphic(v) {
			return true
		}
		// 空白符
		if !ignoreSpace && unicode.IsSpace(v) {
			return true
		}
		// 掩码
		if unicode.IsMark(v) {
			return true
		}
		// 符号
		if unicode.IsSymbol(v) {
			return true
		}
	}
	// 配置的字符
	letters := []string{} /*strings.Split(excel.GetConfigMgr().GetCfg().NAME_SHIELD, ",")*/
	return strings.ContainsAny(str, strings.Join(letters, ""))
}

// CheckSensitiveWord
//
//	@Description: 屏蔽词校验接口
//	@receiver s
//	@param ctype 校验类型 1=玩家昵称，2=编队名称
//	@param content 待校验内容
//	@return bool 通过返回true，否则返回false
//	@return error
func (s *ActorServer) CheckSensitiveWord(ctype int32, content string) (bool, error) {

	logger.Debugf("CheckSensitiveWord: ctype=%v content=%v", ctype, content)

	// 空的？扔回业务层自己处理
	if content == "" {
		return false, errors.New("content is empty")
	}

	// 动态屏蔽词命中
	if ok, _ := dynamicWordMgr.FindIn(content); ok {
		return false, nil
	}

	// 远程和本地双重校验，任一接口返回false，则最终结果为false
	if conf.UGC().Switch > 0 {
		ok, err := CheckRemote(ctype, content)
		if err != nil || !ok {
			logger.Debugf("CheckRemote 判定非法 %s", content)
			return false, err
		}
	}
	if CheckLocal(content) {
		logger.Debugf("CheckLocal 判定非法 %s", content)
		return false, nil
	}

	return true, nil
}

// 本地屏蔽词接口
func CheckLocal(content string) bool {
	ok, _ := sensitive.GetSensitiveWordMgr().FindIn(content)
	return ok
}

// lilith屏蔽词接口
func CheckRemote(ctype int32, content string) (bool, error) {
	// 构造请求body
	body := make(map[string]interface{})
	switch ctype {
	case myCommon.CHECK_TYPE_PLAYERNAME:
		body["biz_type"] = BIZ_TYPE_NICKNAME
	case myCommon.CHECK_TYPE_SQUADNAME:
		body["biz_type"] = BIZ_TYPE_SQUADNAME
	default:
		// 没实现的类型
		return false, fmt.Errorf("unrealized biz type %d", ctype)
	}

	body["project_access_key"] = conf.UGC().AccessKey
	body["content"] = content
	body["uid"] = 0
	body["ext"] = ""
	logger.Debugf("request body: %+v", body)

	b, err := json.Marshal(body)
	if err != nil {
		return false, err
	}

	bodyCopy, err := utils.CloneAny[map[string]interface{}](body)
	if err != nil {
		return false, err
	}

	// 构造机审req
	baseUrl := conf.UGC().BaseUrl
	apiPath := conf.UGC().ApiPath
	secretId := conf.UGC().SecretId
	secretKey := conf.UGC().SecretKey

	// 签名
	now := time.Now()
	nonce := randStr(10)
	encStr, err := calRequestSign(http.MethodPost, apiPath, nonce, secretId, secretKey, now.Unix(), bodyCopy)
	if err != nil {
		return false, err
	}

	// 构造url
	queryStr := fmt.Sprintf("Nonce=%s&SecretId=%s&Timestamp=%d&Version=%s&Signature=%s", nonce, secretId, now.Unix(), SIGN_VERSION, encStr)
	urlPath := fmt.Sprintf("%s?%s", apiPath, queryStr)

	resp, err := httpreq.New(baseUrl).PostJSON(urlPath, b)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	logger.Debugf("request urlPath: %s , resp: %+v", urlPath, resp)

	// 判定莉莉丝接口返回
	if httpreq.IsOK(resp.StatusCode) {
		// 解析返回数据
		retData := UGCResult{}
		err = json.NewDecoder(resp.Body).Decode(&retData)
		if err != nil {
			return false, err
		}
		logger.Debugf("retData: %+v", retData)
		// 校验结果处理
		if retData.Result != 0 {
			return false, fmt.Errorf("check result is failed, code: %d", retData.Result)
		}

		// 审核通过
		data := retData.Data
		if data.Suggestion == BLOCK || (data.Suggestion == REVIEW && data.Score > 50) {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// 请求签名
func calRequestSign(method, path, nonce, secretId, secretKey string, ts int64, body map[string]interface{}) (string, error) {

	if method == "" || path == "" || secretId == "" || secretKey == "" || ts == 0 {
		return "", errors.New("invalid sign params")
	}

	// 填充其他数据
	body["Nonce"] = nonce
	body["SecretId"] = secretId
	body["Timestamp"] = strconv.FormatInt(ts, 10)
	body["Version"] = SIGN_VERSION

	// 排序
	var keys []string
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接query字符串
	var queries []string
	for _, v := range keys {
		queries = append(queries, fmt.Sprintf("%s=%v", v, body[v]))
	}
	// 拼接sign字符串
	signStr := fmt.Sprintf("%s%s?%s", method, path, strings.Join(queries, "&"))
	logger.Debugf("calRequestSign signStr: %s", signStr)

	// 加密
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(signStr))
	b := mac.Sum(nil)
	encStr := base64.StdEncoding.EncodeToString(b)

	return url.QueryEscape(encStr), nil
}

// 随机字符串
func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = LETTERS[rand.Int63()%int64(len(LETTERS))]
	}
	return string(b)
}

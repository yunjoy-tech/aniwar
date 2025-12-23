package logic

import (
	"encoding/json"
	"errors"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/utils"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	randutil "gitee.com/aniwar2/musae/utils/rand"
	"io"
	"net/http"
	"strconv"
	"time"
)

// TaptapLoginResp taptap登陆请求结果
type TaptapLoginResp struct {
	Success bool            `json:"success"` // 成功为true，否则返回false
	Now     int64           `json:"now"`     // 当前时间戳
	Data    TaptapLoginData `json:"data"`    // 响应数据
}

// TaptapLoginData 账户信息
type TaptapLoginData struct {
	Name    string `json:"name"`    // 用户名
	Avatar  string `json:"avatar"`  // 用户头像图片地址
	OpenId  string `json:"openid"`  // 授权用户唯一标识，每个玩家在每个游戏中的 openid 都是不一样的，同一游戏获取同一玩家的 openid 总是相同
	UnionId string `json:"unionid"` // 授权用户唯一标识，一个玩家在一个厂商的所有游戏中 unionid 都是一样的，不同厂商 unionid 不同
}

func (s *LoginServer) handleAuthTaptap(unionId, accessToken, extra string) (int, *pb.TaptapUserInfo, pb.ErrorCode) {
	var uid int

	// 解析json数据
	taptapUser := &pb.TaptapUserInfo{}
	err := json.Unmarshal([]byte(extra), taptapUser)
	if err != nil {
		logger.Errorf("extra data unmarshal failed. extra=%s", extra)
		return uid, nil, pb.ErrorCode_Account_auth_fail
	}
	if accessToken == "" || unionId == "" || taptapUser.MacKey == "" || taptapUser.Token == "" {
		logger.Warnf("taptap, 登陆请求验证, 无请求参数")
		return uid, nil, pb.ErrorCode_Account_auth_fail
	}

	// 校验token
	clientId := conf.TapTap().ClientId

	// 随机数
	nonce := randutil.RandomStr(24, true, true, true)
	// 时间戳转换成字符串
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	// 请求url
	reqHost := conf.TapTap().BaseUrl
	reqURI := "/account/profile/v1?client_id=" + clientId
	reqURL := "https://" + reqHost + reqURI

	macStr := timestamp + "\n" + nonce + "\n" + "GET" + "\n" + reqURI + "\n" + reqHost + "\n" + "443" + "\n\n"
	mac := utils.HmacSha1(macStr, taptapUser.MacKey)
	authorization := "MAC id=" + "\"" + accessToken + "\"" + "," + "ts=" + "\"" + timestamp + "\"" + "," + "nonce=" + "\"" + nonce + "\"" + "," + "mac=" + "\"" + mac + "\""
	logger.Debugf("MAC校验串:%s", authorization)
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		logger.Error(err)
		return uid, nil, pb.ErrorCode_InternalError
	}

	// 添加请求头
	req.Header.Add("Authorization", authorization)
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(err)
		return uid, nil, pb.ErrorCode_InternalError
	}
	defer resp.Body.Close()
	// http请求异常
	if resp.StatusCode == http.StatusUnauthorized {
		return uid, nil, pb.ErrorCode_TapTapLoginInfoExpire
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return uid, nil, pb.ErrorCode_TapTapServerError
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err)
		return uid, nil, pb.ErrorCode_InternalError
	}
	logger.Debugf("taptap auth respond: %s", string(respBody))

	tapRes := &TaptapLoginResp{}
	err = json.Unmarshal(respBody, tapRes)
	if err != nil {
		return uid, nil, pb.ErrorCode_DeSerializeError
	}
	if !tapRes.Success || tapRes.Data.UnionId != unionId {
		return uid, nil, pb.ErrorCode_Account_auth_fail
	}

	// 映射查询
	uid, err = s.GetTaptapUid(unionId)
	if err != nil && !errors.Is(err, service.DB_ERROR_NOT_EXIST) {
		return uid, nil, pb.ErrorCode_Account_auth_fail
	}
	// 创建映射
	if uid == 0 {
		uid, err = s.UpdateTaptapUidCache(unionId)
		if err != nil {
			return uid, nil, pb.ErrorCode_Account_auth_fail
		}
	}
	return uid, taptapUser, pb.ErrorCode_Success
}

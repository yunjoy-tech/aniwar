package sdksign

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/sdkconstant/sdkrsa"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/utils"
	"github.com/samber/lo"
	"net/url"
	"sort"
	"strings"
)

// ParkSignVerify 验证签名
func ParkSignVerify(argsMap map[string]interface{}, excludeSignKeys []string) bool {
	// argsMap := ParseUrlArgs(params)

	signStr := buildSignArgsStr(argsMap, excludeSignKeys)

	signBytes, err := base64.StdEncoding.DecodeString(argsMap["sign"].(string))
	if err != nil {
		logger.Errorf(err.Error())
		return false
	}

	return sdkrsa.GetBillRsa().Verify([]byte(signStr), signBytes, crypto.SHA1)
}

// ParseUrlArgs 解析url参数
func ParseUrlArgs(argsStr string) map[string]interface{} {
	var (
		err     error
		argsMap = make(map[string]interface{})
	)

	// url decode
	unescape, err := url.QueryUnescape(argsStr)
	if err != nil {
		logger.Errorf(err.Error())
		return nil
	}

	params := strings.Split(unescape, "&")
	for _, param := range params {
		eachKV := strings.SplitN(param, "=", 2)

		if len(eachKV) != 2 {
			logger.Infof("无法解析参数:%s", param)
			continue
		}

		key := eachKV[0]
		val := eachKV[1]

		argsMap[key] = val
	}

	return argsMap
}

// buildSignArgsStr 构建参与签名的参数字符串
func buildSignArgsStr(argsMap map[string]interface{}, excludeSignKeys []string) string {
	// 过滤不参与签名的字段(sign和pay_env不参与验签)
	signDataMap := make(map[string]interface{})
	for key, val := range argsMap {
		if utils.SliceContain(excludeSignKeys, key) {
			// 过滤不参与签名的子弹
			continue
		}

		signDataMap[key] = val
	}

	// 按key的字典序取值
	keys := lo.Keys(signDataMap)
	sort.Strings(keys)
	strSlice := make([]string, 0, len(keys))
	for _, k := range keys {
		var value interface{}
		if signDataMap[k] != nil {
			value = signDataMap[k]
		}
		strSlice = append(strSlice, fmt.Sprintf("%s=%v", k, value))
	}

	return strings.Join(strSlice[:], "&")
}

package utils

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
)

// 注意：带base64编码
func HmacSha1(valStr, keyStr string) string {
	key := []byte(keyStr)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(valStr))

	// 进行 Base64 编码
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

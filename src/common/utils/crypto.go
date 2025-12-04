package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
)

func Md5Str(str string) string {
	m := md5.New()
	m.Write([]byte(str))
	return hex.EncodeToString(m.Sum(nil))
}

func Md5Str2(bytes []byte) string {
	m := md5.New()
	m.Write(bytes)
	return hex.EncodeToString(m.Sum(nil))
}

// 注意：带base64编码
func HmacSha1(valStr, keyStr string) string {
	key := []byte(keyStr)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(valStr))

	// 进行 Base64 编码
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

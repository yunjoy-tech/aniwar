package utils

import (
	"math/rand"
)

var lowers = []byte("abcdefghijklmnopqrstuvwxyz")
var uppers = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
var nums = []byte("0123456789")

// RandomStr 随机字符串，包含 1~9 和 a~z - [i,l,o]
func RandomStr(n int, hasLower, hasUpper, hasNum bool) string {
	if n <= 0 {
		return ""
	}

	charPool := make([]byte, 0)
	if hasLower {
		charPool = append(charPool, lowers...)
	}
	if hasUpper {
		charPool = append(charPool, uppers...)
	}
	if hasNum {
		charPool = append(charPool, nums...)
	}
	if len(charPool) <= 0 {
		return ""
	}

	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = charPool[rand.Int()%len(charPool)]
	}

	return string(b)
}

package utils

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestMd5(t *testing.T) {
	str := "tet---11"

	m := md5.New()
	m.Write([]byte(str))
	fmt.Println(hex.EncodeToString(m.Sum(nil)))

	data := []byte(str) //切片
	has := md5.Sum(data)
	fmt.Println(fmt.Sprintf("%x", has)) //将[]byte转成16进制
}

func TestLilithAuth(t *testing.T) {
	//reqData := "f945ee9808745f04ff7977d1a0c0b51beyJ1c2VyX2lkIjoyMjIzMywidHlwZSI6InF1ZXJ5X3VzZXIifQ==" // 22233
	reqData := "f20da8d6c3323a1810cbb4ceb2784f7beyJ1c2VyX2lkIjoxMTIsInR5cGUiOiJxdWVyeV91c2VyIn0=" // 112

	//验签和解析数据规则:
	//1. 截取数据前32个字符作为签名
	//2. 将数据剩余部分与APISecret拼接后算出md5值，与步骤1获取的签名做对比
	//3. 如果签名验证通过，则将步骤2中数据剩余部分用base64解码得到json格式的查询数据

	fmt.Println(fmt.Sprintf("check sign reqData: %s", reqData))
	signStr := string([]rune(reqData)[:32])
	dataStr := string([]rune(reqData)[32:])

	tempStr := fmt.Sprintf("%s%s", dataStr, "111111111")
	//tempStr := fmt.Sprintf("%s%s", dataStr, "1")

	fmt.Println(fmt.Sprintf("lilith 签名:%s", signStr))

	md5Str := Md5Str(tempStr)
	fmt.Println(fmt.Sprintf("self md5原文:%s", tempStr))
	fmt.Println(fmt.Sprintf("self 签名:%s", md5Str))

	if md5Str == signStr {
		// 解析数据
		_, err := base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

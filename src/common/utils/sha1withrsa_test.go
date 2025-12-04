package utils

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRsaSign(t *testing.T) {
	var data = "1234567890"
	prvKey, pubKey, err := genRsaKey()
	fmt.Println(string(prvKey))
	fmt.Println(string(pubKey))
	if err != nil {
		fmt.Printf("GenRsaKey error:%v\n", err)
	}

	signData, err := rsaSignWithSha1([]byte(data))
	fmt.Printf("RsaSignWithSha512 signData:%v, error:%v\n", signData, err)
	if RsaVerifySignWithSha1([]byte(data), signData) != nil {
		fmt.Println("check sign failed")
	} else {
		fmt.Println("check sign success")
	}
}

func TestRsaEncrypt(t *testing.T) {
	var data = "1234567890"
	prvKey, pubKey, err := genRsaKey()
	if err != nil {
		fmt.Printf("GenRsaKey error:%v", err)
	}
	fmt.Println(string(prvKey))
	fmt.Println(string(pubKey))

	encryptData, err := rsaEncrypt([]byte(data))
	fmt.Printf("encrypted data:%v,err:%v\n", hex.EncodeToString(encryptData), err)
	sourceData, err := rsaDecrypt(encryptData)
	fmt.Printf("decrypted data:%v,err:%v\n", string(sourceData), err)
}

type Point struct {
	X int    `json:"point_x"`
	Y string `json:"point_y"`
}

type KspayTest struct {
	Point
	Test_X string `json:"test_x"`
	Test_Y string `json:"test_y"`
}

func TestStruct2String(test *testing.T) {
	u := KspayTest{}
	t := reflect.TypeOf(u)
	v := reflect.ValueOf(u)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() { //判断是否为可导出字段
			//判断是否是嵌套结构
			if v.Field(i).Type().Kind() == reflect.Struct {
				structField := v.Field(i).Type()
				for j := 0; j < structField.NumField(); j++ {
					//fmt.Printf("%s %s = %v -tag:%s \n",
					//	structField.Field(j).Name,
					//	structField.Field(j).Type,
					//	v.Field(i).Field(j).Interface(),
					//	structField.Field(j).Tag)
					varName := structField.Field(j).Name
					varType := structField.Field(j).Type
					varValue := v.Field(i).Field(j).Interface()
					tag := structField.Field(j).Tag.Get("json")
					fmt.Printf("%v %v %v %v\n", varName, varType, varValue, tag)
				}
				continue
			}
			varName := t.Field(i).Name
			varType := t.Field(i).Type
			varValue := v.Field(i).Interface()
			tag := t.Field(i).Tag.Get("json")
			fmt.Printf("%v %v %v %v\n", varName, varType, varValue, tag)
		}
	}
}

/*
func TestReqBalance(t *testing.T) {
	ks := &KspayBalance{
		Game_token: "ChJvcGVucGxhdGZvcm0udG9rZW4SMM2uDBIaNxgwP8RYfxEfwUoP5BSjUnb9Cp2z5y7wzhqXfICyOVTtj68NuSM_7ABFzBoSghF_z6jJSeemzJZpZOfXpEfKIiDrmfxzTEnhnfwWi3UE3lq8qUsDYPdZHdgey7UeZv4gHigFMAE",
		Game_id:    "120592009db2d8e94d07fb38a265ef2b54",
		App_id:     "ks704531870435612361",
		Role_id:    "228",
		Os:         "android",
		User_ip:    "58.34.113.98",
		Ts:         strconv.FormatInt(time.Now().UnixNano()/1e6, 10),
		Extension:  "test",
		Url:        "https://allin.kuaishoupay.com/api/kpay/balance",
	}
	ret, err := KsReqBalance(ks)
	if err != nil {
		fmt.Printf("KsReqBalance err:%v", err)
	}
	fmt.Printf("KsReqBalance ret:%v", ret)
}

func TestReqPay(t *testing.T) {
	ks := &KspayPay{}
	ret, err := KsReqPay(ks)
	if err != nil {
		fmt.Printf("TestReqPay err:%v", err)
	}
	fmt.Printf("TestReqPay ret:%v", ret)
}

func TestReqPresent(t *testing.T) {
	ks := &KspayPresent{}
	ret, err := KsReqPresent(ks)
	if err != nil {
		fmt.Printf("TestReqPresent err:%v", err)
	}
	fmt.Printf("TestReqPresent ret:%v", ret)
}

func TestKsReqQueryOrder(t *testing.T) {
	ks := &KsQueryOrder{}
	ret, err := KsReqQueryOrder(ks)
	if err != nil {
		fmt.Printf("TestKsReqQueryOrder err:%v", err)
	}
	fmt.Printf("TestKsReqQueryOrder ret:%v", ret)
}

func TestKsReqAntispam(t *testing.T) {
	ks := &KsAntispam{}
	ret, err := KsReqAntispam(ks)
	if err != nil {
		fmt.Printf("TestKsReqAntispam err:%v", err)
	}
	fmt.Printf("TestKsReqAntispam ret:%v", ret)
}
*/

func TestKsOrderID(t *testing.T) {
	for i := 0; i < 10; i++ {
		orderId := strings.Join([]string{"1", "1", "1", strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)}, "o")
		fmt.Printf("TestKsOrderID orderId:%v\n", orderId)
		orderId = strings.Join([]string{"1", "1", "1", strconv.FormatInt(time.Now().UnixNano(), 10)}, "o")
		fmt.Printf("TestKsOrderID orderId:%v\n", orderId)
	}

}

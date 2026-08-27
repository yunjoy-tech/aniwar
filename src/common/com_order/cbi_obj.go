package com_order

import (
	"encoding/base64"
	"encoding/json"

	"github.com/yunjoy-tech/aniwar/src/common/aes"

	"github.com/yunjoy-tech/aniwar/src/common/db"

	"github.com/yunjoy-tech/musae/service"
)

// CBI_AES_KEY AES加解密的key
var CBI_AES_KEY = []byte("aniwar.ssgame!@#")

// CbiObj 透传参数
type CbiObj struct {
	AccountId string `json:"accountId"`
	Uaid      string `json:"uaid"`
	CpOrderId string `json:"cpOrderId"` // cp的订单id
	PayId     int32  `json:"payId"`     // 支付id
}

func BuildPayCbi(accountId string, uaid string, cpOrderId string, payId int32) string {
	cbi := &CbiObj{
		AccountId: accountId,
		Uaid:      uaid,
		CpOrderId: cpOrderId,
		PayId:     payId,
	}

	bytes, err := json.Marshal(cbi)
	if err != nil {
		return ""
	}

	encryptBytes, err := aes.AesEncrypt(bytes, CBI_AES_KEY)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(encryptBytes)
}

func ParsePayCbi(jsonStr string) (*CbiObj, error) {
	cbi := &CbiObj{}

	decodeBytes, err := base64.StdEncoding.DecodeString(jsonStr)
	if err != nil {
		return nil, err
	}

	decryptBytes, err := aes.AesDecrypt(decodeBytes, CBI_AES_KEY)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(decryptBytes, cbi)
	if err != nil {
		return nil, err
	}

	return cbi, nil
}

func OrderDBTable(accountId string) (service.MongoDbType, string) {
	return service.MongoDbType_MongoAccount, db.KeyUserOrderInfo(accountId)
}

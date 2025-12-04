package com_order

import "strconv"

// LilithPayCallbackReq Lilith支付回调参数
type LilithPayCallbackReq struct {
	Serial       int32  `json:"serial"`        // 订单号
	PayType      int32  `json:"pay_type"`      // 支付平台类型(iOS/Alipay/...)
	AppId        int32  `json:"app_id"`        // app id
	AppUid       int32  `json:"app_uid"`       // 用户平台id
	Ext          string `json:"ext"`           // CP自定义数据(长度限制255)
	Amount       int32  `json:"amount"`        // 支付金额（单位为分）
	ProductType  string `json:"product_type"`  // 商品ID
	AdditionInfo string `json:"addition_info"` // 附加参数,可能为空 (该附加信息为一些支付类型特有数据的集合。数据格式为json，需要先base64解码。 只有在有附加信息的时候，该参数才会下发并则参与签名)
	PayEnv       int32  `json:"pay_env"`       // 支付环境
	Sign         string `json:"sign"`          // 签名
	//Cbi          *com_order.CbiObj `json:"com_order"`           // 透传解析结构
}

// 解析参数
func ParseLilithPayCallbackReq(argsMap map[string]interface{}) *LilithPayCallbackReq {
	apiReq := &LilithPayCallbackReq{
		Serial:       getInt32(argsMap["serial"]),
		PayType:      getInt32(argsMap["pay_type"]),
		AppId:        getInt32(argsMap["app_id"]),
		AppUid:       getInt32(argsMap["app_uid"]),
		Ext:          getString(argsMap["ext"]),
		Amount:       getInt32(argsMap["amount"]),
		ProductType:  getString(argsMap["product_type"]),
		AdditionInfo: getString(argsMap["addition_info"]),
		PayEnv:       getInt32(argsMap["pay_env"]),
		Sign:         getString(argsMap["sign"]),
	}

	return apiReq
}

// 解析参数
func ParseLilithRefundReq(argsMap map[string]interface{}) *LilithRefundReq {
	apiReq := &LilithRefundReq{
		Serial: getInt32(argsMap["serial"]),
		AppId:  getInt32(argsMap["app_id"]),
		AppUid: getInt32(argsMap["app_uid"]),
		Ext:    getString(argsMap["ext"]),
		Sign:   getString(argsMap["sign"]),
	}

	return apiReq
}

// LilithRefundReq Lilith退款回调参数
type LilithRefundReq struct {
	Serial int32  `json:"serial"`  // 订单号
	AppId  int32  `json:"app_id"`  // app id
	AppUid int32  `json:"app_uid"` // 用户平台id TODO 需确认该字段是否有？文档中不存在
	Ext    string `json:"ext"`     // CP自定义数据(长度限制255)
	Sign   string `json:"sign"`    // 签名
}

func getInt32(i interface{}) int32 {
	s, ok := i.(string)
	if !ok {
		return 0
	}

	atoi, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return int32(atoi)
}

func getString(i interface{}) string {
	s, ok := i.(string)
	if !ok {
		return ""
	}

	return s
}

package sdkconstant

import (
	"encoding/base64"
	"fmt"
)

const (
	Taptap_channel            = "taptap"
	Tap_url_prefix            = "https://tds-tapsdk.cn.tapapis.com/anti-addiction/v1/clients/"
	Tap_url_check_play_limit  = Tap_url_prefix + "%s/users/%s/playable"
	Tap_url_check_pay_limit   = Tap_url_prefix + "%s/users/%s/payable"
	Tap_url_upload_pay_amount = Tap_url_prefix + "%s/users/%s/payments"
)

// type TaptapUserInfo struct {
//	AppUid   string `json:"appUid"`
//	AppToken string `json:"appToken"`
//	Token    string `json:"token"`
//	MacKey   string `json:"macKey"`
//	Result   int    `json:"result"`
//	Age      int    `json:"age"`
// }

// GenTaptapUid
//
//	@Description: 生成taptap专用uid
//	@param uid 原uid
//	@return string
func GenTaptapUid(uid int) string {
	return fmt.Sprintf("%s_%d", Taptap_channel, uid)
}

func GetTapUserIdentifier(userIdentifier string) string {
	// url encode
	return base64.URLEncoding.EncodeToString([]byte(userIdentifier))
}

// GenTaptapChannel
//
//	@Description:  生成taptap渠道
//	@return string
func GenTaptapChannel() string {
	return Taptap_channel
}

// GetCheckPayLimitUrl 检查玩家消费是否受限
func GetCheckPlayLimitUrl(clientId, userIdentifier string) string {
	return fmt.Sprintf(Tap_url_check_play_limit, clientId, userIdentifier)
}

// GetCheckPayLimitUrl 检查玩家消费是否受限
func GetCheckPayLimitUrl(clientId, userIdentifier string) string {
	return fmt.Sprintf(Tap_url_check_pay_limit, clientId, userIdentifier)
}

// GetUploadPayAmountUrl 上报充值金额
/*
上报成功时响应的状态码为 200，返回结果：
{
    "success":true,
    "data":{"message":"上传金额成功"}
}

金额格式异常时返回 400 错误：
{
    "success":false,
    "data":{
        "code":3,
        "error":"上传金额不正确",
        "error_description":"金额大于等于0并小于100_000_000_000","msg":"请输入正确的金额格式"
    }
}
*/
func GetUploadPayAmountUrl(clientId, userIdentifier string) string {
	return fmt.Sprintf(Tap_url_upload_pay_amount, clientId, userIdentifier)
}

package sdkutil

import (
	"encoding/json"
	"fmt"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"

	"github.com/pkg/errors"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/sdkconstant"
	myHttp "gitlab.musadisca-games.com/wangxw/musae/framework/http"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

type TapUrlResp struct {
	Success bool            `json:"success"`
	Data    *TapUrlRespData `json:"data"`
}

type TapUrlRespData struct {
	Code        int32  `json:"code"`
	Status      bool   `json:"status"`
	Message     string `json:"message"`
	Title       string `json:"title"`
	Description string `json:"description"`

	// got error
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Msg              string `json:"msg"`
}

// TapCheckPayLimit 检查玩家消费是否受限
/* 文档说明
curl -X POST \
-H "Content-Type: application/json" \
-H 'Authorization: {{token}}' \
-d '{"amount": 100}' \
https://tds-tapsdk.cn.tapapis.com/anti-addiction/v1/clients/{{clientId}}/users/{{userIdentifier}}/payable

消费受限和不受限时响应的状态码都是 200
// 受限
{
    "success":true,
    "data":{
        "code":200,
        "status":false,
        "message":"限额提示",
        "title":"健康消费提示",
        "description":"允许充值根据国家相关规定，未满8周岁：不提供付费服务；8-16周岁以下：单笔付费不超过50元，每月累计不超过200元；16-18周岁以下：单笔付费不超过100元，每月累计不超过400元。"
    }
}

// 允许
{
    "success":true,
    "data":{
        "code":200,
        "status":true,
        "message":"限额提示",
        "title":"健康消费提示",
        "description":"允许充值根据国家相关规定，未满8周岁：不提供付费服务；8-16周岁以下：单笔付费不超过50元，每月累计不超过200元；16-18周岁以下：单笔付费不超过100元，每月累计不超过400元。"
    }
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

实名认证失败（包括 Token 解析错误）时返回 401 错误：
{
    "success":false,
    "data":{
        "code":16,
        "error":"实名认证失败",
        "error_description":"未实名用户不能进入游戏",
        "msg":"该账号没有通过实名认证"
    }
}
*/
func TapCheckPayLimit(tapUserInfo *pb.TaptapUserInfo, tapToken, userIdentifier string, amount int32) (bool, pb.ErrorCode) {
	headMap := make(map[string]string, 0)
	headMap["Content-Type"] = "application/json"
	headMap["Authorization"] = tapToken

	resp := &TapUrlResp{}
	params := fmt.Sprintf("{\"amount\":%d}", amount)
	err := myHttp.Post2(
		sdkconstant.GetCheckPayLimitUrl(conf.GConf().TapTap.ClientId, sdkconstant.GetTapUserIdentifier(userIdentifier)),
		params,
		resp,
		headMap)
	if err != nil {
		logger.Errorf(err.Error())
		return false, pb.ErrorCode_Tap_pay_limit_code_unknown_err
	}

	respBytes, _ := json.Marshal(resp)
	logger.Warnf("tap 检查玩家消费是否受限:%s", string(respBytes))

	if resp == nil {
		err = errors.New(fmt.Sprintf("检查消费限制, resp为空"))
		logger.Errorf(err.Error())
		return false, pb.ErrorCode_Tap_pay_limit_code_unknown_err
	}

	if !resp.Success {
		err = errors.New(fmt.Sprintf("err:%s", resp.Data.ErrorDescription))
		logger.Errorf(err.Error())

		if resp.Data.Code == 3 {
			return false, pb.ErrorCode_Tap_pay_limit_code_pay_num_err
		} else if resp.Data.Code == 16 {
			return false, pb.ErrorCode_Tap_certification_fail
		}
	}

	if resp.Success {
		if resp.Data.Status {
			return true, pb.ErrorCode_Tap_pay_limit_code_can_pay // 允许消费
		} else {
			if tapUserInfo != nil {

				if tapUserInfo.Age < 8 {
					return false, pb.ErrorCode_Tap_pay_limit_code_limit_8
				} else if tapUserInfo.Age <= 15 {
					if amount > 50*100 /*单位:分*/ {
						return false, pb.ErrorCode_Tap_pay_limit_code_limit_15
					} else {
						return false, pb.ErrorCode_Tap_pay_limit_code_limit_15_total
					}
				} else if tapUserInfo.Age <= 17 {
					if amount > 100*100 /*单位:分*/ {
						return false, pb.ErrorCode_Tap_pay_limit_code_limit_17
					} else {
						return false, pb.ErrorCode_Tap_pay_limit_code_limit_17_total
					}
				}
			}
			return false, pb.ErrorCode_Tap_pay_limit_code_limit // 消费限制
		}
	}

	// bytes, err := json.Marshal(resp)
	// if err != nil {
	//	err = errors.Wrap(err, "json.Marshal got error")
	//	logger.Errorf(err.Error())
	//	return false, -1
	// }

	return false, pb.ErrorCode_Tap_pay_limit_code_unknown_err
}

// TapUploadPayAmount 上报充值金额
func TapUploadPayAmount(tapToken, userIdentifier string, amount int32) bool {
	headMap := make(map[string]string, 0)
	headMap["Content-Type"] = "application/json"
	headMap["Authorization"] = tapToken

	resp := &TapUrlResp{}
	params := fmt.Sprintf("{\"amount\":%d}", amount)
	err := myHttp.Post2(
		sdkconstant.GetUploadPayAmountUrl(conf.GConf().TapTap.ClientId, sdkconstant.GetTapUserIdentifier(userIdentifier)),
		params,
		resp,
		headMap)
	if err != nil {
		logger.Errorf(err.Error())
		return false
	}

	respBytes, _ := json.Marshal(resp)
	logger.Warnf("tap 上报充值金额:%s", string(respBytes))

	if resp == nil {
		err = errors.New(fmt.Sprintf("上报充值金额, resp为空"))
		logger.Errorf(err.Error())
		return false
	}

	if !resp.Success {
		err = errors.New(fmt.Sprintf("err:%s", resp.Data.ErrorDescription))
		logger.Errorf(err.Error())

		if resp.Data.Code == 3 {
			logger.Errorf("上报金额异常, amount=%d", amount)
			return false
		}
	} else {
		// 上报成功
		logger.Debugf("上报金额成功, amount=%d", amount)
		return true
	}

	return false
}

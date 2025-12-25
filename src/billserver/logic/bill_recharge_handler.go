package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitee.com/aniwar2/musae/gamelib/guid"
	netutil "gitee.com/aniwar2/musae/utils/net"
	"net/url"
	"strconv"

	"gitee.com/aniwar2/aniwar/src/common/com_order"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"google.golang.org/protobuf/proto"

	"gitee.com/aniwar2/aniwar/src/common/sdkconstant/sdksign"

	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/idipserver/logic"

	"github.com/dapr/go-sdk/service/common"
	"github.com/pkg/errors"

	"gitee.com/aniwar2/musae/logger"
)

func (s *BillServer) PayHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("PayHandler failed, err: ", err)
		}
	}()

	/*	kv := &state.KvTable{
			Key:     "test-hdy",
			Id:      1,
			UID:     "123",
			Data:    []byte("1"),
			UpSecTS: 0,
			InSecTS: 0,
			DataSrc: "1",
		}
		s.SaveMongoAndRedisDBByKvTable(service.MongoDbType_MongoGame, kv.Key, kv, nil)
	*/
	// IP校验
	logger.Infof("remote addr: %s", in.Request.RemoteAddr)
	if conf.Bill().IsIpWhite {
		ip, err := netutil.GetClientIP(in.Request)
		if err != nil {
			logger.Errorf(err.Error())
			return reply2Lilith(in, logic.FAIL), err
		}
		if !logic.CheckIp(conf.Bill().IpWhiteList, ip) {
			return reply2Lilith(in, logic.FAIL), errors.New("ip NOT in white list")
		}
	}

	if in == nil {
		err = errors.New("nil invocation parameter")
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}
	logger.Infof("[Bill] PayHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	// 参数
	argsMap := sdksign.ParseUrlArgs(string(in.Data))

	// 验签
	signSucc := sdksign.ParkSignVerify(argsMap, []string{"sign", "pay_env"})
	if !signSucc {
		err = errors.New(logic.Sign_Check_Error)
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	apiReq := com_order.ParseLilithPayCallbackReq(argsMap)
	if err != nil {
		err = errors.Wrap(err, "解析参数失败")
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}
	logger.Infof("充值回调参数:%+v", apiReq)

	// 校验应用id
	// if strconv.Itoa(int(apiReq.AppId)) != conf.SDK().LilithAppId {
	// 	err = errors.New(fmt.Sprintf("应用id不匹配, req.appId=%d, conf.AppId=%s", apiReq.AppId, conf.SDK().LilithAppId))
	// 	logger.Errorf(err.Error())
	// 	return reply2Lilith(in, logic.FAIL), err
	// }

	// 透传参数
	cbiObj, err := com_order.ParsePayCbi(apiReq.Ext)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("解析透传参数失败, ext:%s", apiReq.Ext))
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	// 校验玩家id
	if strconv.Itoa(int(apiReq.AppUid)) != cbiObj.AccountId {
		err = errors.New(fmt.Sprintf("玩家id不匹配, req.AppUid=%d, cbiObj.AccountId=%s", apiReq.AppUid, cbiObj.AccountId))
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	// 测试代码
	// payCbi := com_order.BuildPayCbi("myorderId", 1, 1)
	// apiReq.Ext = payCbi

	SdkParamBytes, err := json.Marshal(apiReq)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("json.Marshal got error"))
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	// 通知下发奖励
	actorData, err := proto.Marshal(&pb.S2S_BillCallbackReq{
		CpOrderId:   cbiObj.CpOrderId,
		SdkParamStr: string(SdkParamBytes),
	})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("proto.Marshal got error"))
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	// 转为自己的uid
	_, err = s.UserInvoke(cbiObj.Uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2S_BillCallbackReq),
		UserId:  cbiObj.Uaid,
		RoleId:  0,
		UAID:    cbiObj.Uaid,
		Data:    actorData,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	})
	if err != nil {
		err = errors.Wrap(err, "通知下发奖励报错")
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	// // 已下发奖励
	// dbOrder.OrderStatus = pb.OrderStatus_OrderStatus_reward
	// s.SaveMongoAndRedisDB(dbMongoType, dbKey, dbOrders, nil)

	// 成功通知
	// out.Data = []byte(logic.SUCCESS) // return success
	// return out, nil
	return reply2Lilith(in, logic.SUCCESS), err
}

func reply2Lilith(req *common.InvocationEvent, msg string) *common.Content {
	out := &common.Content{
		ContentType: req.ContentType,
		DataTypeURL: req.DataTypeURL,
		Data:        []byte(msg),
	}

	return out
}

// // 解析参数
// func parseLilithPayCallbackReq(argsMap map[string]interface{}) *LilithPayCallbackReq {
//	apiReq := &LilithPayCallbackReq{
//		Serial:       getInt32(argsMap["serial"]),
//		PayType:      getInt32(argsMap["pay_type"]),
//		AppId:        getInt32(argsMap["app_id"]),
//		AppUid:       getInt32(argsMap["app_uid"]),
//		Ext:          getString(argsMap["ext"]),
//		Amount:       getInt32(argsMap["amount"]),
//		ProductType:  getString(argsMap["product_type"]),
//		AdditionInfo: getString(argsMap["addition_info"]),
//		PayEnv:       getInt32(argsMap["pay_env"]),
//		Sign:         getString(argsMap["sign"]),
//	}
//
//	return apiReq
// }

// // LilithPayCallbackReq Lilith支付回调参数
// type LilithPayCallbackReq struct {
//	Serial       int32  `json:"serial"`        // 订单号
//	PayType      int32  `json:"pay_type"`      // 支付平台类型(iOS/Alipay/...)
//	AppId        int32  `json:"app_id"`        // app id
//	AppUid       int32  `json:"app_uid"`       // 用户平台id
//	Ext          string `json:"ext"`           // CP自定义数据(长度限制255)
//	Amount       int32  `json:"amount"`        // 支付金额（单位为分）
//	ProductType  string `json:"product_type"`  // 商品ID
//	AdditionInfo string `json:"addition_info"` // 附加参数,可能为空 (该附加信息为一些支付类型特有数据的集合。数据格式为json，需要先base64解码。 只有在有附加信息的时候，该参数才会下发并则参与签名)
//	PayEnv       int32  `json:"pay_env"`       // 支付环境
//	Sign         string `json:"sign"`          // 签名
//	//Cbi          *com_order.CbiObj `json:"com_order"`           // 透传解析结构
// }

// // 解析参数
// func parseLilithRefundReq(argsMap map[string]interface{}) *LilithRefundReq {
//	apiReq := &LilithRefundReq{
//		Serial: getInt32(argsMap["serial"]),
//		AppId:  getInt32(argsMap["app_id"]),
//		AppUid: getInt32(argsMap["app_uid"]),
//		Ext:    getString(argsMap["ext"]),
//		Sign:   getString(argsMap["sign"]),
//	}
//
//	return apiReq
// }
//
// // LilithRefundReq Lilith退款回调参数
// type LilithRefundReq struct {
//	Serial int32  `json:"serial"`  // 订单号
//	AppId  int32  `json:"app_id"`  // app id
//	AppUid int32  `json:"app_uid"` // 用户平台id TODO 需确认该字段是否有？文档中不存在
//	Ext    string `json:"ext"`     // CP自定义数据(长度限制255)
//	Sign   string `json:"sign"`    // 签名
// }

func genSignStr(reqStr string, excludeSignKeys []string, reqObj any) string {
	// // sign和pay_env不参与验签
	// signData := lo.OmitByKeys[string, []string](params, []string{"sign", "pay_env"})
	//
	// // 按key的字典序取值
	// keys := lo.Keys(signData)
	// sort.Strings(keys)
	// strSlice := make([]string, 0, len(keys))
	// for _, k := range keys {
	//	value := ""
	//	if len(signData[k]) > 0 {
	//		value = signData[k][0]
	//	}
	//	strSlice = append(strSlice, fmt.Sprintf("%s=%s", k, value))
	// }
	//
	// return strings.Join(strSlice[:], "&")
	if excludeSignKeys == nil {
		excludeSignKeys = make([]string, 0)
	}

	// //默认sign参数不参与前面
	// signKey := "sign"
	// if !gameUtils.ArrayContain(excludeSignKeys, signKey) {
	//	excludeSignKeys = append(excludeSignKeys, signKey)
	// }

	// url decode
	unescape, err := url.QueryUnescape(reqStr)
	if err != nil {
		logger.Errorf(err.Error())
		return ""
	}

	// base64 decode
	decodeBytes, err := base64.StdEncoding.DecodeString(unescape)
	if err != nil {
		logger.Errorf(err.Error())
		return ""
	}

	// json解析
	// m := make(map[string]string)
	err = json.Unmarshal(decodeBytes, reqObj)
	// err = json.Unmarshal([]byte(unescape), &m)
	if err != nil {
		logger.Errorf(err.Error())
		return ""
	}

	return string(decodeBytes)
}

// func getInt32(i interface{}) int32 {
//	s, ok := i.(string)
//	if !ok {
//		return 0
//	}
//
//	atoi, err := strconv.Atoi(s)
//	if err != nil {
//		return 0
//	}
//
//	return int32(atoi)
// }
//
// func getString(i interface{}) string {
//	s, ok := i.(string)
//	if !ok {
//		return ""
//	}
//
//	return s
// }

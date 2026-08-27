package logic

import (
	"context"
	"fmt"
	"github.com/yunjoy-tech/aniwar/src/common/sdkconstant/sdksign"
	"github.com/yunjoy-tech/musae/gamelib/guid"
	netutil "github.com/yunjoy-tech/musae/utils/net"

	"github.com/dapr/go-sdk/service/common"
	gameCommon "github.com/yunjoy-tech/aniwar/src/common"

	"github.com/yunjoy-tech/musae/base"
	"google.golang.org/protobuf/proto"

	"github.com/yunjoy-tech/aniwar/src/proto/pb"

	"github.com/yunjoy-tech/aniwar/src/common/com_order"
	"github.com/yunjoy-tech/aniwar/src/common/conf"

	"github.com/pkg/errors"
	"github.com/yunjoy-tech/musae/logger"

	"github.com/yunjoy-tech/aniwar/src/idipserver/logic"
)

func (s *BillServer) RefundHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("PayHandler failed, err: ", err)
		}
	}()

	// IP校验
	logger.Debugf("remote addr: %s", in.Request.RemoteAddr)
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
	logger.Debugf("[Bill] RefundHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	// 参数
	argsMap := sdksign.ParseUrlArgs(string(in.Data))

	// 验签
	signSucc := sdksign.ParkSignVerify(argsMap, []string{"sign", "pay_env"})
	if !signSucc {
		err = errors.New(logic.Sign_Check_Error)
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	apiReq := com_order.ParseLilithRefundReq(argsMap)
	if err != nil {
		err = errors.Wrap(err, "解析参数失败")
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}
	logger.Infof("退单回调参数:%+v", apiReq)

	// if strconv.Itoa(int(apiReq.AppId)) != conf.SDK().LilithAppId {
	// 	err = errors.New(fmt.Sprintf("应用id不匹配, req.appId=%d, conf.AppId=%s", apiReq.AppId, conf.SDK().LilithAppId))
	// 	logger.Errorf(err.Error())
	// 	return reply2Lilith(in, logic.FAIL), err
	// }

	// 透传参数
	// cbiObj, err := com_order.ParsePayCbi(apiReq.Ext)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("解析透传参数失败, ext:%s", apiReq.Ext))
		logger.Errorf(err.Error())
		return reply2Lilith(in, logic.FAIL), err
	}

	// shopGiftCfg := excel.GetShopGiftMgr().GetById(cbiObj.PayId)
	// if shopGiftCfg == nil {
	// 	err = errors.New(fmt.Sprintf("无效的支付id, payId=%d", cbiObj.PayId))
	// 	logger.Errorf(err.Error())
	// 	return reply2Lilith(in, logic.FAIL), err
	// }

	// 转为自己的uid
	// myUid := sdkconstant.GenLilithUid(int(apiReq.AppUid))
	myUid := ""
	// 订单信息db配置
	dbMongoType, dbKey := com_order.OrderDBTable(myUid)

	// 查找订单
	dbOrders := &pb.OrderData{}
	_, err = s.LoadMongoDB(dbMongoType, dbKey, dbOrders)

	refundData := dbOrders.RefundData

	// 累计退款次数
	refundData.RefundCount += 1
	// refundData.RefundAmount += shopGiftCfg.PayCount

	// 持久化
	s.SaveMongoAndRedisDB(dbMongoType, dbKey, dbOrders, nil)

	// 规则：退款2次或者退款金额达到648, 进行封号处理
	if refundData.RefundCount >= 2 || refundData.RefundAmount >= 648*100 /*单位:分*/ {
		// 封号
		// 通知下发奖励
		actorData, err := proto.Marshal(&pb.S2AS_ExcuteGMReq{
			CmdName: gameCommon.GM_BANNED,                         // 封禁
			OptVal:  fmt.Sprintf("%v %s", 365*24*3600*10, "异常退款"), // 封禁时长(秒)+封禁原因
		})
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("proto.Marshal got error"))
			logger.Errorf(err.Error())
			return reply2Lilith(in, logic.FAIL), err
		}

		_, err = s.UserInvoke(myUid, &base.ProtoMsg{
			AppId:   s.AppId,
			MsgId:   int32(pb.Protocols_PS2AS_GmExecuteReq),
			UserId:  myUid,
			RoleId:  0,
			UAID:    myUid,
			Data:    actorData,
			ErrCode: 0,
			// GUID:    utils.GenIntUUID(),
			ServerReqIdx: guid.GenIntUuid(),
			Topic:        "",
		})
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("对:%s, 进行封号遇到错误", myUid))
			logger.Errorf(err.Error())
			return reply2Lilith(in, logic.FAIL), err
		}
	}

	return reply2Lilith(in, logic.SUCCESS), nil
}

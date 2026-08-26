package useractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yunjoy-tech/aniwar/src/common"
	"github.com/yunjoy-tech/aniwar/src/common/com_order"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/datalog/taptap"
	"github.com/yunjoy-tech/aniwar/src/common/sdkconstant/sdkutil"
	"github.com/yunjoy-tech/aniwar/src/meta"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/gamelib/guid"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/service"
	"github.com/yunjoy-tech/musae/utils"
	timeutil "github.com/yunjoy-tech/musae/utils/time"
	"google.golang.org/protobuf/proto"
	"strconv"
)

type OrderHandler struct {
	*UABaseHandler
}

func NewOrderHandler(actor *UserActor) *OrderHandler {
	h := &OrderHandler{UABaseHandler: NewUABaseHandler(actor, "OrderHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CreateOrderReq), h.CreateOrderReq)          // 创建订单 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_BillCallbackReq), h.BillCallbackReq)         // bill服务通知下发奖励
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CheckOrderReq), h.CheckOrderReq)            // 检查订单 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_OrderListReq), h.OrderListReq)               // 查看订单列表(角色id)
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_ReplacementOrderReq), h.ReplacementOrderReq) // 补单

	return h
}

// Init 初始化模块数据
func (h *OrderHandler) Init() error {
	// 初始化
	h.actor.OrderData = h.actor.GetOrderData()
	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	logger.Debug("init order data success. player: %s", h.actor.ID())
	return nil
}

func (h *OrderHandler) EnterGame() error {
	return nil
}

func (h *OrderHandler) DailyRefresh() error {
	h.trySendMonthMail(0)
	return nil
}

func (h *OrderHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.OrderData); ok {
		h.actor.OrderData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *OrderHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	dbTable, dbKey := com_order.OrderDBTable(h.actor.GetUID())
	return dbTable, dbKey, h.actor.OrderData
	// return service.MongoDbType_MongoAccount, db.KeyUserOrderInfo(h.actor.GetUID()), h.actor.OrderDatas
}

func (h *OrderHandler) CreateOrderReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
		now = time.Now()
	)

	if conf.Bill().CanPay == 0 {
		// 充值关闭
		return nil, errors.New("充值功能关闭"), int32(pb.ErrorCode_Order_pay_closed)
	}

	var req pb.C2LS_CreateOrderReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	var productCfg *meta.GameShopPkgGiftMeta
	// productCfg := excel.GetShopGiftMgr().GetById(req.ProductId)
	if productCfg == nil {
		return nil, fmt.Errorf("无效的ProductId=%d", req.ProductId), int32(pb.ErrorCode_NotFoundConfig)
	}

	// rechargeCfg := excel.GetShopRechargeMgr().GetById(productCfg.)
	// if rechargeCfg == nil {
	//	return nil, fmt.Errorf("无效的payId=%d", productCfg.RechargeId), int32(pb.ErrorCode_NotFoundConfig)
	// }

	// 月卡商品检查
	err, code := h.checkCanBuy(req.ProductId)
	if err != nil {
		return nil, err, int32(code)
	}

	// tap检查支付是否受限
	if h.actor.Srv.IsTapChannel(h.actor.Account.Account.Channel) {
		userSession, err, errCode := h.actor.Srv.GetUserSession(h.actor.uid)
		if err != nil {
			return nil, err, int32(errCode)
		}

		if ok, errCode := sdkutil.TapCheckPayLimit(h.actor.Account.TapUserInfo, userSession.TaptapToken, userSession.TaptapOpenId, productCfg.PayCount); !ok {
			return nil, errors.New("tap限制消费"), int32(errCode)
		}
	} else {
		h.Debugf("不是tap用户, 不做tap消费限制检查")
	}

	orderId := guid.GenStrUuid()
	order := &pb.Order{
		CreateTs:      now.Unix(),
		UpdateTs:      now.Unix(),
		Uid:           h.actor.uid,
		RoleId:        h.actor.roleId,
		CpOrderId:     orderId,
		OrderStatus:   pb.OrderStatus_OrderStatus_default,
		CpProductId:   req.ProductId,
		PayId:         productCfg.Id,
		ProductType:   productCfg.ProductType,
		RechargePrice: strconv.Itoa(int(productCfg.PayCount)),
		PaymentTs:     0,
		PaymentType:   pb.PaymentType_PaymentType_min,
		CpCbi:         com_order.BuildPayCbi(h.actor.Account.Account.Uid, h.actor.ID(), orderId, productCfg.Id),
		ApiParams:     "",
	}

	orderData := h.actor.GetOrderData()
	// orderDatas.OrderList = append(orderDatas.OrderList, order)
	orderData.Orders[order.CpOrderId] = order
	err = h.SaveDB(true)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_CreateOrderRes{
		CpOrderId: order.CpOrderId,
		Cbi:       order.CpCbi,
	}

	// 模拟充值
	h.VirtualPay(order.CpOrderId)

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *OrderHandler) BillCallbackReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.S2S_BillCallbackReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	sdkReq := &com_order.LilithPayCallbackReq{}
	err = json.Unmarshal([]byte(req.SdkParamStr), sdkReq)
	if err != nil {
		err = fmt.Errorf("解析sdk参数失败, ext:%s: %w", req.SdkParamStr, err)
		logger.Errorf(err.Error())
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 透传参数
	cbiObj, err := com_order.ParsePayCbi(sdkReq.Ext)
	if err != nil {
		err = fmt.Errorf("解析透传参数失败, ext:%s: %w", sdkReq.Ext, err)
		logger.Errorf(err.Error())
		return nil, errors.New(fmt.Sprintf("解析透传参数失败:%s, %s", req.CpOrderId, sdkReq.Ext)), int32(pb.ErrorCode_Order_cbi_invalid)
	}
	var shopGiftCfg *meta.GameShopPkgGiftMeta
	// shopGiftCfg := excel.GetShopGiftMgr().GetById(cbiObj.PayId)
	if shopGiftCfg == nil {
		err = errors.New(fmt.Sprintf("无效的支付id, payId=%d", cbiObj.PayId))
		logger.Errorf(err.Error())
		return nil, err, int32(pb.ErrorCode_Invalid_order)
	}

	if sdkReq.ProductType != "" /*国内sdk的没有数据*/ && shopGiftCfg.ProductType != sdkReq.ProductType {
		err = errors.New(fmt.Sprintf("无效的商品ID, %s", sdkReq.ProductType))
		logger.Errorf(err.Error())
		return nil, err, int32(pb.ErrorCode_Invalid_order)
	}

	if shopGiftCfg.PayCount != sdkReq.Amount { // 单位分
		err = errors.New(fmt.Sprintf("商品金额错误, payId=%d, %d, %d", cbiObj.PayId, shopGiftCfg.Price, sdkReq.Amount))
		logger.Errorf(err.Error())
		return nil, err, int32(pb.ErrorCode_Invalid_order)
	}

	// // 订单信息db配置
	// dbMongoType, dbKey := com_order.OrderDBTable(cbiObj.AccountId)
	// // 查找订单
	// orders := &pb.OrderData{}
	// _, err = s.LoadMongoDB(dbMongoType, dbKey, orders)
	// if err != nil {
	//	err = fmt.Errorf(": %w", err)fmt.Sprintf("没有查找到订单信息, dbMongoType:%s, dbKey:%s", dbMongoType, dbKey))
	//	logger.Errorf(err.Error())
	//	return nil, err, int32(pb.ErrorCode_Invalid_order)
	// }

	order, err, errCode := h.GetOrderById(req.CpOrderId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	// 接口参数
	order.ApiParams = req.SdkParamStr

	// 下发奖励
	err, errCode = h.paySendReward(order, pb.PaymentType_PaymentType_sdk_callback, strconv.Itoa(int(sdkReq.PayType)))
	if err != nil {
		return nil, err, int32(errCode)
	}

	rsp := &pb.S2S_BillCallbackRes{}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *OrderHandler) CheckOrderReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.C2LS_CheckOrderReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// orderDatas := h.actor.GetOrderData()

	// var order *pb.Order
	// if _order, ok := orderDatas.Orders[req.CpOrderId]; ok {
	//	order = _order
	// } else {
	//	return nil, errors.New(fmt.Sprintf("无效的订单:%s", req.CpOrderId)), int32(pb.ErrorCode_Invalid_order)
	// }
	order, err, errCode := h.GetOrderById(req.CpOrderId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	if order.OrderStatus != pb.OrderStatus_OrderStatus_reward &&
		order.OrderStatus != pb.OrderStatus_OrderStatus_show {

		return nil, errors.New(fmt.Sprintf("订单还未支付完成, %s", req.CpOrderId)), int32(pb.ErrorCode_Order_do_NOT_pay)
	}

	order.OrderStatus = pb.OrderStatus_OrderStatus_show
	err = h.SaveDB(true)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_CheckOrderRes{
		OrderStatus: order.OrderStatus,
		DropChange:  order.DropChange,
		CommonData:  h.actor.comData.FixDownComData(),
		ItemInfo:    h.getOrderItemById(order.CpProductId),
	}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *OrderHandler) OrderListReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.S2S_OrderListReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	orderList := h.getOrderList(req.OrderStatus)

	rsp := &pb.S2S_OrderListRes{
		Orders: orderList,
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *OrderHandler) ReplacementOrderReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.S2S_ReplacementOrderReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	order, err, errCode := h.GetOrderById(req.OrderId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	// 下发奖励
	err, errCode = h.paySendReward(order, pb.PaymentType_PaymentType_replacement, "")
	if err != nil {
		return nil, err, int32(errCode)
	}

	rsp := &pb.S2S_ReplacementOrderRes{
		// Orders: orderList,
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *OrderHandler) GetOrderById(orderId string) (*pb.Order, error, pb.ErrorCode) {
	orderDatas := h.actor.GetOrderData()

	var order *pb.Order
	if _order, ok := orderDatas.Orders[orderId]; ok {
		order = _order
	} else {
		return nil, errors.New(fmt.Sprintf("无效的订单:%s", orderId)), pb.ErrorCode_Invalid_order
	}

	return order, nil, pb.ErrorCode_Success
}

// 是否是第一次购买
func (h *OrderHandler) isFirstPay(cpProductId int32) bool {
	orderData := h.actor.GetOrderData()
	_, ok := orderData.HistoryProducts[cpProductId]
	return !ok
}

// 获取历史购买次数
func (h *OrderHandler) incrPayCount(cpProductId int32) {
	orderData := h.actor.GetOrderData()
	if oldCount, ok := orderData.HistoryProducts[cpProductId]; ok {
		orderData.HistoryProducts[cpProductId] = oldCount + 1
	} else {
		orderData.HistoryProducts[cpProductId] = 1
	}
}

func (h *OrderHandler) getOrderList(orderStatus pb.OrderStatus) []*pb.Order {
	orderDatas := h.actor.GetOrderData()

	orderList := make([]*pb.Order, 0)
	for _, order := range orderDatas.Orders {

		if orderStatus == pb.OrderStatus_OrderStatus_Max {
			// 全量返回
			orderList = append(orderList, order)
		} else if order.OrderStatus == orderStatus {
			// 根据订单状态返回
			orderList = append(orderList, order)
		}
	}

	return orderList
}

// 支付下发奖励
func (h *OrderHandler) paySendReward(order *pb.Order, paymentType pb.PaymentType, payType string) (error, pb.ErrorCode) {
	var (
		err error
	)

	if order.OrderStatus != pb.OrderStatus_OrderStatus_default && order.OrderStatus != pb.OrderStatus_OrderStatus_payment {
		return errors.New(fmt.Sprintf("订单已经处理过了:%v", order.OrderStatus)), pb.ErrorCode_Order_status_unusual
	}

	var shopGiftCfg *meta.GameShopPkgGiftMeta
	// shopGiftCfg := excel.GetShopGiftMgr().GetById(order.CpProductId)

	// 累计充值金额
	h.actor.OrderData.TotalRecharge += shopGiftCfg.PayCount
	// 已支付
	order.PaymentTs = time.Now().Unix()
	order.OrderStatus = pb.OrderStatus_OrderStatus_payment
	order.PaymentType = paymentType
	err = h.SaveDB(true)
	if err != nil {
		return err, pb.ErrorCode_SaveDBError
	}
	h.Infof("充值成功, %+v", order)

	var giftCfg *meta.GameShopPkgGiftMeta
	// giftCfg := excel.GetShopGiftMgr().GetById(order.CpProductId)
	rewards := make([]*meta.ItemReward, 0)
	rewards = append(rewards, giftCfg.Reward...)

	// 首次购买
	if h.isFirstPay(order.CpProductId) {
		rewards = append(rewards, giftCfg.RechargeFirstReward...)
	}
	// 月卡奖励
	if giftCfg.PeriodReward > 0 {
		var monthcardCfg *meta.GameShopPkgMonthCardMeta
		// monthcardCfg := excel.GetMonthcardMgr().GetById(giftCfg.PeriodReward)
		rewards = append(rewards, monthcardCfg.DirectReward...)
	}

	dropChange, err := GetDropMgr(h.actor).DropListByItems(rewards, true, nil, h.actor.comData, common.CR_RECHARGE)
	if err != nil {
		return err, pb.ErrorCode_InternalError
	}
	order.DropChange = dropChange

	// 记录购买次数
	h.incrPayCount(order.CpProductId)
	// 额外处理
	h.extraHandle(order.CpProductId)
	// 修改订单状态
	order.OrderStatus = pb.OrderStatus_OrderStatus_reward
	err = h.SaveDB(true)
	if err != nil {
		return err, pb.ErrorCode_SaveDBError
	}
	h.Infof("发放奖励成功, %+v", order)

	// 埋点
	// utils.SafeRunNoError(func() {
	//	lilith.WriteDataLog(&lilith.Purchase{
	//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_Purchase, h.actor.uid, &pb.CliDeviceInfo{}),
	//		RoleId:   h.actor.ID(),
	//		ItemId:   strconv.Itoa(int(order.CpProductId)),
	//		Level:    int32(h.actor.Data.Base.Common.RoleLevel),
	//		VipLevel: 0,
	//		IsTest:   0,
	//		OrderId:  order.CpOrderId,
	//		Recharge: float32(h.actor.OrderData.TotalRecharge),
	//		Currency: "",
	//		Price:    float32(shopGiftCfg.PayCount),
	//		Iap:      shopGiftCfg.ProductType,
	//		PayType:  payType,
	//	})
	// })
	utils.SafeRunNoError(func() {
		e := &taptap.Purchase{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			RoleId:            h.actor.ID(),
			ItemId:            strconv.Itoa(int(order.CpProductId)),
			Level:             int32(h.actor.Data.Base.Common.RoleLevel),
			VipLevel:          0,
			IsTest:            0,
			OrderId:           order.CpOrderId,
			Recharge:          float32(h.actor.OrderData.TotalRecharge),
			Currency:          "",
			Price:             float32(shopGiftCfg.PayCount),
			Iap:               shopGiftCfg.ProductType,
			PayType:           payType,
		}
		taptap.WriteDataLog(taptap.LogType_Purchase, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	if h.actor.Srv.IsTapChannel(h.actor.Account.Account.Channel) {
		userSession, err, errCode := h.actor.Srv.GetUserSession(h.actor.uid)
		if err != nil {
			return nil, errCode
		}

		if ok := sdkutil.TapUploadPayAmount(userSession.TaptapToken, userSession.TaptapOpenId, shopGiftCfg.PayCount); !ok {
			h.Debugf("tap上报充值金额异常")
		}
	} else {
		h.Debugf("不是tap用户, 充值金额不做上报")
	}

	return nil, pb.ErrorCode_Success
}

// VirtualPay 虚拟充值
func (h *OrderHandler) VirtualPay(cpOrderId string) {
	if conf.Bill().CanVirtualPay != 1 {
		h.Debugf("不支持模拟充值, CanVirtualPay:%d", conf.Bill().CanVirtualPay)
		return
	}

	channel := h.actor.Account.Account.Channel
	if !h.actor.Srv.IsPCChannel(channel) && !h.actor.Srv.IsTapChannel(channel) {
		// 不是pc, 不是tap用户, 则不支持模拟充值
		h.Debugf("不是tap用户, 不支持模拟充值")
		return
	}

	order, err, _ := h.GetOrderById(cpOrderId)
	if err != nil {
		logger.Errorf(err.Error())
		return
	}

	err, errCode := h.paySendReward(order, pb.PaymentType_PaymentType_virtual_pay, "")
	if err != nil {
		err = fmt.Errorf("errCode:%v: %w", errCode, err)
		h.Debugf(err.Error())
		return
	}
}

// 构建订单相关的数据
func (h *OrderHandler) buildOrderInfo() *pb.OrderInfo {
	info := &pb.OrderInfo{}
	info.HistoryProducts = h.GetHistoryProducts()
	info.OrderItems = h.GetOrderItemInfo()

	return info
}

func (h *OrderHandler) GetHistoryProducts() []*pb.HistoryProduct {
	var (
		historyProducts = make([]*pb.HistoryProduct, 0)
	)

	for productId, count := range h.actor.OrderData.HistoryProducts {
		historyProducts = append(historyProducts, &pb.HistoryProduct{
			ProductId: productId,
			Count:     count,
		})
	}

	return historyProducts
}

func (h *OrderHandler) GetOrderItemInfo() []*pb.OrderItemInfo {
	ret := make([]*pb.OrderItemInfo, 0)
	for _, items := range h.actor.OrderData.ItemInfo {
		if fixMonthExpire(items) {
			continue
		}
		ret = append(ret, items)
	}
	return ret
}

func (h *OrderHandler) getOrderItemById(productId int32) *pb.OrderItemInfo {
	orderData := h.actor.GetOrderData()
	return orderData.ItemInfo[productId]
}

func (h *OrderHandler) extraHandle(productId int32) {
	var cfg *meta.GameShopPkgGiftMeta
	// cfg := excel.GetShopGiftMgr().GetById(productId)
	orderData := h.actor.GetOrderData()
	info := orderData.ItemInfo[productId]
	if info == nil {
		info = &pb.OrderItemInfo{ProductId: productId}
		orderData.ItemInfo[productId] = info
	}

	now := time.Now().Unix()

	// 月卡类型处理
	if cfg.PeriodReward > 0 {
		var f bool
		if info.MonthExpire < now {
			f = true
			info.MonthExpire = now
		}
		// 记录月卡数据
		var monthcardCfg *meta.GameShopPkgMonthCardMeta
		// monthcardCfg := excel.GetMonthcardMgr().GetById(cfg.PeriodReward)
		target := time.Unix(info.MonthExpire, 0).AddDate(0, 0, int(monthcardCfg.Day))
		info.MonthExpire = timeutil.NextNDayResetTime(target, 0).Unix()
		info.MonthNodeExpire = append(info.MonthNodeExpire, info.MonthExpire)

		// 每日奖励邮件
		if f {
			h.trySendMonthMail(productId)
		}
	}

	// 其他类型的额外处理...
}

// 订单商品是否能购买
func (h *OrderHandler) checkCanBuy(productId int32) (error, pb.ErrorCode) {
	orderData := h.actor.GetOrderData()
	var cfg *meta.GameShopPkgGiftMeta
	// cfg := excel.GetShopGiftMgr().GetById(productId)
	if cfg == nil {
		return fmt.Errorf("config not found %v", productId), pb.ErrorCode_NotFoundConfig
	}

	// 月卡类型检查
	if cfg.PeriodReward > 0 {
		var monthcardCfg *meta.GameShopPkgMonthCardMeta
		// monthcardCfg := excel.GetMonthcardMgr().GetById(cfg.PeriodReward)
		if monthcardCfg == nil {
			return fmt.Errorf("config not found %v", productId), pb.ErrorCode_NotFoundConfig
		}
		item := orderData.ItemInfo[productId]
		if item != nil {
			limit := time.Now().AddDate(0, 0, int(monthcardCfg.DayLimit))
			target := time.Unix(item.MonthExpire, 0).AddDate(0, 0, int(monthcardCfg.Day))
			if limit.Before(target) {
				return fmt.Errorf("product buy limit"), pb.ErrorCode_Shop_goods_buy_count_limit
			}
		}
	}

	// 其他类型检查...

	return nil, 0
}

// 尝试发送月卡奖励邮件
func (h *OrderHandler) trySendMonthMail(productId int32) {
	orderData := h.actor.GetOrderData()

	// 筛选处理数据
	itemInfos := make(map[int32]*pb.OrderItemInfo)
	if productId == 0 {
		itemInfos = orderData.ItemInfo
	} else {
		itemInfos[productId] = orderData.ItemInfo[productId]
	}

	// 尝试发奖励
	for _, info := range itemInfos {
		var cfg *meta.GameShopPkgGiftMeta
		// cfg := excel.GetShopGiftMgr().GetById(id)
		if cfg == nil {
			continue
		}

		// 不是月卡
		if cfg.PeriodReward <= 0 {
			continue
		}

		if fixMonthExpire(info) {
			continue
		}

		var monthcardCfg *meta.GameShopPkgMonthCardMeta
		// monthcardCfg := excel.GetMonthcardMgr().GetById(cfg.PeriodReward)
		if monthcardCfg == nil {
			continue
		}
		// 发奖励邮件
		// attachment := datahelper.ConvertItem2ByTpl(monthcardCfg.DailyReward)
		h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_6, nil, h.actor.comData)
	}
	h.SaveDB()
}

// 修正月卡过期时间数据，返回是否过期的标记，过期返回true
func fixMonthExpire(info *pb.OrderItemInfo) bool {
	now := time.Now().Unix()
	var f bool
	// 维护一下每期月卡到期时间记录
	for _, ts := range info.MonthNodeExpire {
		if ts < now {
			info.MonthNodeExpire = info.MonthNodeExpire[1:]
		}
	}

	// 过期了
	if info.MonthExpire < now {
		info.MonthExpire = 0
		f = true
	}
	return f
}

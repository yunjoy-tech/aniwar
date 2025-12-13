package useractor

import (
	"context"
	"fmt"
	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"
	"math"
	"strconv"
	"time"

	"gitee.com/bychannel/musae/framework/threading"

	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/common/clidto"
	"gitee.com/bychannel/aniwar/src/common/db"
	"gitee.com/bychannel/aniwar/src/common/utils"
	excel "gitee.com/bychannel/aniwar/src/excel/data"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/base"
	"gitee.com/bychannel/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type CurrencyHandler struct {
	*UABaseHandler
}

func NewCurrencyHandler(actor *UserActor) *CurrencyHandler {
	h := &CurrencyHandler{UABaseHandler: NewUABaseHandler(actor, "CurrencyHandler")}
	h.ChildHandler = h

	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CurrencyExchangeReq), h.CurrencyExchangeReq)
	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CurrencyBuyReq), h.CurrencyBuyReq)

	return h
}

// Init 初始化模块数据
func (h *CurrencyHandler) Init() error {
	// 初始数据
	currencyMap := make(map[int32]*pb.CurrencyItem)
	currencyMap[common.CURRENCY_ITEM_ID_2010] = &pb.CurrencyItem{
		Key:   common.CURRENCY_ITEM_ID_2010,
		Value: 999,
	}
	currencyMap[common.CURRENCY_ITEM_ID_2011] = &pb.CurrencyItem{
		Key:   common.CURRENCY_ITEM_ID_2011,
		Value: 999,
	}

	h.actor.Data.Currency = &pb.PCurrencyInfo{
		Createtime: time.Now().Unix(),
		Currencyx:  currencyMap,
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init currency data success.")
	return nil
}

func (h *CurrencyHandler) EnterGame() error {
	return nil
}

func (h *CurrencyHandler) DailyRefresh() error {
	return nil
}

func (h *CurrencyHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PCurrencyInfo); ok {
		h.actor.Data.Currency = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *CurrencyHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCurrency(h.actor.ID()), h.actor.Data.Currency
}

func (h *CurrencyHandler) buildCurrencyList() []*pb.CurrencyItem {
	items := make([]*pb.CurrencyItem, 0)
	for _, v := range h.actor.GetCurrencyData().Currencyx {
		data := &pb.CurrencyItem{
			Key:   v.GetKey(),
			Value: v.GetValue(),
		}
		items = append(items, data)
	}
	return items
}

func (h *CurrencyHandler) CurrencyExchangeReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_CurrencyExchangeReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 是否可以兑换
	cfg := excel.GetCoinageMgr().GetById(req.CurrencyType)
	if cfg == nil {
		return nil, fmt.Errorf("currency unsupport exchange %d", req.CurrencyType), int32(pb.ErrorCode_CurrencyUnsupportExchange)
	}

	// 道具检查
	costs := utils.ConvertItem4(req.GetCosts())
	sum, errorCode := h.exchangeCheck(cfg, costs)
	if errorCode != pb.ErrorCode_Success {
		return nil, fmt.Errorf("param check failed"), int32(errorCode)
	}

	item, err := h.GetValue(req.CurrencyType)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 硬上限
	if sum+item.GetValue() > getHardLimit(req.CurrencyType) {
		return nil, fmt.Errorf("currency up to hard limit"), int32(pb.ErrorCode_CurrencyUpToLimit)
	}

	// 扣道具
	err = GetConsumeMgr(h.actor).ConsumeListByUniqueId(costs, h.actor.comData, common.CR_Currency_Exchange)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 加货币值
	_, err = GetDropMgr(h.actor).DropList2(map[int32]int32{req.CurrencyType: int32(sum)}, true, nil, h.actor.comData, common.CR_Currency_Exchange)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 埋点log
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CurrencyExchange{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CurrencyExchange, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		MoneyType:      req.CurrencyType,
	//		Cost:           lilith.ConvertMap2Str(costs),
	//		ExchangeNum:    uint64(sum),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.CurrencyExchange{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			MoneyType:         req.CurrencyType,
			Cost:              taptap.ConvertMap2Str(costs),
			ExchangeNum:       uint64(sum),
		}
		taptap.WriteDataLog(taptap.LogType_CurrencyExchange, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &pb.LS2C_CurrencyExchangeRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *CurrencyHandler) exchangeCheck(cfg *excel.CoinageCfg, costs map[uint64]uint32) (int64, pb.ErrorCode) {
	sum := int64(0)
	// 是否可用道具
	for uniqueId, num := range costs {
		f := true
		item := h.actor.BagHandler.GetItemByUniqueId(uniqueId)
		if item == nil {
			return 0, pb.ErrorCode_ParamError
		}

		for _, keyVal := range cfg.BuyEffect {
			if keyVal.Key == int32(item.BaseId) {
				f = false
				sum += int64(uint32(keyVal.Val) * num)
			}
		}
		if f {
			return 0, pb.ErrorCode_ParamError
		}
	}

	// 道具是否足够
	if !GetConsumeMgr(h.actor).CheckMapEnoughByUniqueId(costs) {
		return 0, pb.ErrorCode_NotEnoughItem
	}

	return sum, pb.ErrorCode_Success
}

func (h *CurrencyHandler) CurrencyBuyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_CurrencyBuyReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 是否可以兑换
	cfg := excel.GetCoinageMgr().GetById(req.CurrencyType)
	if cfg == nil {
		return nil, fmt.Errorf("currency unsupport exchange %d", req.CurrencyType), int32(pb.ErrorCode_CurrencyUnsupportExchange)
	}

	// 货币检查
	itemCfg := excel.GetItemMgr().GetById(cfg.CurrencyExchange.GetKey())
	if itemCfg == nil {
		return nil, fmt.Errorf("item config not found %d", 0), int32(pb.ErrorCode_NotFoundConfig)
	}
	if !h.CheckEnough(itemCfg.ItemId, int64(req.GetNum())) {
		return nil, fmt.Errorf("currency not enough"), int32(pb.ErrorCode_CurrencyNotEnough)
	}
	sum := cfg.CurrencyExchange.GetVal() * req.GetNum()

	item, err := h.GetValue(req.CurrencyType)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	if int64(sum)+item.GetValue() > getHardLimit(req.CurrencyType) {
		return nil, fmt.Errorf("currency up to hard limit"), int32(pb.ErrorCode_CurrencyUpToLimit)
	}

	// 扣货币
	err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{itemCfg.ItemId: req.GetNum()}, h.actor.comData, common.CR_Currency_Exchange)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 加货币
	_, err = GetDropMgr(h.actor).DropList2(map[int32]int32{req.CurrencyType: sum}, true, nil, h.actor.comData, common.CR_Currency_Exchange)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 埋点log
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CurrencyBuy{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CurrencyBuy, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		MoneyType:      req.CurrencyType,
	//		ExchangeNum:    int32(sum),
	//		CostType:       itemCfg.SubType,
	//		CostNum:        req.Num,
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.CurrencyBuy{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			MoneyType:         req.CurrencyType,
			ExchangeNum:       int32(sum),
			CostType:          itemCfg.SubType,
			CostNum:           req.Num,
		}
		taptap.WriteDataLog(taptap.LogType_CurrencyBuy, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 消息返回
	rsp := &pb.LS2C_CurrencyBuyRes{CommonData: h.actor.comData.FixDownComData()}
	return rsp, nil, 0
}

// AddValue 增加value的公共方法
func (h *CurrencyHandler) AddValue(typ int32, value int64, commonData *clidto.Comdata, reason common.ChangeReason) error {
	var (
		beforeNum, afterNum int64
	)
	// 增加
	item, err := h.GetValue(typ)
	if err != nil {
		return err
	}
	beforeNum = item.GetValue()
	afterNum = beforeNum + value
	item.Value = afterNum

	h.actor.GetCurrencyData().Currencyx[typ] = item
	commonData.Data.Currency = append(commonData.Data.Currency, item)
	// 保存
	if err = h.SaveDB(); err != nil {
		return err
	}

	// 一级货币才输出
	if typ == common.CURRENCY_ITEM_ID_2006 {
		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.MoneyFlow{
		//		HeadInfo:    lilith.BuildHeadInfo(lilith.LogType_MoneyFlow, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		RoleId:      h.actor.ID(),
		//		MoneyBefore: beforeNum,
		//		MoneyAfter:  afterNum,
		//		Flow:        "in",
		//		Action:      strconv.Itoa(int(reason)),
		//		Level:       int32(h.actor.LoginHandler.getRoleLevel()),
		//		VipLevel:    0,
		//		MoneyType:   strconv.Itoa(int(typ)),
		//		Item:        "",
		//		Recharge:    0,
		//	})
		// })
		threading.RunSafe(func() {
			e := &taptap.MoneyFlow{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            h.actor.ID(),
				MoneyBefore:       beforeNum,
				MoneyAfter:        afterNum,
				Flow:              "in",
				Action:            strconv.Itoa(int(reason)),
				Level:             int32(h.actor.LoginHandler.getRoleLevel()),
				VipLevel:          0,
				MoneyType:         strconv.Itoa(int(typ)),
				Item:              "",
				Recharge:          0,
			}
			taptap.WriteDataLog(taptap.LogType_MoneyFlow, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	} else {
		// 其他货币输出
		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.ResourceFlow{
		//		HeadInfo:       lilith.BuildHeadInfo(lilith.LogType_ResourceFlow, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		RoleId:         strconv.FormatUint(h.actor.roleId, 10),
		//		ResourceId:     strconv.FormatUint(uint64(typ), 10),
		//		ResourceBefore: beforeNum,
		//		ResourceAfter:  afterNum,
		//		Flow:           "in",
		//		Action:         strconv.Itoa(int(reason)),
		//		Level:          int32(h.actor.LoginHandler.getRoleLevel()),
		//		VipLevel:       0,
		//		Recharge:       0,
		//	})
		// })
		threading.RunSafe(func() {
			e := &taptap.ResourceFlow{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(h.actor.roleId, 10),
				ResourceId:        strconv.FormatUint(uint64(typ), 10),
				ResourceBefore:    beforeNum,
				ResourceAfter:     afterNum,
				Flow:              "in",
				Action:            strconv.Itoa(int(reason)),
				Level:             int32(h.actor.LoginHandler.getRoleLevel()),
				VipLevel:          0,
				Recharge:          0,
			}
			taptap.WriteDataLog(taptap.LogType_ResourceFlow, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	return nil
}

// AddCurrency 原增加货币方法
func (h *CurrencyHandler) AddCurrency(typ int32, value int64, commonData *clidto.Comdata, reason common.ChangeReason) error {

	item, err := h.GetValue(typ)
	if err != nil {
		return err
	}

	// 溢出判定
	after := item.GetValue() + value
	hardLimit := getHardLimit(typ)
	if after > hardLimit {
		sub := after - hardLimit
		value = hardLimit - item.GetValue()
		err = h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_2, map[int32]int32{typ: int32(sub)}, commonData)
		if err != nil {
			return err
		}
	}

	// 增加
	if err = h.AddValue(typ, value, commonData, reason); err != nil {
		return err
	}

	return nil
}

// SubValue 扣除value的公共方法
func (h *CurrencyHandler) SubValue(typ int32, value int64, commonData *clidto.Comdata, reason common.ChangeReason) error {
	var beforeNum, afterNum int64
	// 扣除
	item, err := h.GetValue(typ)
	if err != nil {
		return err
	}

	// 够扣除？
	if item.GetValue() < value {
		return fmt.Errorf("sub currency failed. curValue: %d, subValue: %d", item.GetValue(), value)
	}
	beforeNum = item.GetValue()
	afterNum = beforeNum - value
	item.Value = afterNum

	h.actor.GetCurrencyData().Currencyx[typ] = item
	commonData.Data.Currency = append(commonData.Data.Currency, item)

	// 保存
	err = h.SaveDB()
	if err != nil {
		return err
	}

	if typ == common.CURRENCY_ITEM_ID_2006 {
		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.MoneyFlow{
		//		HeadInfo:    lilith.BuildHeadInfo(lilith.LogType_MoneyFlow, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		RoleId:      h.actor.ID(),
		//		MoneyBefore: beforeNum,
		//		MoneyAfter:  afterNum,
		//		Flow:        "out",
		//		Action:      strconv.Itoa(int(reason)),
		//		Level:       int32(h.actor.LoginHandler.getRoleLevel()),
		//		VipLevel:    0,
		//		MoneyType:   strconv.Itoa(int(typ)),
		//		Item:        "",
		//		Recharge:    0,
		//	})
		// })
		threading.RunSafe(func() {
			e := &taptap.MoneyFlow{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            h.actor.ID(),
				MoneyBefore:       beforeNum,
				MoneyAfter:        afterNum,
				Flow:              "out",
				Action:            strconv.Itoa(int(reason)),
				Level:             int32(h.actor.LoginHandler.getRoleLevel()),
				VipLevel:          0,
				MoneyType:         strconv.Itoa(int(typ)),
				Item:              "",
				Recharge:          0,
			}
			taptap.WriteDataLog(taptap.LogType_MoneyFlow, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	} else {
		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.ResourceFlow{
		//		HeadInfo:       lilith.BuildHeadInfo(lilith.LogType_ResourceFlow, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		RoleId:         strconv.FormatUint(h.actor.roleId, 10),
		//		ResourceId:     strconv.FormatUint(uint64(typ), 10),
		//		ResourceBefore: beforeNum,
		//		ResourceAfter:  afterNum,
		//		Flow:           "out",
		//		Action:         strconv.Itoa(int(reason)),
		//		Level:          int32(h.actor.LoginHandler.getRoleLevel()),
		//		VipLevel:       0,
		//		Recharge:       0,
		//	})
		// })
		threading.RunSafe(func() {
			e := &taptap.ResourceFlow{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(h.actor.roleId, 10),
				ResourceId:        strconv.FormatUint(uint64(typ), 10),
				ResourceBefore:    beforeNum,
				ResourceAfter:     afterNum,
				Flow:              "out",
				Action:            strconv.Itoa(int(reason)),
				Level:             int32(h.actor.LoginHandler.getRoleLevel()),
				VipLevel:          0,
				Recharge:          0,
			}
			taptap.WriteDataLog(taptap.LogType_ResourceFlow, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	// 发布事件
	// e := event.NewBasicEvent(TASK_EVENT_CURRENCY_SUB, []int32{TASK_TYPE_112}, nil)
	// e.Set("type", typ)
	// e.Set("count", value)
	// h.actor.eventManager.SyncPublish(e)
	return nil
}

// SubCurrency 原扣除货币方法
func (h *CurrencyHandler) SubCurrency(typ int32, value int64, commonData *clidto.Comdata, reason common.ChangeReason) error {
	// 消耗免费灵之砂,不足自动消耗付费灵之砂
	if typ == common.CURRENCY_ITEM_ID_2005 {
		// 计算差值
		item, err := h.GetValue(typ)
		if err != nil {
			return err
		}
		need := value - item.GetValue()
		if need > 0 {
			value = item.GetValue()
			if err = h.SubValue(common.CURRENCY_ITEM_ID_2006, need, commonData, reason); err != nil {
				return err
			}
		}
	}
	return h.SubValue(typ, value, commonData, reason)
}

func (h *CurrencyHandler) GetValue(typ int32) (*pb.CurrencyItem, error) {
	if !isValidType(typ) {
		return nil, fmt.Errorf("GetValue. unknown currency type: %d", typ)
	}

	if _, ok := h.actor.GetCurrencyData().Currencyx[typ]; !ok {
		h.actor.GetCurrencyData().Currencyx[typ] = &pb.CurrencyItem{
			Key:   typ,
			Value: 0,
		}
	}

	return h.actor.GetCurrencyData().Currencyx[typ], nil
}

// CheckEnough 检查货币是否足够，足够则返回true
func (h *CurrencyHandler) CheckEnough(typ int32, value int64) bool {
	item, err := h.GetValue(typ)
	if err != nil {
		return false
	}
	// 消耗免费灵之砂，不足自动消耗付费灵之砂
	if typ == common.CURRENCY_ITEM_ID_2005 {
		need := value - item.GetValue()
		if need > 0 {
			return h.CheckEnough(common.CURRENCY_ITEM_ID_2006, need)
		}
	}
	return item.GetValue() >= value
}

func isValidType(typ int32) bool {
	cfg := excel.GetItemMgr().GetById(typ)
	if cfg == nil {
		return false
	}
	return cfg.Type == int32(pb.ItemType_Currency)
}

// 获取硬上限
func getHardLimit(typ int32) int64 {
	cfg := excel.GetItemMgr().GetById(typ)
	if cfg == nil {
		return math.MaxInt64
	}
	return cfg.NumLimit
}

func (h *CurrencyHandler) CheckLimit(typ int32, value int64) bool {
	item, err := h.GetValue(typ)
	if err != nil {
		return true
	}

	if item.GetValue()+value > getHardLimit(typ) {
		return true
	}

	return false
}

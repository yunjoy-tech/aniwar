package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type ItemUseFunc = func(*clidto.Comdata, int32, int32) error
type ItemUseCheckFunc = func(int32, int32) cmd.ErrorCode

type BagHandler struct {
	*UABaseHandler
	UseHandlerMap   map[int32]ItemUseFunc
	CheckHandlerMap map[int32]ItemUseCheckFunc
}

func NewBagHandler(actor *UserActor) *BagHandler {
	h := &BagHandler{
		UABaseHandler:   NewUABaseHandler(actor, "BagHandler"),
		UseHandlerMap:   make(map[int32]ItemUseFunc),
		CheckHandlerMap: make(map[int32]ItemUseCheckFunc),
	}
	h.ChildHandler = h

	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_UseItemReq), h.UseItemReq)                     // 使用道具
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PLS2S_DestroyExpireItemReq), h.DestroyExpireItemReq) // 销毁过期道具
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ItemBuyReq), h.ItemBuyReq)                     // 购买道具
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PS2S_ReduceUserItemReq), h.ReduceUserItem)           // gm 扣除道具

	return h
}

// Init 初始化模块数据
func (h *BagHandler) Init() error {
	// 初始化
	h.actor.Data.ItemData = &cmd.PCommonItemInfos{
		Createtime: time.Now().Unix(),
		Items:      make(map[uint64]*cmd.PCommonItemInfo),
	}

	// 保存
	err := h.SaveDB(true)
	if err != nil {
		return err
	}

	h.Debug("init bag data success. player: %s", h.actor.ID())
	return nil
}

func (h *BagHandler) EnterGame() error {
	return nil
}

func (h *BagHandler) DailyRefresh() error {
	return nil
}

func (h *BagHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PCommonItemInfos); ok {
		h.actor.Data.ItemData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *BagHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserItems(h.actor.ID()), h.actor.Data.ItemData
}

// 延迟初始化使用道具协议映射
func (h *BagHandler) tryInitMap() {
	if len(h.UseHandlerMap) > 0 && len(h.CheckHandlerMap) > 0 {
		return
	}
	// 注册使用道具方法
	h.UseHandlerMap[int32(cmd.ItemType_Consumable)*100+int32(cmd.ItemConsumableType_BuildingBlueprint)] = h.actor.CampHandler.UnlockBuildingByBlueprint
	h.UseHandlerMap[int32(cmd.ItemType_Consumable)*100+int32(cmd.ItemConsumableType_StaminaWater)] = h.actor.PlayerLevelHandler.useStaminaItem
	// 注册前置校验方法
	h.CheckHandlerMap[int32(cmd.ItemType_Consumable)*100+int32(cmd.ItemConsumableType_BuildingBlueprint)] = h.actor.CampHandler.BuildingBlueprintCheck
	h.CheckHandlerMap[int32(cmd.ItemType_Consumable)*100+int32(cmd.ItemConsumableType_StaminaWater)] = h.actor.PlayerLevelHandler.useStaminaItemCheck
	h.Debugf("tryInitMap init==== %s", h.actor.ID())
}

// UseItemReq 使用道具
func (h *BagHandler) UseItemReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_UseItemReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 检查配置
	itemCfg := excel.GetItemMgr().GetById(int32(req.ItemId))
	if itemCfg == nil {
		return nil, fmt.Errorf("item config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}
	costItem := make(map[int32]int32)
	costItem[int32(req.ItemId)] = int32(req.ItemNum)
	consumeMgr := GetConsumeMgr(h.actor)
	// 检查道具是否充足
	if int32(req.ItemNum) <= 0 && !consumeMgr.CheckMapEnough(costItem) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}
	h.tryInitMap()
	// 取注册的执行逻辑
	k := itemCfg.Type*100 + itemCfg.SubType
	useFunc := h.UseHandlerMap[k]
	checkFunc := h.CheckHandlerMap[k]
	if useFunc == nil || checkFunc == nil {
		return nil, fmt.Errorf("item use func not implemented %d", k), int32(cmd.ErrorCode_InternalError)
	}

	// 校验方法执行
	code := checkFunc(int32(req.ItemId), int32(req.ItemNum))
	if code != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("item check failed"), int32(code)
	}

	// 扣除道具
	err := consumeMgr.doConsume(itemCfg, req.ItemNum, h.actor.comData, common.CR_UseItem)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 执行其他逻辑
	err = useFunc(h.actor.comData, int32(req.ItemId), int32(req.ItemNum))
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.UseItem{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_UseItem, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ItemId:         int32(req.ItemId),
	//		ItemNum:        int32(req.ItemNum),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.UseItem{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ItemId:            int32(req.ItemId),
			ItemNum:           int32(req.ItemNum),
		}
		taptap.WriteDataLog(taptap.LogType_UseItem, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &cmd.LS2C_UseItemRes{ErrCode: cmd.ErrorCode_Success, CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *BagHandler) DestroyExpireItemReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_DestroyExpireItemReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 检查是否过期, 过期了才销毁
	for _, uniqueId := range req.ItemUniqueIds {
		target := h.actor.GetUserItems().Items[uniqueId]
		if target == nil {
			return nil, fmt.Errorf("not found item %d", uniqueId), int32(cmd.ErrorCode_NotFoundItem)
		}

		if !hasExpiration(target) {
			return nil, fmt.Errorf("item not expire %d", target.ExpirationTimestamp), int32(cmd.ErrorCode_Had_Not_Expiration)
		}
	}

	// 销毁全部
	consumeMgr := GetConsumeMgr(h.actor)
	err := consumeMgr.ConsumeAll(req.ItemUniqueIds, h.actor.comData, common.CR_Destroy_EXP_ITEM)
	h.Debugf("consumeMgr.ConsumeAll ===>>> err:%+v", err)

	h.Debugf("consumeMgr.ConsumeAll ===>>> 奖励:%+v", consumeMgr.ExchangeRewards)

	_, err = GetDropMgr(h.actor).DropListByItems(consumeMgr.ExchangeRewards, true, nil, h.actor.comData, common.CR_Destroy_EXP_ITEM)

	return &cmd.LS2C_DestroyExpireItemRes{ErrCode: cmd.ErrorCode_Success, CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *BagHandler) ItemBuyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.C2LS_ItemBuyReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	if len(req.Items) == 0 {
		return &cmd.LS2C_ItemBuyRes{}, nil, 0
	}

	needItem := make(map[int32]int32)
	costItem := make(map[int32]int32)
	// 检查是否可购买
	for _, v := range req.Items {
		cfg := excel.GetDirectPurchaseMgr().GetById(v.Key)
		if cfg == nil || v.Value <= 0 {
			return nil, fmt.Errorf("param error"), int32(cmd.ErrorCode_ParamError)
		}
		needItem[v.Key] = v.Value
		costItem[cfg.Price.Key] = cfg.Price.Val * v.Value
	}

	// 货币是否足够
	if !GetConsumeMgr(h.actor).CheckMapEnough(costItem) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_CurrencyNotEnough)
	}

	// 扣除货币
	err := GetConsumeMgr(h.actor).ConsumeList(costItem, h.actor.comData, common.CR_BuyItem)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 发道具
	_, err = GetDropMgr(h.actor).DropList2(needItem, true, nil, h.actor.comData, common.CR_BuyItem)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	for _, v := range req.Items {
		cfg := excel.GetDirectPurchaseMgr().GetById(v.Key)
		// 埋点log
		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.ItemBuy{
		//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_ItemBuy, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		ItemId:         v.Key,
		//		ItemNum:        v.Value,
		//		MoneyType:      cfg.Price.Key,
		//		MoneyNum:       cfg.Price.Val * v.Value,
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.ItemBuy{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				ItemId:            v.Key,
				ItemNum:           v.Value,
				MoneyType:         cfg.Price.Key,
				MoneyNum:          cfg.Price.Val * v.Value,
			}
			taptap.WriteDataLog(taptap.LogType_ItemBuy, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	return &cmd.LS2C_ItemBuyRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *BagHandler) ReduceUserItem(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.S2S_ReduceUserItemReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 道具消耗check
	if !GetConsumeMgr(h.actor).CheckEnough(req.GetItemId(), req.GetNum()) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}

	// 扣除道具
	err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{req.GetItemId(): req.GetNum()}, h.actor.comData, common.CR_GM)
	if err != nil {
		h.Error("ConsumeList err:", err)
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_InternalError)
	}
	rsp := &cmd.S2S_ReduceUserItemRes{}

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

// 推送真实的道具数据
func (h *BagHandler) buildItemList() []*cmd.PCommonItemInfo {
	userItems := h.actor.UserData.GetUserItems().Items
	items := make([]*cmd.PCommonItemInfo, 0)
	for _, info := range userItems {
		items = append(items, &cmd.PCommonItemInfo{
			BaseId:              info.BaseId,
			UniqueId:            info.UniqueId,
			ItemNum:             info.ItemNum,
			ExpirationTimestamp: info.ExpirationTimestamp,
		})
	}

	return items
}

//func (h *BagHandler) GetItemValueById(itemId int32) []*cmd.KeyValueItem {
//	return h.GetItemValue([]int32{itemId})
//}

func (h *BagHandler) GetItemValueById(itemId int32) *cmd.KeyValueItem {
	items := h.GetItemValueByIds(itemId)
	if len(items) <= 0 {
		// 没有数据
		return nil
	}

	return items[0]
}

func (h *BagHandler) GetItemValueByIds(items ...int32) []*cmd.KeyValueItem {
	result := make([]*cmd.KeyValueItem, 0, len(items))

	for _, itemId := range items {
		temp := &cmd.KeyValueItem{
			Key:   itemId,
			Value: h.GetItemNum(itemId),
		}
		result = append(result, temp)
	}

	return result
}

func (h *BagHandler) GetItemNum(itemId int32) int32 {
	item := h.actor.GetUserItems().Items[uint64(itemId)]
	if item != nil {
		return int32(item.ItemNum)
	}
	return 0
}

func (h *BagHandler) GetItemByUniqueId(uniqueId uint64) *cmd.PCommonItemInfo {
	item := h.actor.GetUserItems().Items[uniqueId]
	if item != nil {
		return item
	}

	return nil
}

// GetMulItemNum 获取多个道具的数量
func (h *BagHandler) GetMulItemNum(itemIds []int32) []*cmd.KeyValueItem {
	items := make([]*cmd.KeyValueItem, 0, len(itemIds))
	for _, id := range itemIds {
		items = append(items, &cmd.KeyValueItem{
			Key:   id,
			Value: h.GetItemNum(id),
		})
	}
	return items
}

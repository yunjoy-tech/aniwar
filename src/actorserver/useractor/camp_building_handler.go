package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"google.golang.org/protobuf/proto"
	"time"
)

/////////////////////////////////////////////////////////建筑相关

// PlayerCampFuncUpCardReq 建筑卡牌驻守
func (h *CampHandler) PlayerCampFuncUpCardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	msgId, _ := in.MsgId, in.Data
	var req cmd.C2LS_PlayerCampFuncUpCardReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 0, msgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	// 卡牌是否存在
	if !h.actor.CardHandler.IsExistCard(uint32(req.CardId)) {
		return nil, fmt.Errorf("card not exist %d", req.CardId), int32(cmd.ErrorCode_CardNotExist)
	}

	// 是否已经上阵
	for _, v := range h.actor.GetCampData().DecorationBuilding {
		if v.CardId == req.CardId {
			return nil, fmt.Errorf("card is up %d", req.CardId), int32(cmd.ErrorCode_CampCardIsUp)
		}
	}
	// 是否有增益角色生产，且生产队列没有完成
	if !h.CheckGainCard(commonParams, req.GetBuildingId()) {
		return nil, fmt.Errorf("camp Production Status"), int32(cmd.ErrorCode_CampProductionStatus)
	}
	// 上阵
	beforeCard := commonParams.building.CardId // 上阵前的卡牌
	commonParams.building.CardId = req.CardId
	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.actor.comData.GetCampData().DecorationBuilding = append(h.actor.comData.GetCampData().DecorationBuilding, commonParams.building)

	// 建筑卡牌驻守 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampBuildingUpcard{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampBuildingUpcard, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                            //建筑唯一id
	//		BuildingId:     req.BuildingId,                      //建筑id
	//		Lv:             commonParams.building.BuildingLevel, //建筑等级
	//		BeforeCard:     beforeCard,                          //上阵前的卡牌
	//		AfterCard:      commonParams.building.CardId,        //上阵后的卡牌
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampBuildingUpcard{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                            //建筑唯一id
			BuildingId:        req.BuildingId,                      //建筑id
			Lv:                commonParams.building.BuildingLevel, //建筑等级
			BeforeCard:        beforeCard,                          //上阵前的卡牌
			AfterCard:         commonParams.building.CardId,        //上阵后的卡牌
		}
		taptap.WriteDataLog(taptap.LogType_CampBuildingUpcard, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
	// 任务计数
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_UP_CARD, []int32{TASK_TYPE_518}, map[string]interface{}{
		"build_id": commonParams.building.ItemId,
		"card_id":  req.CardId,
	}))
	if errx != nil {
		h.Error(errx)
	}

	return &cmd.LS2C_PlayerCampFuncUpCardRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// PlayerCampFuncDownCardReq 建筑卡牌下阵
func (h *CampHandler) PlayerCampFuncDownCardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	msgId, _ := in.MsgId, in.Data
	var req cmd.C2LS_PlayerCampFuncDownCardReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 0, msgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}
	// 是否有增益角色生产，且生产队列没有完成
	if !h.CheckGainCard(commonParams, req.GetBuildingId()) {
		return nil, fmt.Errorf("camp Production Status"), int32(cmd.ErrorCode_CampProductionStatus)
	}

	beforeCard := commonParams.building.CardId // 上阵前的卡牌
	// 上阵
	commonParams.building.CardId = 0
	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.actor.comData.GetCampData().DecorationBuilding = append(h.actor.comData.GetCampData().DecorationBuilding, commonParams.building)

	// 建筑卡牌下阵 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampBuildingDownCard{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampBuildingDownCard, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                            //建筑唯一id
	//		BuildingId:     req.BuildingId,                      //建筑id
	//		Lv:             commonParams.building.BuildingLevel, //建筑等级
	//		BeforeCard:     beforeCard,                          //上阵前的卡牌
	//		AfterCard:      commonParams.building.CardId,        //上阵后的卡牌
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampBuildingDownCard{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                            //建筑唯一id
			BuildingId:        req.BuildingId,                      //建筑id
			Lv:                commonParams.building.BuildingLevel, //建筑等级
			BeforeCard:        beforeCard,                          //上阵前的卡牌
			AfterCard:         commonParams.building.CardId,        //上阵后的卡牌
		}
		taptap.WriteDataLog(taptap.LogType_CampBuildingDownCard, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &cmd.LS2C_PlayerCampFuncDownCardRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// CampMakeFunctionBuildingReq 修建建筑 功能建筑一旦建造，便属于某个具体的营地
func (h *CampHandler) CampMakeFunctionBuildingReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	_, uid, _ := in.MsgId, in.UserId, in.Data
	var req cmd.C2LS_PlayerCampMakeFunctionBuildingReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	if req.GetItemId() != common.Camp_Building_Type_Food { // 食品加工厂不走功能解锁功能
		if err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
			return nil, err, int32(code)
		}
	}
	commonParams := NewOutputParams(false, 0, 0, 0) //OutputParams{isFunctionBuilding: false}
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	build := excel.GetBuildMainMgr().GetById(req.ItemId)
	if build == nil {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	buildType := build.GetBuildType()
	if buildType <= int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_None) || buildType >= int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_Max) {
		return nil, fmt.Errorf("building type not match"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
	}

	// 根据道具表配置的道具上限检查现有道具是否达到上限，错误码需要更改
	if isLimit := h.checkBuildingIsNumLimit(req.ItemId, 1); isLimit {
		return nil, fmt.Errorf("building num limit"), int32(cmd.ErrorCode_CampBuildingNumLimit)
	}
	// 是否光合树，单营地只允许存在一颗树
	if buildType == int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_LightingComposeTree) && commonParams.camp.LightingComposeTree != nil {
		return nil, fmt.Errorf("building num limit"), int32(cmd.ErrorCode_CampBuildingNumLimit)
	}

	// 功能建筑首次建造等级必然是1，建筑道具ID*100 加上1即为建筑等级编号
	buildLevelId := build.Id*100 + 1
	buildLevelConfig := excel.GetBuildingLevelMgr().GetById(buildLevelId)
	if buildLevelConfig == nil {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	//检查解锁条件
	openCondition := NewOpenCondition(h.actor)
	if ok := openCondition.AllowOpens(buildLevelConfig.BuildCondition); !ok {
		return nil, fmt.Errorf("condition not match"), int32(cmd.ErrorCode_CampBuildingMakeRequirePreCondition)
	}
	// 耗材扣除,1级代表建造所需材料
	cost := map[int32]int32{}
	for _, v := range buildLevelConfig.GetLevelCost() {
		cost[v.ItemId] = v.Num
	}

	//检查道具是否充足
	if !h.consumerItem(uid, cost, h.actor.comData, common.CR_Camp_Building_Create) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}

	//build
	buildings := h.GetProcess(cmd.PlayerCampBuildingType(build.BuildType))
	building, err, code := buildings.Build(commonParams, h, &req, buildLevelConfig)
	if code != int32(cmd.ErrorCode_Success) {
		return nil, err, code
	}

	// 先落地再通知
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 修建建筑埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.MakeFuncBuilding{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_MakeFuncBuilding, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ItemId:         req.ItemId,                      //功能建筑道具id
	//		Id:             build.Id,                        //建筑唯一id
	//		BuildingId:     building.Building.BuildingId,    //建筑配置id
	//		Lv:             building.Building.BuildingLevel, //建筑等级
	//		BuildingType:   build.GetBuildType(),            //建筑类型id
	//		Costs:          lilith.ConvertMap2Str(cost),     //建造消耗材料
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.MakeFuncBuilding{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ItemId:            req.ItemId,                      //功能建筑道具id
			Id:                build.Id,                        //建筑唯一id
			BuildingId:        building.Building.BuildingId,    //建筑配置id
			Lv:                building.Building.BuildingLevel, //建筑等级
			BuildingType:      build.GetBuildType(),            //建筑类型id
			Costs:             taptap.ConvertMap2Str(cost),     //建造消耗材料
		}
		taptap.WriteDataLog(taptap.LogType_MakeFuncBuilding, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 任务计数
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_MAKE, []int32{TASK_TYPE_517, TASK_TYPE_511, TASK_TYPE_512}, map[string]interface{}{
		"build_id": building.Building.ItemId,
		"level":    building.Building.BuildingLevel,
	}))
	if errx != nil {
		h.Error(errx)
	}

	res := &cmd.LS2C_PlayerCampMakeFunctionBuildingRes{}
	res.Building = building
	res.CommonData = h.actor.comData.FixDownComData()
	return res, nil, 0
}

// CampFunctionBuildingLvUpReq 功能建筑升级
func (h *CampHandler) CampFunctionBuildingLvUpReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	msgId, uid, _ := in.MsgId, in.UserId, in.Data
	var req cmd.C2LS_PlayerCampFunctionBuildingLvUpReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 1, msgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	// 如果是光合树, 检查camp.LightingComposeTree是否是空指针
	isLightingComposeTree := commonParams.buildType == int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_LightingComposeTree)
	if isLightingComposeTree && commonParams.camp.LightingComposeTree == nil {
		return nil, fmt.Errorf("building type not match"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
	}

	//检查解锁条件
	openCondition := NewOpenCondition(h.actor)
	if ok := openCondition.AllowOpens(commonParams.buildLevelConfig.BuildCondition); !ok {
		return nil, fmt.Errorf("condition not match"), int32(cmd.ErrorCode_CampBuildingMakeRequirePreCondition)
	}

	// 耗材扣除,1级代表建造所需材料
	cost := map[int32]int32{}
	for _, v := range commonParams.buildLevelConfig.GetLevelCost() {
		cost[v.ItemId] = v.Num
	}
	//检查道具是否充足
	if !h.consumerItem(uid, cost, h.actor.comData, common.CR_Camp_Building_Create) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}

	res := &cmd.LS2C_PlayerCampFunctionBuildingLvUpRes{}
	campRet := &cmd.PPlayerCamp{}
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)

	//升级
	buildings := h.GetProcess(cmd.PlayerCampBuildingType(build.BuildType))
	if err, code := buildings.LevelUp(commonParams, campRet, h); code != int32(cmd.ErrorCode_Success) {
		return nil, err, code
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	// 建筑升级埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.BuildingLevelUp{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_BuildingLevelUp, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                                //建筑唯一id
	//		BuildingId:     req.BuildingId,                          //建筑配置id
	//		BuildingType:   build.BuildType,                         //建筑类型id
	//		Costs:          lilith.ConvertMap2Str(cost),             //建造消耗材料
	//		BeforeLv:       commonParams.building.BuildingLevel - 1, //升级前等级
	//		AfterLv:        commonParams.building.BuildingLevel,     //升级后等级
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.BuildingLevelUp{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                                //建筑唯一id
			BuildingId:        req.BuildingId,                          //建筑配置id
			BuildingType:      build.BuildType,                         //建筑类型id
			Costs:             taptap.ConvertMap2Str(cost),             //建造消耗材料
			BeforeLv:          commonParams.building.BuildingLevel - 1, //升级前等级
			AfterLv:           commonParams.building.BuildingLevel,     //升级后等级
		}
		taptap.WriteDataLog(taptap.LogType_BuildingLevelUp, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 任务计数
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_LEVELUP, []int32{TASK_TYPE_302, TASK_TYPE_303, TASK_TYPE_511, TASK_TYPE_512}, map[string]interface{}{
		"build_id": commonParams.building.ItemId,
		"level":    commonParams.building.BuildingLevel,
	}))
	if errx != nil {
		h.Error(errx)
	}

	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, campRet)
	res.Building = commonParams.building
	res.CommonData = h.actor.comData.FixDownComData()

	return res, nil, 0
}

// CampBuildingFoundryReq 家具制造请求
func (h *CampHandler) CampBuildingFoundryReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	_, uid, _ := in.MsgId, in.UserId, in.Data

	var req cmd.C2LS_PlayerCampBuildingFoundryReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	buildMainMgr := excel.GetBuildMainMgr()

	// check 家具数量到达上限或者该行为会触发超过上限的情况
	buildCfg := buildMainMgr.GetById(req.ItemId)
	if buildCfg == nil {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}
	if req.Num <= 0 || req.Num > buildCfg.MakeLimit {
		return nil, fmt.Errorf("building num limit"), int32(cmd.ErrorCode_CampBuildingNumLimit)
	}
	// 达到持有上限
	if isLimit := h.checkBuildingIsNumLimit(req.ItemId, req.Num); isLimit {
		return nil, fmt.Errorf("building num limit"), int32(cmd.ErrorCode_CampBuildingNumLimit)
	}

	buildsMap := h.actor.GetCampData().DecorationBuilding
	building, ok := buildsMap[req.BuildingId]
	if !ok || building.BuildingId == 0 {
		return nil, fmt.Errorf("building not exist"), int32(cmd.ErrorCode_CampBuildingNotExist)
	}
	// 获取该建筑对应的配置
	funBuildCfg := buildMainMgr.GetById(building.ItemId)
	if funBuildCfg == nil {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	if funBuildCfg.BuildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_BuildingFoundry) {
		return nil, fmt.Errorf("building type not match"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
	}

	// 检测家具时候解锁
	unlockList := h.actor.GetCampData().BuildingUnlockList
	if _, ok := unlockList[req.ItemId]; !ok {
		return nil, fmt.Errorf("camp building locked"), int32(cmd.ErrorCode_CampBuildingLocked)
	}

	// cost
	cost := make(map[int32]int32)
	for _, v := range buildCfg.MakeCost {
		cost[v.ItemId] += req.Num * v.Num
	}

	//减少消耗
	cost, _ = h.LifeSkillSubCost(building, cost, 0)

	//检查道具是否充足，如果充足直接消费掉
	if !h.consumerItem(uid, cost, h.actor.comData, common.CR_Camp_Building_Create) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}
	// 批量添加道具
	buildings, err := h.batchAddDecorationBuildingInternal(req.ItemId, uint32(req.Num))
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.actor.comData.GetCampData().DecorationBuilding = buildings
	res := &cmd.LS2C_PlayerCampBuildingFoundryRes{
		CommonData: h.actor.comData.FixDownComData(),
	}

	// 发布事件
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_CREATE, []int32{TASK_TYPE_311}, map[string]interface{}{
		"count": req.Num,
	}))
	if errx != nil {
		h.Error(errx)
	}

	return res, nil, int32(cmd.ErrorCode_Success)
}

//////////////////////////////////////////////////// 通过bag_handler调用

// BuildingBlueprintCheck 检查家具图纸
func (h *CampHandler) BuildingBlueprintCheck(blueprintId, itemNum int32) cmd.ErrorCode {
	if itemNum != 1 {
		return cmd.ErrorCode_InvalidParam
	}
	cfg := excel.GetItemMgr().GetById(blueprintId)

	buildItemId := cfg.Id / 1000
	if buildItemId*1000+1 != cfg.Id {
		return cmd.ErrorCode_ConfigError
	}
	buildCfg := excel.GetBuildMainMgr().GetById(buildItemId)
	if buildCfg == nil {
		return cmd.ErrorCode_NotFoundConfig
	}
	return cmd.ErrorCode_Success
}

// UnlockBuildingByBlueprint 通过图纸解锁家具制造方式
func (h *CampHandler) UnlockBuildingByBlueprint(commonData *clidto.Comdata, itemId, itemNum int32) error {
	cfg := excel.GetItemMgr().GetById(itemId)

	// 构建数据
	buildId := itemId / 1000
	changeItemId := cfg.Change.ItemId
	changeItemNum := cfg.Change.Num

	buildingUnlockList := h.actor.GetCampData().BuildingUnlockList
	if _, ok := buildingUnlockList[buildId]; !ok {
		buildingUnlockList[buildId] = buildId
		if err := h.SaveDB(); err != nil {
			return err
		}
		commonData.GetCampData().BuildingUnlockList = []int32{buildId}
	} else {
		// 已经激活的图纸可能有其他转换
		if changeItemId > 0 && changeItemNum > 0 {
			reward := make(map[int32]int32)
			reward[changeItemId] = changeItemNum
			_, err := GetDropMgr(h.actor).DropList2(reward, true, nil, commonData, common.CR_Camp_CampMaterial_Conversion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//////////////////////////////////////////////////////内部调用发发

// 通过家具制作台来解锁
func (h *CampHandler) setAndNtfBuildingUnlockList(level int32) []int32 {
	list := make([]int32, 0)
	buildingUnlockList := h.actor.GetCampData().BuildingUnlockList
	excel.GetBuildMainMgr().Foreach(func(cfg *excel.BuildMainCfg) bool {
		if cfg.MakeLevel == level {
			if _, ok := buildingUnlockList[cfg.Id]; !ok {
				buildingUnlockList[cfg.Id] = cfg.Id
				list = append(list, cfg.Id)
			}
		}
		return true
	}, true)
	return list
}

// 检查建筑数量是否达到配置表的上限
func (h *CampHandler) checkBuildingIsNumLimit(itemId int32, itemNum int32) bool {
	numLimit := int32(GetItemLimit(itemId))
	if numLimit == 0 {
		return false
	}
	num := h.GetBuildingNum(itemId)
	return num+itemNum > numLimit
}

func (h *CampHandler) GetBuildingNum(itemId int32) int32 {
	total := int32(0)
	buildings := h.actor.GetCampData().DecorationBuilding
	for _, v := range buildings {
		if v.ItemId == itemId {
			total++
		}
	}
	return total
}

func (h *CampHandler) batchAddDecorationBuildingInternal(id int32, itemNum uint32) ([]*cmd.PPlayerCampDecorationBuilding, error) {
	buildings := make([]*cmd.PPlayerCampDecorationBuilding, 0)

	buildMainMgr := excel.GetBuildMainMgr()
	buildCfg := buildMainMgr.GetById(id)
	cost := make(map[int32]uint32)
	for _, v := range buildCfg.MakeCost {
		cost[v.ItemId] += itemNum * uint32(v.Num) // 所需消耗的材料
	}

	for i := 0; i < int(itemNum); i++ {
		buildData := h.NewPPlayerCampDecorationBuilding(id, 1, 0)
		h.Debugf("批量添加道具，batchAddDecorationBuildingInternal add building,itemId[%d],buildingId[%d],buildingItermId[%d]", id, buildData.BuildingId, buildData.ItemId)
		h.actor.GetCampData().DecorationBuilding[buildData.BuildingId] = buildData

		// 家具打造埋点
		threading.RunSafe(func() {
			e := &taptap.CampBuildingFoundry{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				Id:                buildCfg.Id,                 //建筑唯一id
				BuildingId:        buildData.BuildingId,        //建筑id
				Lv:                1,                           //建筑等级
				ItemId:            id,                          //待制造的家具id
				Num:               1,                           //待制造的数量
				Cost:              taptap.ConvertMap2Str(cost), //消耗材料
			}
			taptap.WriteDataLog(taptap.LogType_CampBuildingFoundry, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})

		buildings = append(buildings, buildData)
	}

	if err := h.SaveDB(); err != nil {
		h.Debug("CampHandler.PlayerCampAddDecorationBuildingNtf error:", err)
		return nil, err
	}

	h.Debug("addDecorationBuilding id:", id)
	return buildings, nil
}

// CheckGainCard 检查是否有增益角色，且生产队列还未完成
func (h *CampHandler) CheckGainCard(commonParams *OutputParams, buildingId int64) bool {
	// 只有熔炉才有生产队列
	if commonParams.building.ItemId != 90074 {
		return true
	}
	if commonParams.building.CardId == 0 {
		return true
	}
	cardCfg := h.actor.CardHandler.GetCardCfg(commonParams.building.CardId)
	if cardCfg == nil {
		h.Debug("LifeSkillAddProduct get cardCfg is nil", commonParams.building.CardId)
		return true
	}
	lifeSkillCfgs := h.LifeSkill[commonParams.building.ItemId]
	if len(lifeSkillCfgs) == 0 {
		return true
	}
	if _, ok := lifeSkillCfgs[cardCfg.LifeskillID]; !ok {
		return true
	}
	buildingQueue, ok := commonParams.camp.WorkQueue[buildingId]
	if !ok {
		return true
	}
	// 有增益角色且队列未完成才返回false
	now := time.Now().Unix()
	for _, v := range buildingQueue.Queue {
		if v.EndTimestamp > now { //有未完成的
			return false
		}
	}
	return true
}

func GetItemLimit(itemId int32) uint32 {
	itemConfig := excel.GetItemMgr().GetById(itemId)
	if itemConfig == nil {
		return 0
	}
	return uint32(itemConfig.NumLimit)
}

// GetItemDiff 获取家具与高限制的插值
func (h *CampHandler) GetItemDiff(result []int32) map[int32]int32 {
	diffMap := make(map[int32]int32, len(result))
	for _, itemId := range result {
		diff := int32(GetItemLimit(itemId)) - h.GetBuildingNum(itemId)
		if diff < 0 {
			diff = 0
		}
		diffMap[itemId] = diff
	}
	return diffMap
}

// 邮件家具下发特殊处理
func (h *CampHandler) HandleDropMailItem(items []*cmd.ItemReward) ([]*cmd.ItemReward, error) {
	retItems := make([]*cmd.ItemReward, 0)
	drops := make(map[uint32]uint32)
	for _, item := range items {
		limit := GetItemLimit(int32(item.ItemId))
		// 获取现有数量
		curNum := uint32(h.GetBuildingNum(int32(item.ItemId)))
		// 区分可掉背包数量和可转换数量
		dropNum := item.Num
		convertNum := uint32(0)
		if limit > 0 && curNum+item.Num > limit {
			dropNum = limit - curNum
			convertNum = curNum + item.Num - limit
		}
		if dropNum > 0 {
			drops[item.ItemId] += dropNum
		}
		if convertNum > 0 {
			cfg := excel.GetItemMgr().GetById(int32(item.ItemId))
			id, num := h.ConvertBuilding(cfg.Quality)
			drops[uint32(id)] += uint32(num)
		}
	}

	if len(drops) > 0 {
		dropChange, err := GetDropMgr(h.actor).DropList(drops, true, nil, h.actor.comData, common.CR_Mail_Attachment)
		if err != nil {
			return nil, err
		}
		retItems = append(retItems, dropChange.Items...)
	}

	return retItems, nil
}

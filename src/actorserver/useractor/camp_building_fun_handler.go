package useractor

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"math/rand"

	//"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"google.golang.org/protobuf/proto"
	"time"
)

///////////////////////////////////////////////////////////////相关协议

// CampInfoReq 获取营地信息
func (h *CampHandler) CampInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	camp := h.buildCampList()
	for _, c := range camp.Camp {
		h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{Trader: c.Trader})
	}
	return &cmd.LS2C_PlayerCampInfoRes{CommonData: h.actor.comData.FixDownComData()}, nil, int32(cmd.ErrorCode_Success)
}

// CampFuncBuildingGetRewardReq 获取建造奖励（熔炉和食物补给站）
func (h *CampHandler) CampFuncBuildingGetRewardReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	if err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
		return nil, err, int32(code)
	}
	msgId, uid, _ := in.MsgId, in.UserId, in.Data
	var req cmd.C2LS_PlayerCampFuncBuildingGetRewardReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	h.Debugf("领取营地队列奖励 building:%d, queueId:%d", req.BuildingId, req.QueueId)

	// 检查建筑
	commonParams := NewOutputParams(true, req.BuildingId, 0, msgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	building, ok := commonParams.camp.WorkQueue[req.BuildingId]
	if !ok || building == nil {
		return nil, fmt.Errorf("queue not exist"), int32(cmd.ErrorCode_CampWorkQueueNotExist)
	}

	//get workQueue
	var workQueue *cmd.PPlayerCampFunctionBuildingWorkQueue
	var idx int
	for id, v := range building.Queue {
		if v.QueueId == req.QueueId {
			idx = id
			workQueue = v
			break
		}
	}
	if workQueue == nil {
		return nil, fmt.Errorf("queue not exist"), int32(cmd.ErrorCode_CampWorkQueueNotExist)
	}

	now := time.Now().Unix()
	if workQueue.EndTimestamp > now {
		return nil, fmt.Errorf("queue task not complete"), int32(cmd.ErrorCode_CampWorkQueueTaskIncomplete)
	}
	//发放奖励
	ok, reward := canGetReward(workQueue)
	if !ok {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// 增加产量
	extraAward := make(map[int32]int32, len(reward))
	for id, value := range reward {
		rate := float32(workQueue.Rate) / float32(100)
		extraAward[id] = int32(float32(value) * float32(rate))
		h.Debugf("建筑[%d],产物[%d],产量[%d],产量提升[%d]", building.BuildingId, id, value, float32(value)*float32(rate), int32(float32(value)*float32(rate)))
	}
	//extraAward := h.LifeSkillAddProduct(commonParams.building, reward)

	//判断是否超过背包
	if h.CheckMapLimit(reward, extraAward) {
		return nil, fmt.Errorf("package is full"), int32(cmd.ErrorCode_PackageIsFull)
	}

	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Camp_Building_Get)
	if err != nil {
		h.Error("CampFuncBuildingGetRewardReq DropListByItems, error", h.actor.ID(), uid)
	}
	extraChange, err := GetDropMgr(h.actor).DropList2(extraAward, true, nil, h.actor.comData, common.CR_Camp_Building_Get)
	if err != nil {
		h.Error("CampFuncBuildingGetRewardReq DropListByItems, error", h.actor.ID(), uid)
	}
	//删除队列
	building.Queue = append(building.Queue[:idx], building.Queue[idx+1:]...)

	//先落地，再发奖励
	if err := h.SaveDB(); err != nil {
		h.Error(err)
	}

	// 领取队列奖励 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	threading.RunSafe(func() {
		e := &taptap.GetqueueReward{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                            //建筑唯一id
			BuildingId:        req.BuildingId,                      //建筑id
			Lv:                commonParams.building.BuildingLevel, //建筑等级
			QueueId:           req.QueueId,                         //队列id
			Reward:            taptap.ConvertMap2Str(reward),       //产出奖励
		}
		taptap.WriteDataLog(taptap.LogType_GetqueueReward, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 发布事件
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_REWARD, []int32{TASK_TYPE_313, TASK_TYPE_510}, map[string]interface{}{
		"buildId": building.BuildConfigId,
		"reward":  reward, // fixme 需要总产物
	}))
	if errx != nil {
		h.Error(errx)
	}
	res := &cmd.LS2C_PlayerCampFuncBuildingGetRewardRes{BuildingId: req.BuildingId, QueueId: req.QueueId, CommonData: h.actor.comData.FixDownComData(), DropChange: dropChange, ExtraDropChange: extraChange}

	return res, nil, 0
}

// CampBuildFunReq 材料转化，熔炼、装备制作
func (h *CampHandler) CampBuildFunReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	if err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
		return nil, err, int32(code)
	}
	msgId, uid, _ := in.MsgId, in.UserId, in.Data
	var req cmd.C2LS_PlayerCampBuildFunOpReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	//if req.Formula == nil || req.Formula.Num <= 0 {
	//	return nil, fmt.Errorf("formula param error"), int32(cmd.ErrorCode_CampFormulaParamError)
	//}

	commonParams := NewOutputParams(true, req.BuildingId, 0, msgId)
	err, errCode := h.commonCheck(commonParams)
	if err != nil {
		return nil, err, errCode
	}

	formulaCfgMap := h.getFormulaByCfg(commonParams.buildLevelConfig, true)
	if len(formulaCfgMap) == 0 {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}
	buildings := h.GetProcess(cmd.PlayerCampBuildingType(commonParams.buildType))
	//check
	if code := buildings.Check(formulaCfgMap, req.GetFormula(), commonParams, h); code != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("buildings check err"), code
	}

	// 获取消耗
	totalCost, code := buildings.Cost(formulaCfgMap, req.GetFormula(), commonParams)
	if code != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("buildings get totalCost err"), code
	}
	//生活技能减少消耗
	totalCost, _ = h.LifeSkillSubCost(commonParams.building, totalCost, 0)
	// 检查道具是否充足
	if !GetConsumeMgr(h.actor).CheckMapEnough(totalCost) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}
	if err = GetConsumeMgr(h.actor).ConsumeList(totalCost, h.actor.comData, buildings.GetChangeReason()); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	res := &cmd.LS2C_PlayerCampBuildFunOpRes{
		BuildingId: req.BuildingId,
		Queue:      nil,
		CommonData: h.actor.comData.FixDownComData(),
	}

	// 发放产物
	pru, code := buildings.Product(formulaCfgMap, req.GetFormula(), h, commonParams, res)
	if code != int32(cmd.ErrorCode_Success) {
		h.Errorf("buildings product err, uid: %v,code:%d", uid, code)
		return nil, fmt.Errorf("buildings product err"), code
	}

	//埋点
	buildings.DataLog(commonParams, h, req.GetBuildingId(), taptap.ConvertMap2Str(totalCost), pru)

	return res, nil, 0
}

// CampFuncBuildingOpCancelReq 食物或者熔炉取消建造
func (h *CampHandler) CampFuncBuildingOpCancelReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	if err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
		return nil, err, int32(code)
	}
	msgId, uid, _ := in.MsgId, in.UserId, in.Data

	var req cmd.C2LS_PlayerCampFuncBuildingOpCancelReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	// 检查建筑
	commonParams := NewOutputParams(true, req.BuildingId, 0, msgId)
	err, errCode := h.commonCheck(commonParams)
	if err != nil {
		return nil, err, errCode
	}

	builds := h.actor.GetCampData().DecorationBuilding

	if _, ok := builds[req.BuildingId]; !ok {
		return nil, err, int32(cmd.ErrorCode_CampBuildingNotExist)
	}

	workQueue := h.getWorkQueueBuilding(req.BuildingId)
	if workQueue == nil {
		return nil, err, int32(cmd.ErrorCode_CampWorkQueueNotExist)
	}
	var curQueue *cmd.PPlayerCampFunctionBuildingWorkQueue
	var idx int
	for id, v := range workQueue.Queue {
		if v.QueueId == req.QueueId {
			curQueue = v
			idx = id
			break
		}
	}
	if curQueue == nil {
		return nil, err, int32(cmd.ErrorCode_CampWorkQueueNotExist)
	}

	cost := map[uint32]uint32{}
	itemSynthesisMgr := excel.GetItemSynthesisMgr()
	for _, v := range curQueue.Formula {
		cfg := itemSynthesisMgr.GetById(v.FormulaId)
		if cfg == nil {
			h.Error("CampFurnaceOpReq FormulaId not found", h.actor.ID(), uid)
			return nil, fmt.Errorf(""), int32(cmd.ErrorCode_CampFormulaNotExist)
		}
		for _, itemReward := range cfg.ItemCost {
			cost[uint32(itemReward.ItemId)] += uint32(itemReward.Num * v.Num)
		}
	}
	//删除建造队列中的项
	workQueue.Queue = append(workQueue.Queue[:idx], workQueue.Queue[idx+1:]...)
	if err = h.SaveDB(); err != nil {
		h.Error(err)
	}

	//add
	if _, err = GetDropMgr(h.actor).DropList(cost, true, nil, h.actor.comData, common.CR_Camp_Building_OpCancel); err != nil {
		h.Error("CampFuncBuildingOpCancelReq DropListByItems, error", h.actor.ID(), uid)
	}

	// 熔炉队列取消 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CancelFurnaceQueue{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CancelFurnaceQueue, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                                       //建筑唯一id
	//		BuildingId:     req.BuildingId,                                 //建筑id
	//		Lv:             commonParams.building.BuildingLevel,            //建筑等级
	//		QueueId:        req.QueueId,                                    //队列id
	//		Formula:        lilith.ConvertListStruct2Str(curQueue.Formula), //消耗材料
	//		Costs:          lilith.ConvertMap2Str(cost),                    //建造消耗材料
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CancelFurnaceQueue{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                                       //建筑唯一id
			BuildingId:        req.BuildingId,                                 //建筑id
			Lv:                commonParams.building.BuildingLevel,            //建筑等级
			QueueId:           req.QueueId,                                    //队列id
			Formula:           taptap.ConvertListStruct2Str(curQueue.Formula), //消耗材料
			Costs:             taptap.ConvertMap2Str(cost),                    //建造消耗材料
		}
		taptap.WriteDataLog(taptap.LogType_CancelFurnaceQueue, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	rsp := &cmd.LS2C_PlayerCampFuncBuildingOpCancelRes{BuildingId: req.BuildingId, QueueId: req.QueueId, CommonData: h.actor.comData.FixDownComData()}
	return rsp, nil, 0
}

// CampMakeFoodReq 自主烹饪
func (h *CampHandler) CampMakeFoodReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_PlayerCampMakeFoodReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 0, in.MsgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	costs := myUtils.ConvertItem(req.Items)
	// 校验食材
	var total int32
	max := h.getUpgradeChangeValue4(commonParams.buildLevelConfig)
	for k, v := range costs {
		total += v
		// 品质校验
		itemCfg := excel.GetItemMgr().GetById(k)
		if itemCfg == nil {
			return nil, fmt.Errorf("item cfg not found %d", k), int32(cmd.ErrorCode_NotFoundItem)
		}

		// 道具类型错误
		if itemCfg.Type != int32(cmd.ItemType_Material) || itemCfg.SubType != int32(cmd.ItemMaterialType_ItemMaterialType_Food) { //3
			return nil, fmt.Errorf("item type error %d", k), int32(cmd.ErrorCode_ItemTypeError)
		}
		if itemCfg.Quality > max {
			return nil, fmt.Errorf("item quality is limit %d", k), int32(cmd.ErrorCode_CampFoodItemQualityLimit)
		}
	}
	if req.FoodId < 0 || (req.FoodId == 0 && (total != DEFAULT_FOOD_COST_NUM)) {
		return nil, fmt.Errorf("invalid parameter"), int32(cmd.ErrorCode_InvalidParam)
	}
	// 奖励
	rewardId := int32(0)
	rewardNum := int32(0)
	if req.FoodId > 0 {
		// 食谱是否解锁
		if !h.IsUnLockFood(req.FoodId, commonParams) {
			return nil, fmt.Errorf("food not unlock %d", req.FoodId), int32(cmd.ErrorCode_CampFoodNotUnlock)
		}
		// 指定食物制造
		foodCfg := excel.GetFoodMgr().GetById(req.FoodId)
		if foodCfg == nil || len(foodCfg.Cost) == 0 {
			return nil, fmt.Errorf("food cfg not found %d", req.FoodId), int32(cmd.ErrorCode_NotFoundConfig)
		}
		// 计算可以制造的产物数量
		tempArr := make([]int32, 0)
		for id, num := range foodCfg.Cost {
			tempArr = append(tempArr, costs[id]/num)
		}

		maxNum := myUtils.GetArrayMinElement(tempArr)
		if maxNum <= 0 {
			return nil, fmt.Errorf("invalid param"), int32(cmd.ErrorCode_InvalidParam)
		}

		rewardId = req.FoodId
		rewardNum = maxNum
	} else {
		// 匹配食物制造
		tempId := int32(DEFAULT_FOOD_ID)
		excel.GetFoodMgr().Foreach(func(cfg *excel.FoodCfg) bool {
			if myUtils.CompareSameMap(costs, cfg.Cost) {
				tempId = cfg.Id
				return false
			}
			return true
		}, false)
		if !h.IsUnLockFood(tempId, commonParams) {
			// 解锁新食谱
			commonParams.camp.Foods.UnlockFoods = append(commonParams.camp.Foods.UnlockFoods, tempId)
			commonParams.camp.Foods.IsNew = append(commonParams.camp.Foods.IsNew, tempId)
			h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{Foods: commonParams.camp.Foods})
		}
		rewardId = tempId
		rewardNum = 1
	}

	//检查道具是否充足
	if !h.consumerItem(in.UserId, costs, h.actor.comData, common.CR_Camp_Building_FoodSupply) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}
	reward := map[int32]int32{rewardId: rewardNum}
	var extraDropChange *cmd.DropChange
	//额外产物
	extraAward := h.DoubleProduct(rewardId, rewardNum, commonParams)
	if len(extraAward) > 0 {
		var err error
		//发放额外奖励
		extraDropChange, err = GetDropMgr(h.actor).DropList2(extraAward, true, nil, h.actor.comData, common.CR_Camp_Building_Get)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 发奖励
	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Camp_Building_Get)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 发布事件
	myUtils.MergeItems(reward, extraAward)
	if err = h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_REWARD, []int32{TASK_TYPE_312, TASK_TYPE_510}, map[string]interface{}{
		"buildId": commonParams.building.ItemId,
		"reward":  reward,
	})); err != nil {
		h.Error(err)
	}

	threading.RunSafe(func() {
		build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
		e := &taptap.CampMakeFood{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                            //建筑唯一id
			BuildingId:        commonParams.buildingId,             //建筑id
			Lv:                commonParams.building.BuildingLevel, //建筑等级
			Cost:              taptap.ConvertMap2Str(costs),        //消耗材料
			Reward:            taptap.ConvertMap2Str(reward),       //食物产出
		}
		taptap.WriteDataLog(taptap.LogType_CampMakeFood, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	res := &cmd.LS2C_PlayerCampMakeFoodRes{
		BuildingId:      req.BuildingId,
		DropChange:      dropChange,
		ExtraDropChange: extraDropChange,
		CommonData:      h.actor.comData.FixDownComData(),
	}
	return res, nil, 0
}

/////////////////////////////////////////////////// 红点相关

// TriggerLifeSkill 判断是否可以触发生活技能
func (h *CampHandler) TriggerLifeSkill(building *cmd.PPlayerCampDecorationBuilding) *excel.LifeSkillCfg {
	if building == nil {
		return nil
	}

	if building.CardId == 0 {
		return nil
	}

	// 更具buildId 找到可以使用的生活技能
	lifeSkillCfgs := h.LifeSkill[building.ItemId]
	if len(lifeSkillCfgs) == 0 {
		return nil
	}

	//获取卡牌的配置信息
	cardCfg := h.actor.CardHandler.GetCardCfg(building.CardId)
	if cardCfg == nil {
		h.Debug("LifeSkillAddProduct get cardCfg is nil", building.CardId)
		return nil
	}

	//判断卡片的生活技能是否能作用域改建筑
	var lifeSkillCfg *excel.LifeSkillCfg
	var ok bool
	lifeSkillCfg, ok = lifeSkillCfgs[cardCfg.LifeskillID]
	if !ok {
		return nil
	}
	return lifeSkillCfg
}

// LifeSkillAddProduct 生活技能增加收益
func (h *CampHandler) LifeSkillAddProduct(building *cmd.PPlayerCampDecorationBuilding, reward map[int32]int32) map[int32]int32 {
	extraAward := make(map[int32]int32, 0)
	lifeSkillCfg := h.TriggerLifeSkill(building)
	if lifeSkillCfg == nil {
		return extraAward
	}
	switch lifeSkillCfg.Effect {
	case common.Building_Product_Add: //建筑产量提升
		for id, value := range reward {
			rate := float32(lifeSkillCfg.Para) / float32(100)
			extraAward[id] = int32(float32(value) * float32(rate))
			h.Debugf("建筑[%d],产物[%d],产量[%d],产量提升[%d]", building.ItemId, id, value, float32(value)*float32(rate), int32(float32(value)*float32(rate)))
		}
	case common.Produce_Double: //生产双倍产出，针对一次制造
		// rand.Seed(time.Now().UnixNano())
		r := rand.Intn(100)
		if int32(r) <= lifeSkillCfg.Para {
			for id, value := range reward {
				h.Debugf("建筑[%d],产物[%d],双倍产出的概率[%d]", building.ItemId, id, r)
				extraAward[id] = value
			}
		}
	}
	return extraAward
}

func (h *CampHandler) LifeSkillRate(building *cmd.PPlayerCampDecorationBuilding) int32 {
	lifeSkillCfg := h.TriggerLifeSkill(building)
	if lifeSkillCfg == nil {
		return 0
	}
	switch lifeSkillCfg.Effect {
	case common.Building_Product_Add: //建筑产量提升
		return lifeSkillCfg.Para
	}
	return 0
}

// LifeSkillSubCost 生活技能减少消耗
func (h *CampHandler) LifeSkillSubCost(building *cmd.PPlayerCampDecorationBuilding, reward map[int32]int32, totalCostSeconds int64) (map[int32]int32, int64) {
	lifeSkillCfg := h.TriggerLifeSkill(building)
	if lifeSkillCfg == nil {
		return reward, totalCostSeconds
	}
	switch lifeSkillCfg.Effect {
	case common.Produce_Power_Cost_Sub: //生产消耗电力减少
		for id, value := range reward {
			if id != common.CURRENCY_ITEM_ID_2001 {
				return reward, totalCostSeconds
			}
			rate := float32(100-lifeSkillCfg.Para) / float32(100)
			reward[id] = int32(float32(value) * float32(rate))
			h.Debugf("建筑[%d],消耗[%d],原来电力[%d],生产消耗电力减少[%d],%d", building.ItemId, id, value, float32(value)*float32(rate), int32(float32(value)*float32(rate)))
		}
	case common.Produce_Time_Cost_Sub: // 生产耗时减少
		rate := float32(100-lifeSkillCfg.Para) / float32(100)
		totalCostSeconds = int64(float32(totalCostSeconds) * float32(rate))
		h.Debugf("建筑[%d],原来时间[%d],生产耗时减少[%d],%d", building.ItemId, totalCostSeconds, float32(totalCostSeconds)*float32(rate), totalCostSeconds)
	}
	return reward, totalCostSeconds
}

// 处理营地红点已读
func (h *CampHandler) handleRedPoint(commonData *clidto.Comdata, foodIds []int64) error {

	camp := h.getCurCamp()
	if camp == nil {
		return fmt.Errorf("cur camp is nil")
	}

	isNew := make([]int32, len(camp.Foods.IsNew))
	copy(isNew, camp.Foods.IsNew)
	for _, id := range foodIds {
		for i := 0; i < len(isNew); i++ {
			if isNew[i] == int32(id) {
				isNew = append(isNew[:i], isNew[i+1:]...)
			}
		}
	}

	camp.Foods.IsNew = isNew
	commonData.GetCampData().Camp = append(commonData.GetCampData().Camp, &cmd.PPlayerCamp{Foods: camp.Foods})
	if err := h.SaveDB(); err != nil {
		return err
	}

	return nil
}

/////////////////////////////////////////////////////////内部方法调用

// 目前初始营地有4个布局，如果某个账号缺少布局，在这里打补丁补上
func (h *CampHandler) patchCampLayout() {
	saveFlag := false
	campMap := h.actor.GetCampData().Camp
	for _, v := range campMap {
		for i := 1; i <= LAYOUT_MAXIMUM; i++ {
			layout, ok := v.Layout[int32(i)]
			if !ok {
				v.Layout[int32(i)] = &cmd.PPlayerCampServerLayout{
					LayoutId:        int32(i),
					LayoutName:      "",
					AtmosphereValue: 0,
					Building:        make(map[int64]*cmd.PPlayerCampCommonBuilding),
				}
				saveFlag = true
			} else if layout.Building == nil {
				layout.Building = make(map[int64]*cmd.PPlayerCampCommonBuilding)
			}
		}
		//修正光和树数据
		if v.LightingComposeTree != nil && v.LightingComposeTree.BuildingId > 0 {
			if v.LightingComposeTree.EndTimestampList == nil {
				builds := h.actor.GetCampData().DecorationBuilding
				building, ok := builds[v.LightingComposeTree.BuildingId]
				if ok {
					buildLevelConfig := excel.GetBuildingLevelMgr().GetById(building.ItemId*100 + building.BuildingLevel)
					formulaCfg := h.getFormulaByCfg(buildLevelConfig, false)
					endTimeStamp := make([]*cmd.ComposeTreeProductEndTime, 0)
					for _, vv := range formulaCfg {
						for _, p := range vv.ItemProduct {
							endTimeStamp = append(endTimeStamp, &cmd.ComposeTreeProductEndTime{
								ItemId:       p.ItemId,
								EndTimestamp: v.LightingComposeTree.EndTimestamp,
							})
						}
					}
					v.LightingComposeTree.EndTimestampList = endTimeStamp
				}
				saveFlag = true
			}
		}
		//修正营地扩建
		if v.BlockMaxTimes == 0 {
			v.BlockMaxTimes = h.initBlockMaxTimes()
			saveFlag = true
		}
	}
	//修正光和树数据

	if saveFlag {
		h.SaveDB()
	}
}

func (h *CampHandler) buildCampList() *cmd.PPlayerCampList {
	h.patchCampLayout()
	campList := &cmd.PPlayerCampList{}
	campList.Camp = make([]*cmd.PPlayerCamp, 0)
	campList.CurrentCampId = h.getCurrentCampId()
	campList.CurrentLayoutId = h.getCurrentLayoutId()
	campList.HomeCoinStartTime = h.GetHomeCoinStartTime()

	for k := range h.actor.GetCampData().BuildingUnlockList {
		campList.BuildingUnlockList = append(campList.BuildingUnlockList, k)
	}

	for _, building := range h.actor.GetCampData().DecorationBuilding {
		campList.DecorationBuilding = append(campList.DecorationBuilding, building)
	}

	now := time.Now()
	for _, v := range h.actor.GetCampData().Camp {
		oneCamp := &cmd.PPlayerCamp{CampId: v.CampId, RoleList: make([]int32, 0)}
		if v.LightingComposeTree != nil {
			oneCamp.LightingComposeTree = v.LightingComposeTree
		}
		if v.Trader != nil {
			// 尝试刷新数据
			if !common.IsSameDayByOffset(time.Unix(v.Trader.Refresh, 0), now, common.GAME_DAILY_REFRESH_HOUR) {
				v.Trader.Items = h.TryRefreshTraderList(v.Trader.Level, nil)
				v.Trader.Refresh = now.Unix()
			}
			oneCamp.Trader = v.Trader
			oneCamp.Trader.NextTime = common.GetNextDailyRefreshTime()
		}
		if v.Foods != nil {
			oneCamp.Foods = v.Foods
		}
		// 顺便初始化，防止空指针
		if v.RoleList == nil {
			v.RoleList = make([]int32, 0)
		}
		for _, roleId := range v.RoleList {
			oneCamp.RoleList = append(oneCamp.RoleList, roleId)
		}
		layouts := make([]*cmd.PPlayerCampClientLayout, 0)
		if v.Layout != nil {
			//布局打补丁
			for _, layout := range v.Layout {
				if layout.ThemeId == 0 {
					layout.ThemeId = 90189
					themeCfg := excel.GetBuildMainMgr().GetById(90189)
					if themeCfg == nil {
						h.Warnf("build main config not found, id: %d", 90189)
					}
					layout.AtmosphereValue += themeCfg.Ambience
				}
			}

			for layoutId, layout := range v.Layout {
				layouts = append(layouts, ServerLayout2ClientLayout(layoutId, layout, false))
			}
			oneCamp.Layout = layouts
		}
		campList.Camp = append(campList.Camp, oneCamp)
		for _, queue := range v.WorkQueue {
			campList.WorkQueue = append(campList.WorkQueue, queue)
		}
		oneCamp.BlockMaxTimes = v.BlockMaxTimes
	}
	if err := h.SaveDB(); err != nil {
		h.Error(err)
	}

	return campList
}

func (h *CampHandler) GetHomeCoinStartTime() int64 {
	camp := h.getCurCamp()
	if camp == nil {
		return 0
	}
	return camp.HomeCoinStartTime
}

func (h *CampHandler) consumerItem(uid string, cost map[int32]int32, commonData *clidto.Comdata, reason common.ChangeReason) bool {
	if !GetConsumeMgr(h.actor).CheckMapEnough(cost) {
		return false
	}
	err := GetConsumeMgr(h.actor).ConsumeList(cost, commonData, reason)
	if err != nil {
		return false
	}
	return true
}

// 公共检查
func (h *CampHandler) commonCheck(outPutParams *OutputParams) (error, int32) {

	// 玩家必然有一个营地数据，不会返回nil
	outPutParams.camp = h.getCurCamp()
	outPutParams.layout = h.getCurLayout()
	if !outPutParams.isFunctionBuilding {
		return nil, 0
	}
	builds := h.actor.GetCampData().DecorationBuilding
	building, ok := builds[outPutParams.buildingId]
	if !ok || building.BuildingId == 0 {
		return fmt.Errorf("building not found"), int32(cmd.ErrorCode_CampBuildingNotExist)
	}

	build := excel.GetBuildMainMgr().GetById(building.ItemId)
	if build == nil {
		return fmt.Errorf("buildmain config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	if err, code := checkBuildType(outPutParams.messageId, build.GetBuildType()); code != int32(cmd.ErrorCode_Success) {
		return err, code
	}
	nextLevelId := building.ItemId*100 + building.BuildingLevel + outPutParams.incrLevel

	buildLevelConfig := excel.GetBuildingLevelMgr().GetById(nextLevelId)
	if buildLevelConfig == nil {
		return fmt.Errorf("build level config not found, levelId: %d", nextLevelId), int32(cmd.ErrorCode_NotFoundConfig)
	}
	outPutParams.buildType = build.GetBuildType()
	outPutParams.buildLevelConfig = buildLevelConfig
	outPutParams.building = building

	return nil, 0
}

// 批量添加家具，同类型家具
func (h *CampHandler) batchAddDecorationBuilding(id int32, itemNum uint32, commonData *clidto.Comdata) error {
	//获取现有数量
	num := uint32(h.GetBuildingNum(id))
	// 获取配置表里的最大限制
	limit := GetItemLimit(id)
	addValue, mailValue := itemNum, uint32(0)
	if num+itemNum > limit {
		addValue = limit - num
		mailValue = num + itemNum - limit
	}

	if mailValue > 0 {
		return errors.New("已达持有最大数量")
		//if err := h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_2, map[int32]int32{id: int32(mailValue)}, commonData); err != nil {
		//	return err
		//}
	}

	buildings, err := h.batchAddDecorationBuildingInternal(id, addValue)
	if err != nil {
		return err
	}

	commonData.GetCampData().DecorationBuilding = append(commonData.GetCampData().DecorationBuilding, buildings...)
	return nil
}

// ConvertBuilding 转换家具
func (h *CampHandler) ConvertBuilding(quality int32) (int32, int32) {
	//更具道具的品质获取要转换的道具
	exchange := GetCampExchange(quality)
	if exchange == nil {
		h.Debug("ConvertBuilding get camp exchange is err:", quality)
		return 0, 0
	}
	return exchange.GetKey(), exchange.GetVal()

}

// @Description: 获取建筑等级解锁的效果值, UpgradeChange字段类型4
// @receiver h
// @param levelCfg
// @return int
func (h *CampHandler) getUpgradeChangeValue4(levelCfg *excel.BuildingLevelCfg) int32 {

	if levelCfg == nil {
		return 0
	}
	var max int32
	excel.GetBuildingLevelMgr().Foreach(func(cfg *excel.BuildingLevelCfg) bool {
		if cfg.BuildId == levelCfg.BuildId && cfg.Level <= levelCfg.Level && cfg.UpgradeChange[LEVEL_UPGRADE_CHANGE_4] > max {
			max = cfg.UpgradeChange[LEVEL_UPGRADE_CHANGE_4]
		}
		return true
	}, true)

	h.Debugf("getUpgradeChangeValue levelCfg:%+v,max:%d", levelCfg, max)
	return max
}

// @Description: 获取建筑等级解锁的效果值, UpgradeChange字段类型1
// @receiver h
// @param levelCfg
// @return int
func (h *CampHandler) getUpgradeChangeValue(levelCfg *excel.BuildingLevelCfg) int {

	if levelCfg == nil {
		return 0
	}
	var max int32
	excel.GetBuildingLevelMgr().Foreach(func(cfg *excel.BuildingLevelCfg) bool {
		if cfg.BuildId == levelCfg.BuildId && cfg.Level <= levelCfg.Level && cfg.UpgradeChange[LEVEL_UPGRADE_CHANGE_1] > max {
			max = cfg.UpgradeChange[LEVEL_UPGRADE_CHANGE_1]
		}
		return true
	}, true)

	h.Debugf("getUpgradeChangeValue levelCfg:%+v,max:%d", levelCfg, max)
	return int(max)
}

// 获取指定建筑的建造公式配置，不会返回nil，至少返回一个长度为0的map
func (h *CampHandler) getFormulaByCfg(cfg *excel.BuildingLevelCfg, needUnion bool) map[int32]*excel.ItemSynthesisCfg {

	formulaMap := make(map[int32]*excel.ItemSynthesisCfg)
	if cfg.Function == nil {
		return formulaMap
	}
	var list []int32
	if needUnion {
		list = unionFormulaByLvCfg(cfg)
	} else {
		list = cfg.Function
	}
	itemSynthesisMgr := excel.GetItemSynthesisMgr()
	for _, v := range list {
		itemSynthesisCfg := itemSynthesisMgr.GetById(v)
		if itemSynthesisCfg == nil {
			h.Warnf("CampFurnaceOpReq FormulaId not found, ItemSynthesisId:%d", v)
			return make(map[int32]*excel.ItemSynthesisCfg)
		}
		formulaMap[itemSynthesisCfg.Id] = itemSynthesisCfg
	}
	return formulaMap
}

// @Description: 尝试刷新商人清单
// @param level 当前商人等级
// @param curList 当前的清单列表
// @return []*cmd.PPlayerCampTraderList 新的清单列表
func (h *CampHandler) TryRefreshTraderList(level int32, curList []*cmd.PPlayerCampTraderList) []*cmd.PPlayerCampTraderList {
	newList := make([]*cmd.PPlayerCampTraderList, 0)
	levelCfg := excel.GetProfiteerLevelMgr().GetById(level)
	if levelCfg == nil {
		return curList
	}
	// 构建临时map
	curMap := make(map[int32]*cmd.PPlayerCampTraderList)
	for _, v := range curList {
		curMap[v.Id] = v
	}

	var index int32
	// 固定池处理 默认1个
	if v, ok := curMap[index]; !ok || (ok && v.Status == TRADER_STATUS_1) {
		ret := buildTraderItem(index, TRADER_TYPE_1, levelCfg.FixedPool)
		if ret != nil {
			newList = append(newList, ret)
		}
	} else {
		newList = append(newList, v)
	}
	index++

	// 杂货池处理 默认3个
	for i := 0; i < 3; i++ {
		if curV, curOK := curMap[index]; !curOK || (curOK && curV.Status == TRADER_STATUS_1) {
			// 随机出品质
			m := make(map[interface{}]int32)
			for k, v := range levelCfg.RandomPool {
				m[k] = v
			}
			t := myUtils.RandomMap(m)
			if quality, ok := t.(int32); ok {
				temp := buildTraderItem(index, TRADER_TYPE_2, quality)
				if temp != nil {
					newList = append(newList, temp)
				}
			}
		} else {
			newList = append(newList, curV)
		}
		index++
	}
	h.Debugf("营地商人清单：%+v", newList)
	return newList
}

// IsUnLockFood 是否解锁食谱
func (h *CampHandler) IsUnLockFood(foodId int32, commonParams *OutputParams) bool {
	for _, v := range commonParams.camp.Foods.UnlockFoods {
		if v == foodId {
			return true
		}
	}
	return false
}

func (h *CampHandler) DoubleProduct(rewardId, rewardNum int32, commonParams *OutputParams) map[int32]int32 {
	extraAward := make(map[int32]int32, 0)
	for i := int32(0); i < rewardNum; i++ {
		tempMap := map[int32]int32{rewardId: 1}
		//检测时候有增益
		extraTemp := h.LifeSkillAddProduct(commonParams.building, tempMap)
		for k, v := range extraTemp {
			if value, ok := extraAward[k]; ok {
				extraAward[k] = value + v
			} else {
				extraAward[k] = v
			}
		}
	}
	return extraAward
}
func (h *CampHandler) CheckMapLimit(reward, extraReward map[int32]int32) bool {
	newReward := make(map[int32]int32, len(reward)+len(extraReward))
	myUtils.MergeItems(newReward, reward)
	myUtils.MergeItems(newReward, extraReward)
	return GetDropMgr(h.actor).CheckMapLimit(newReward)
}

func buildTraderItem(index, typ, quality int32) *cmd.PPlayerCampTraderList {
	cfg := getPoolCfg(typ, quality)
	if cfg == nil {
		return nil
	}
	// 随机消耗
	var temp []int32
	for k := range cfg.CostItem {
		temp = append(temp, k)
	}
	costs := make([]*cmd.KeyValueItem, 0)
	for i := int32(0); i < cfg.CostNum; i++ {
		k := myUtils.RandomList(temp)
		costs = append(costs, &cmd.KeyValueItem{
			Key:   k,
			Value: cfg.CostItem[k],
		})
	}
	// 随机奖励
	var temp2 []int32
	for k := range cfg.Reward {
		temp2 = append(temp2, k)
	}
	rewardKey := myUtils.RandomList(temp2)
	// 构建数据
	return &cmd.PPlayerCampTraderList{
		Id:       index,
		Category: typ,
		Status:   TRADER_STATUS_1,
		Quality:  quality,
		Costs:    costs,
		Reward:   &cmd.KeyValueItem{Key: rewardKey, Value: cfg.Reward[rewardKey]},
	}
}

func getPoolCfg(typ, quality int32) *excel.ProfitterPoolCfg {
	var ret *excel.ProfitterPoolCfg
	excel.GetProfitterPoolMgr().Foreach(func(cfg *excel.ProfitterPoolCfg) bool {
		if cfg.Type == typ && cfg.Quality == quality {
			ret = cfg
			return false
		}
		return true
	}, false)
	return ret
}

func canGetReward(queue *cmd.PPlayerCampFunctionBuildingWorkQueue) (bool, map[int32]int32) {
	reward := map[int32]int32{}
	itemSynthesisMgr := excel.GetItemSynthesisMgr()
	itemMgr := excel.GetItemMgr()
	for _, v := range queue.Formula {
		cfg := itemSynthesisMgr.GetById(v.FormulaId)
		if cfg == nil {
			return false, nil
		}
		for _, vp := range cfg.GetItemProduct() {
			itemCfg := itemMgr.GetById(vp.ItemId)
			if itemCfg == nil {
				return false, nil
			}
			itemId, itemNum := vp.ItemId, v.Num*vp.Num

			reward[itemId] += itemNum
		}
	}
	return true, reward
}

// 兼容低等级公式
func unionFormulaByLvCfg(levelCfg *excel.BuildingLevelCfg) []int32 {
	list := make([]int32, 0, 4)
	excel.GetBuildingLevelMgr().Foreach(func(cfg *excel.BuildingLevelCfg) bool {
		if cfg.Id <= levelCfg.Id && cfg.BuildId == levelCfg.BuildId {
			list = append(list, cfg.Function...)
		}
		return true
	}, true)
	return list
}

// 获取权重map
func getEquipRandomMap(formulaCfg *excel.ItemSynthesisCfg) (bool, map[int32]*EquipItemIdMap) {
	allCfgMap := make(map[int32]*EquipItemIdMap)
	equipmentBlueprintMgr := excel.GetEquipmentBlueprintMgr()
	equipmentPoolMgr := excel.GetEquipmentPoolMgr()
	for _, v := range formulaCfg.ItemCost {
		bluCfg := equipmentBlueprintMgr.GetById(v.ItemId)
		if bluCfg == nil {
			return false, allCfgMap
		}

		cfgMap := make(map[interface{}]int32)
		equipmentPoolMgr.Foreach(func(cfg *excel.EquipmentPoolCfg) bool {
			if cfg.PoolId == bluCfg.PoolId {
				cfgMap[cfg.EquipmentId] = cfg.Weight
			}
			return true
		}, true)
		if len(cfgMap) == 0 {
			return false, allCfgMap
		}
		allCfgMap[bluCfg.PoolId] = &EquipItemIdMap{cfg: cfgMap}
	}
	return true, allCfgMap
}

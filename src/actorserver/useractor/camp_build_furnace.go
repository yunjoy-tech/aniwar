package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"
	"time"
)

// Furnace 熔炉
type Furnace struct {
	BaseBuilding
}

func NewFurnace() IBuilding {
	return &Furnace{
		BaseBuilding: BaseBuilding{},
	}
}

func (lt *Furnace) Cost(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams) (map[int32]int32, int32) {
	//消耗物
	cost := map[int32]int32{}
	for _, v := range items {
		cfg, ok := formulaMap[v.FormulaId]
		if !ok {
			return nil, int32(cmd.ErrorCode_CampFormulaNotExist)
		}
		// 制造数量超出上限
		if cfg.MaxnumberItem < v.Num || v.Num == 0 {
			return nil, int32(cmd.ErrorCode_CampFormulaParamError)
		}
		for _, itemCost := range cfg.ItemCost {
			cost[itemCost.ItemId] += itemCost.Num * v.Num
		}
	}

	// 2.cost
	// 数量上限检查
	var total int32
	for _, v := range cost {
		total += v
	}
	if total > commonParams.buildLevelConfig.EleLimit {
		return nil, int32(cmd.ErrorCode_CampItemNumUpLimit)
	}

	return cost, 0
}

func (lt *Furnace) Product(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, h *CampHandler, commonParams *OutputParams, res *cmd.LS2C_PlayerCampBuildFunOpRes) (interface{}, int32) {
	workQueue := h.getWorkQueueBuilding(commonParams.buildingId)
	var totalCostSeconds int64
	fWorkQueue := &cmd.PPlayerCampFunctionBuildingWorkQueue{QueueId: utils.GenIntGUID()}
	for _, v := range items {
		cfg, ok := formulaMap[v.FormulaId]
		if !ok {
			return "", int32(cmd.ErrorCode_CampFormulaNotExist)
		}
		// 制造数量超出上限
		if cfg.MaxnumberItem < v.Num || v.Num == 0 {
			return "", int32(cmd.ErrorCode_CampFormulaParamError)
		}
		totalCostSeconds += int64(cfg.TimeCost) * int64(v.Num)
		fWorkQueue.Formula = append(fWorkQueue.Formula, &cmd.PPlayerCampFunctionBuildingFormula{FormulaId: v.FormulaId, Num: v.Num})
	}
	//生活技能减少耗时
	_, totalCostSeconds = h.LifeSkillSubCost(commonParams.building, nil, totalCostSeconds)

	fWorkQueue.StartTimestamp = time.Now().Unix()
	fWorkQueue.EndTimestamp = fWorkQueue.StartTimestamp + int64(float64(totalCostSeconds))
	fWorkQueue.Rate = h.LifeSkillRate(commonParams.building)

	if workQueue != nil {
		workQueue.Queue = append(workQueue.Queue, fWorkQueue)
	} else {
		if commonParams.camp.WorkQueue == nil {
			commonParams.camp.WorkQueue = map[int64]*cmd.PPlayerCampBuildingWorkQueue{}
		}
		commonParams.camp.WorkQueue[commonParams.buildingId] = &cmd.PPlayerCampBuildingWorkQueue{
			BuildingId:    commonParams.buildingId,
			Queue:         []*cmd.PPlayerCampFunctionBuildingWorkQueue{fWorkQueue},
			BuildConfigId: commonParams.buildLevelConfig.BuildId,
		}
	}
	// 4.save
	h.SaveDB()

	res.Queue = fWorkQueue
	return fWorkQueue, int32(cmd.ErrorCode_Success)

}

func (lt *Furnace) Check(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams, h *CampHandler) int32 {

	workQueue := h.getWorkQueueBuilding(commonParams.buildingId)
	if workQueue != nil && len(workQueue.Queue) >= h.getUpgradeChangeValue(commonParams.buildLevelConfig) {
		return int32(cmd.ErrorCode_CampWorkQueueIsFull)
	}

	return int32(cmd.ErrorCode_Success)
}

func (lt *Furnace) DataLog(commonParams *OutputParams, h *CampHandler, buildingId int64, costs string, formula interface{}) {
	fWorkQueue := formula.(*cmd.PPlayerCampFunctionBuildingWorkQueue)
	// 熔炉熔炼 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampFurnaceoPerate{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampFurnaceoPerate, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                                         //建筑唯一id
	//		BuildingId:     buildingId,                                       //建筑id
	//		Lv:             commonParams.building.BuildingLevel,              //建筑等级
	//		Formula:        lilith.ConvertListStruct2Str(fWorkQueue.Formula), //熔炼产出材料
	//		StartTs:        fWorkQueue.StartTimestamp,                        //队列开始时间戳
	//		EndTs:          fWorkQueue.EndTimestamp,                          //队列结束时间戳
	//		Costs:          costs,                                            //建造消耗材料
	//		QueueId:        fWorkQueue.QueueId,                               //队列id myUtils.ConvertItem2(
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampFurnaceoPerate{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                                         //建筑唯一id
			BuildingId:        buildingId,                                       //建筑id
			Lv:                commonParams.building.BuildingLevel,              //建筑等级
			Formula:           taptap.ConvertListStruct2Str(fWorkQueue.Formula), //熔炼产出材料
			StartTs:           fWorkQueue.StartTimestamp,                        //队列开始时间戳
			EndTs:             fWorkQueue.EndTimestamp,                          //队列结束时间戳
			Costs:             costs,                                            //建造消耗材料
			QueueId:           fWorkQueue.QueueId,                               //队列id myUtils.ConvertItem2(
		}
		taptap.WriteDataLog(taptap.LogType_CampFurnaceoPerate, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
}

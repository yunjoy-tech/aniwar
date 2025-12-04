package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
)

// EquipmentFoundry 装备锻造
type EquipmentFoundry struct {
	BaseBuilding
}

func NewEquipmentFoundry() IBuilding {
	return &EquipmentFoundry{
		BaseBuilding: BaseBuilding{},
	}
}

func (lt *EquipmentFoundry) Cost(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams) (map[int32]int32, int32) {
	if len(items) <= 0 {
		return nil, int32(cmd.ErrorCode_InvalidParam)
	}
	//消耗物
	formulaCfg, ok := formulaMap[items[0].GetFormulaId()]
	if !ok {
		return nil, int32(cmd.ErrorCode_InvalidParam)
	}
	totalCost := map[int32]int32{}
	equipmentBlueprintMgr := excel.GetEquipmentBlueprintMgr()
	for _, v := range formulaCfg.ItemCost {
		totalCost[v.ItemId] = v.Num * items[0].GetNum()
		cfg := equipmentBlueprintMgr.GetById(v.ItemId)
		if cfg == nil {
			return nil, int32(cmd.ErrorCode_NotFoundConfig)
		}
		for _, cost := range cfg.Cost {
			totalCost[cost.ItemId] += cost.Num * items[0].GetNum()
		}
	}
	return totalCost, int32(cmd.ErrorCode_Success)
}

func (lt *EquipmentFoundry) Product(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, h *CampHandler, commonParams *OutputParams, res *cmd.LS2C_PlayerCampBuildFunOpRes) (interface{}, int32) {
	if len(items) <= 0 {
		return "", int32(cmd.ErrorCode_InvalidParam)
	}
	//消耗物
	formulaCfg, ok := formulaMap[items[0].GetFormulaId()]
	if !ok {
		return "", int32(cmd.ErrorCode_InvalidParam)
	}

	ok, allCfgMap := getEquipRandomMap(formulaCfg)
	if !ok {
		return "", int32(cmd.ErrorCode_NotFoundConfig)
	}

	equipIds := make([]int32, 0, 4) // 装备唯一id列表
	dropChanges := &cmd.DropChange{}
	count := 0
	for i := 0; i < int(items[0].GetNum()); i++ {
		for _, v := range allCfgMap {
			equipList := make(map[int32]int32)
			target := myUtils.RandomMap(v.cfg)
			baseId := target.(int32)
			equipList[baseId] = 1
			equipIds = append(equipIds, baseId)
			dropChange, err := GetDropMgr(h.actor).DropList2(equipList, true, nil, h.actor.comData, lt.GetChangeReason())
			if err != nil {
				return nil, int32(cmd.ErrorCode_InternalError)
			} else {
				dropChanges.Items = append(dropChanges.Items, dropChange.Items...)
			}
			count++
		}

	}
	//dropChange, err := GetDropMgr(h.actor).DropList2(equipList, true, nil, h.actor.comData, lt.GetChangeReason())
	//if err != nil {
	//	return nil, int32(cmd.ErrorCode_InternalError)
	//}
	res.DropChange = dropChanges

	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_EQUIP_BUILD, []int32{TASK_TYPE_203}, map[string]interface{}{
		"count": int32(count),
	}))
	if errx != nil {
		h.Error(errx)
	}

	return equipIds, int32(cmd.ErrorCode_Success)
}

func (lt *EquipmentFoundry) GetChangeReason() common.ChangeReason {
	return common.CR_Camp_Equip_Fuoundry
}

func (lt *EquipmentFoundry) Check(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams, h *CampHandler) int32 {

	for _, v := range items {
		formulaCfg, ok := formulaMap[v.GetFormulaId()]
		if !ok {
			return int32(cmd.ErrorCode_NotFoundConfig)
		}

		//check
		if v.GetNum() > formulaCfg.MaxnumberItem {
			return int32(cmd.ErrorCode_CampFormulaParamError)
		}
	}

	return int32(cmd.ErrorCode_Success)
}

func (lt *EquipmentFoundry) DataLog(commonParams *OutputParams, h *CampHandler, buildingId int64, costs string, formula interface{}) {
	equipIds := formula.([]int32)
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampequipFoundry{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampequipFoundry, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                            //建筑唯一id
	//		BuildingId:     buildingId,                          //建筑id
	//		Lv:             commonParams.building.BuildingLevel, //建筑等级
	//		Formula:        costs,                               //消耗材料
	//		Equips:         lilith.ConvertList2Str(equipIds),    //装备唯一id列表
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampequipFoundry{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                            //建筑唯一id
			BuildingId:        buildingId,                          //建筑id
			Lv:                commonParams.building.BuildingLevel, //建筑等级
			Formula:           costs,                               //消耗材料
			Equips:            taptap.ConvertList2Str(equipIds),    //装备唯一id列表
		}
		taptap.WriteDataLog(taptap.LogType_CampequipFoundry, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
}

package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
)

// MaterialInstitute 材料研究所
type MaterialInstitute struct {
	BaseBuilding
}

func NewMaterialInstitute() IBuilding {
	return &MaterialInstitute{
		BaseBuilding: BaseBuilding{},
	}
}
func (lt *MaterialInstitute) Cost(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams) (map[int32]int32, int32) {
	totalCost := make(map[int32]int32, 0)
	for _, item := range items {
		cfg := formulaMap[item.GetFormulaId()]
		if cfg == nil {

		}
		for _, v := range formulaMap[item.GetFormulaId()].ItemCost {
			totalCost[v.ItemId] = v.Num * item.GetNum()
		}
	}
	return totalCost, int32(cmd.ErrorCode_Success)
}

func (lt *MaterialInstitute) Product(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, h *CampHandler, commonParams *OutputParams, res *cmd.LS2C_PlayerCampBuildFunOpRes) (interface{}, int32) {
	reward := make(map[int32]int32, 0)
	itemMgr := excel.GetItemMgr()
	extraAward := make(map[int32]int32, 0)
	for _, item := range items {
		formulaCfg := formulaMap[item.GetFormulaId()]
		tempAward := make(map[int32]int32, 0)
		for _, v := range formulaCfg.ItemProduct {
			itemCfg := itemMgr.GetById(v.ItemId)
			if itemCfg == nil {
				return "", int32(cmd.ErrorCode_NotFoundConfig)
			}
			reward[itemCfg.ItemId] += v.Num * item.GetNum()
			tempAward[itemCfg.ItemId] += v.Num * item.GetNum()
		}
		// 每一次的产物都双倍
		extraTemp := h.LifeSkillAddProduct(commonParams.building, tempAward)
		for k, v := range extraTemp {
			if value, ok := extraAward[k]; ok {
				extraAward[k] = value + v
			} else {
				extraAward[k] = v
			}
		}
	}
	// 生活节能增加产物
	for _, item := range items {
		formulaCfg := formulaMap[item.GetFormulaId()]
		tempAward := make(map[int32]int32, 0)
		for _, v := range formulaCfg.ItemProduct {
			itemCfg := itemMgr.GetById(v.ItemId)
			if itemCfg == nil {
				return "", int32(cmd.ErrorCode_NotFoundConfig)
			}
			tempAward[itemCfg.ItemId] += v.Num
		}
		// 每一次的产物都双倍
		for i := int32(0); i < item.GetNum(); i++ {
			extraTemp := h.LifeSkillAddProduct(commonParams.building, tempAward)
			for k, v := range extraTemp {
				if value, ok := extraAward[k]; ok {
					extraAward[k] = value + v
				} else {
					extraAward[k] = v
				}
			}
		}
	}

	// 发放产物
	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Camp_CampMaterial_Conversion)
	if err != nil {
		return nil, int32(cmd.ErrorCode_InternalError)
	}
	res.DropChange = dropChange

	// 发放额外奖励
	if len(extraAward) > 0 {
		extraDropChange, err := GetDropMgr(h.actor).DropList2(extraAward, true, nil, h.actor.comData, common.CR_Camp_CampMaterial_Conversion)
		if err != nil {
			return nil, int32(cmd.ErrorCode_InternalError)
		}
		res.ExtraDropChange = extraDropChange
	}

	return reward, int32(cmd.ErrorCode_Success)
}
func (lt *MaterialInstitute) GetChangeReason() common.ChangeReason {
	return common.CR_Camp_CampMaterial_Conversion
}

func (lt *MaterialInstitute) Check(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams, h *CampHandler) int32 {

	for _, v := range items {
		formulaCfg, ok := formulaMap[v.GetFormulaId()]
		if !ok {
			return int32(cmd.ErrorCode_NotFoundConfig)
		}
		if v.GetNum() > formulaCfg.MaxnumberItem {
			return int32(cmd.ErrorCode_CampFormulaParamError)
		}
	}
	return int32(cmd.ErrorCode_Success)
}

func (b *MaterialInstitute) DataLog(commonParams *OutputParams, h *CampHandler, buildingId int64, costs string, formula interface{}) {
	reward := formula.(map[int32]int32)
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampmaterialConver{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampmaterialConver, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                            //建筑唯一id
	//		BuildingId:     buildingId,                          //建筑id
	//		Lv:             commonParams.building.BuildingLevel, //建筑等级
	//		Formula:        costs,                               //消耗材料
	//		Reward:         lilith.ConvertMap2Str(reward),       //产出奖励
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampmaterialConver{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                            //建筑唯一id
			BuildingId:        buildingId,                          //建筑id
			Lv:                commonParams.building.BuildingLevel, //建筑等级
			Formula:           costs,                               //消耗材料
			Reward:            taptap.ConvertMap2Str(reward),       //产出奖励
		}
		taptap.WriteDataLog(taptap.LogType_CampmaterialConver, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
}

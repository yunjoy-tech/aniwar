package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// FoodSupplyStation 食物补给站
type FoodSupplyStation struct {
	BaseBuilding
}

func NewFoodSupplyStation() IBuilding {
	return &FoodSupplyStation{
		BaseBuilding: BaseBuilding{},
	}
}

func (lt *FoodSupplyStation) Build(commonParams *OutputParams, h *CampHandler, req *cmd.C2LS_PlayerCampMakeFunctionBuildingReq, buildLevelConfig *excel.BuildingLevelCfg) (*cmd.PPlayerCampCommonBuilding, error, int32) {
	//创建
	building := h.NewPPlayerCampCommonBuilding(req.GetX(), req.GetY(), req.GetParentId(), req.GetParentGridId(), req.GetEdge(), req.GetFlip(), nil)
	building.Building = h.NewPPlayerCampDecorationBuilding(req.ItemId, 1, h.actor.Data.Camp.CurrentCampId)
	if commonParams.layout.Building == nil {
		commonParams.layout.Building = map[int64]*cmd.PPlayerCampCommonBuilding{}
	}

	commonParams.camp.Foods = &cmd.PPlayerCampMakeFood{
		BuildingId:  building.Building.BuildingId,
		Level:       1,
		UnlockFoods: buildLevelConfig.UpgradeFood,
		IsNew:       buildLevelConfig.UpgradeFood,
	}
	commonParams.camp.Foods.UnlockFoods = append(commonParams.camp.Foods.UnlockFoods, DEFAULT_FOOD_ID)
	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{Foods: commonParams.camp.Foods})

	commonParams.layout.Building[building.Building.BuildingId] = building
	h.actor.GetCampData().DecorationBuilding[building.Building.BuildingId] = building.Building
	return building, nil, int32(cmd.ErrorCode_Success)
}
func (lt *FoodSupplyStation) LevelUp(commonParams *OutputParams, campRet *cmd.PPlayerCamp, h *CampHandler) (error, int32) {
	commonParams.camp.Foods.Level++
	// 判断升级解锁的食物，是否在已存在的解锁食谱中
	diff := make([]int32, 0)
	diff = utils.DiffArray(commonParams.camp.Foods.UnlockFoods, commonParams.buildLevelConfig.UpgradeFood)
	if len(diff) > 0 {
		commonParams.camp.Foods.UnlockFoods = append(commonParams.camp.Foods.UnlockFoods, diff...)
		commonParams.camp.Foods.IsNew = append(commonParams.camp.Foods.IsNew, diff...)
	}
	commonParams.building.BuildingLevel++
	campRet.Foods = commonParams.camp.Foods
	return nil, int32(cmd.ErrorCode_Success)
}

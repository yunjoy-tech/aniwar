package useractor

import (
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// BuildingFoundry  家具制造台
type BuildingFoundry struct {
	BaseBuilding
}

func NewBuildingFoundry() IBuilding {
	return &BuildingFoundry{
		BaseBuilding: BaseBuilding{},
	}
}

func (lt *BuildingFoundry) Build(commonParams *OutputParams, h *CampHandler, req *cmd.C2LS_PlayerCampMakeFunctionBuildingReq, buildLevelConfig *excel.BuildingLevelCfg) (*cmd.PPlayerCampCommonBuilding, error, int32) {
	//创建
	building := h.NewPPlayerCampCommonBuilding(req.GetX(), req.GetY(), req.GetParentId(), req.GetParentGridId(), req.GetEdge(), req.GetFlip(), nil)
	building.Building = h.NewPPlayerCampDecorationBuilding(req.ItemId, 1, h.actor.Data.Camp.CurrentCampId)
	if commonParams.layout.Building == nil {
		commonParams.layout.Building = map[int64]*cmd.PPlayerCampCommonBuilding{}
	}

	//家具制作台
	h.actor.comData.GetCampData().BuildingUnlockList = h.setAndNtfBuildingUnlockList(1)

	commonParams.layout.Building[building.Building.BuildingId] = building
	h.actor.GetCampData().DecorationBuilding[building.Building.BuildingId] = building.Building
	return building, nil, int32(cmd.ErrorCode_Success)

}
func (lt *BuildingFoundry) LevelUp(commonParams *OutputParams, campRet *cmd.PPlayerCamp, h *CampHandler) (error, int32) {
	// 家具制造台升级会解锁家具制造方式
	h.actor.comData.GetCampData().BuildingUnlockList = h.setAndNtfBuildingUnlockList(commonParams.building.BuildingLevel + 1)
	commonParams.building.BuildingLevel++
	return nil, int32(cmd.ErrorCode_Success)
}

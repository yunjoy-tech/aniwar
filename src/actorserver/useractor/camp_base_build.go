package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

//BaseBuilding 建筑基类
type BaseBuilding struct{}

func (b *BaseBuilding) Build(commonParams *OutputParams, h *CampHandler, req *cmd.C2LS_PlayerCampMakeFunctionBuildingReq, buildingLevelCfg *excel.BuildingLevelCfg) (*cmd.PPlayerCampCommonBuilding, error, int32) {
	//创建
	building := h.NewPPlayerCampCommonBuilding(req.GetX(), req.GetY(), req.GetParentId(), req.GetParentGridId(), req.GetEdge(), req.GetFlip(), nil)
	building.Building = h.NewPPlayerCampDecorationBuilding(req.ItemId, 1, h.actor.Data.Camp.CurrentCampId)

	if commonParams.layout.Building == nil {
		commonParams.layout.Building = map[int64]*cmd.PPlayerCampCommonBuilding{}
	}
	commonParams.layout.Building[building.Building.BuildingId] = building
	h.actor.GetCampData().DecorationBuilding[building.Building.BuildingId] = building.Building

	// 先落地再通知
	return building, nil, int32(cmd.ErrorCode_Success)
}

func (b *BaseBuilding) LevelUp(commonParams *OutputParams, campRet *cmd.PPlayerCamp, h *CampHandler) (error, int32) {
	commonParams.building.BuildingLevel++
	return nil, int32(cmd.ErrorCode_Success)
}

func (b *BaseBuilding) Product(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, h *CampHandler, commonParams *OutputParams, res *cmd.LS2C_PlayerCampBuildFunOpRes) (interface{}, int32) {
	return "", int32(cmd.ErrorCode_Success)
}
func (b *BaseBuilding) Cost(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams) (map[int32]int32, int32) {
	return nil, int32(cmd.ErrorCode_Success)
}

func (b *BaseBuilding) GetChangeReason() common.ChangeReason {
	return common.ChangeReason(0)
}

func (b *BaseBuilding) Check(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams, h *CampHandler) int32 {
	return int32(cmd.ErrorCode_Success)
}
func (b *BaseBuilding) DataLog(commonParams *OutputParams, h *CampHandler, buildingId int64, costs string, formula interface{}) {

}

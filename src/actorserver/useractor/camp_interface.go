package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// IBuilding 建筑功能的抽象
type IBuilding interface {
	// Build 建筑修复
	Build(commonParams *OutputParams, h *CampHandler, req *cmd.C2LS_PlayerCampMakeFunctionBuildingReq, buildingLevelCfg *excel.BuildingLevelCfg) (*cmd.PPlayerCampCommonBuilding, error, int32)
	// LevelUp 建筑升级
	LevelUp(commonParams *OutputParams, campRet *cmd.PPlayerCamp, h *CampHandler) (error, int32)
	//Cost 建筑制造的消耗
	Cost(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams) (map[int32]int32, int32)
	// Product 建筑制造的产物
	Product(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, h *CampHandler, commonParams *OutputParams, res *cmd.LS2C_PlayerCampBuildFunOpRes) (interface{}, int32)
	// GetChangeReason 获取道具来源
	GetChangeReason() common.ChangeReason
	// Check 检测
	Check(formulaMap map[int32]*excel.ItemSynthesisCfg, items []*cmd.PPlayerCampFunctionBuildingFormula, commonParams *OutputParams, h *CampHandler) int32
	// DataLog 埋点日志
	DataLog(commonParams *OutputParams, h *CampHandler, buildingId int64, costs string, formula interface{})
}

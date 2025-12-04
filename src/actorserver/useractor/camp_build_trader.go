package useractor

import (
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"time"
)

// Trader  商人
type Trader struct {
	BaseBuilding
}

func NewTrader() IBuilding {
	return &Trader{
		BaseBuilding: BaseBuilding{},
	}
}

func (lt *Trader) Build(commonParams *OutputParams, h *CampHandler, req *cmd.C2LS_PlayerCampMakeFunctionBuildingReq, buildLevelConfig *excel.BuildingLevelCfg) (*cmd.PPlayerCampCommonBuilding, error, int32) {
	//创建
	building := h.NewPPlayerCampCommonBuilding(req.GetX(), req.GetY(), req.GetParentId(), req.GetParentGridId(), req.GetEdge(), req.GetFlip(), nil)
	building.Building = h.NewPPlayerCampDecorationBuilding(req.ItemId, 1, h.actor.Data.Camp.CurrentCampId)
	if commonParams.layout.Building == nil {
		commonParams.layout.Building = map[int64]*cmd.PPlayerCampCommonBuilding{}
	}

	commonParams.camp.Trader = &cmd.PPlayerCampTrader{
		BuildingId: building.Building.BuildingId,
		Level:      1,
		Items:      h.TryRefreshTraderList(1, nil),
		Refresh:    time.Now().Unix(),
	}
	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{Trader: commonParams.camp.Trader})

	commonParams.layout.Building[building.Building.BuildingId] = building
	h.actor.GetCampData().DecorationBuilding[building.Building.BuildingId] = building.Building
	return building, nil, int32(cmd.ErrorCode_Success)
}
func (lt *Trader) LevelUp(commonParams *OutputParams, campRet *cmd.PPlayerCamp, h *CampHandler) (error, int32) {
	// 刷新商人的清单
	commonParams.camp.Trader.Level++
	commonParams.camp.Trader.Items = h.TryRefreshTraderList(commonParams.camp.Trader.Level, commonParams.camp.Trader.Items)
	commonParams.building.BuildingLevel++
	campRet.Trader = commonParams.camp.Trader
	return nil, int32(cmd.ErrorCode_Success)
}

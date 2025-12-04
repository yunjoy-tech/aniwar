package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// BuildFactory 建筑工厂
func (h *CampHandler) initBuildFactory() {
	h.Buildings = map[cmd.PlayerCampBuildingType]IBuilding{
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_LightingComposeTree: NewLightingComposeTree(), // 光合树
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_Furnace:             NewFurnace(),             // 熔炉
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_FoodSupplyStation:   NewFoodSupplyStation(),   // 食物补给站
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_MaterialInstitute:   NewMaterialInstitute(),   // 材料研究所
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_EquipmentFoundry:    NewEquipmentFoundry(),    // 装备锻造
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_BuildingFoundry:     NewBuildingFoundry(),     // 家具制造台
		cmd.PlayerCampBuildingType_PlayerCampBuildingType_Trader:              NewTrader(),              // 商人
	}
}

// GetProcess 根据类型获取逻辑处理器
func (h *CampHandler) GetProcess(tye cmd.PlayerCampBuildingType) IBuilding {
	return h.Buildings[tye]
}

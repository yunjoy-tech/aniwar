package builder

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
)

func BuildTroopList(data *pb.PCardTroopsInfo) []*pb.PClientCardTroopInfo {
	// 尝试增加编队类型
	// b := false
	// safa Get
	// 干掉非配置数据，保证登录
	// for troopType, troopName := range data.Troop {
	// 	if _, exist := pb.CardTroopType_name[troopType]; !exist {
	// 		if len(data.Troop[troopType].Troop) > 0 {
	// 			h.Warnf("delete unsupported troop type :%v ,troop data %v", troopName, data.Troop[troopType].Troop)
	// 		} else {
	// 			h.Warnf("delete unsupported troop type :%v ,nil data", troopName)
	// 		}
	// 		delete(data.Troop, troopType)
	// 	}
	// }
	//
	// if _, exist := data.Troop[int32(pb.CardTroopType_CardTroopType_None)]; exist {
	// 	delete(data.Troop, int32(pb.CardTroopType_CardTroopType_None))
	// }
	//
	// if _, exist := data.Troop[int32(pb.CardTroopType_CardTroopType_Max)]; exist {
	// 	delete(data.Troop, int32(pb.CardTroopType_CardTroopType_None))
	// }
	//
	// for troopId := range pb.CardTroopType_name {
	// 	if troopId == int32(pb.CardTroopType_CardTroopType_None) ||
	// 		troopId == int32(pb.CardTroopType_CardTroopType_Max) {
	// 		continue
	// 	}
	//
	// 	if _, exist := data.Troop[troopId]; exist {
	// 		continue
	// 	}
	// 	data.Troop[troopId] = &pb.PServerCardTroopInfo{
	// 		TroopType:  troopId,
	// 		Troop:      make(map[int32]*pb.ServerCardTroopInfo),
	// 		UseTroopId: 0,
	// 		Foods:      make([]int32, 0),
	// 	}
	// 	b = true
	// }

	troopData := make([]*pb.PClientCardTroopInfo, 0)
	// for _, info := range data.Troop {
	// 	// 食物上限容错
	// 	limit := int(excel.GetConfigMgr().GetCfg().BATTLE_FOOD_LIMIT)
	// 	if len(info.Foods) > limit {
	// 		b = true
	// 		info.Foods = info.Foods[:limit]
	// 	}
	//
	// 	troopData = append(troopData, &pb.PClientCardTroopInfo{
	// 		TroopType:  info.TroopType,
	// 		Troop:      convertList(info.Troop),
	// 		UseTroopId: info.UseTroopId,
	// 		Foods:      info.Foods,
	// 	})
	// }
	//
	// if b {
	// 	if err := h.SaveDB(); err != nil {
	// 		h.Warn(err)
	// 		return nil
	// 	}
	// }

	return troopData
}

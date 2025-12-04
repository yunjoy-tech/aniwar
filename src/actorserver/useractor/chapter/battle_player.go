package chapter

//
//import (
//	vdata "gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/data"
//	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
//)
//
//type BattlePlayer struct {
//	//RoleId uint64             // 角色id
//	//BattleCards []*BattleCard // 英雄列表
//	//PickMaxTimes uint32
//	//CutMaxTimes	uint32
//	//DigMaxTimes uint32
//	*cmd.LS2C_EnterLevelRes_BattlePlayer
//}
//
//func newBattlePlayer(roleId uint64, cards []*vdata.Card) *BattlePlayer {
//	retCards := make([]*BattleCard, 0)
//	for _, eachCard := range cards {
//		card := newBattleCard(eachCard)
//		retCards = append(retCards, card)
//	}
//
//	return &BattlePlayer{
//		LS2C_EnterLevelRes_BattlePlayer: &cmd.LS2C_EnterLevelRes_BattlePlayer{
//			RoleId:       roleId,
//			BattleCard:   formatCard2Protos(retCards),
//			PickMaxTimes: 0,
//			CutMaxTimes:  0,
//			DigMaxTimes:  0,
//		},
//		//RoleId:       roleId,
//		//BattleCards:  retCards,
//		//PickMaxTimes: 0,
//		//CutMaxTimes:  0,
//		//DigMaxTimes:  0,
//	}
//}
//
//func (p *BattlePlayer) formatPlayer2Proto() *cmd.LS2C_EnterLevelRes_BattlePlayer {
//	return p.LS2C_EnterLevelRes_BattlePlayer
//}
//
//func formatPlayer2Protos(players []*BattlePlayer) []*cmd.LS2C_EnterLevelRes_BattlePlayer {
//	protoPlayers := make([]*cmd.LS2C_EnterLevelRes_BattlePlayer, 0)
//
//	for _, player := range players {
//		proto := player.formatPlayer2Proto()
//		protoPlayers = append(protoPlayers, proto)
//	}
//
//	return protoPlayers
//}
//

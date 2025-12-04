package chapter

//
//import (
//	vdata "gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/data"
//	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
//)
//
//type BattleTeam struct {
//	//CampId uint32                 // 阵营
//	//BattlePlayers []*BattlePlayer // 玩家列表
//	*cmd.LS2C_EnterLevelRes_BattlePlayerTeam
//}
//
//func NewBattleTeamSelf(roleId uint64, cards []*vdata.Card) *BattleTeam {
//
//	players := make([]*BattlePlayer, 0)
//
//	player := newBattlePlayer(roleId, cards)
//	players = append(players, player)
//
//	return &BattleTeam{
//		LS2C_EnterLevelRes_BattlePlayerTeam: &cmd.LS2C_EnterLevelRes_BattlePlayerTeam{
//			CampId:       uint32(cmd.PBattlePlayerCamp_PBattlePlayerCamp_Red),
//			BattlePlayer: formatPlayer2Protos(players),
//		},
//		//CampId:        uint32(cmd.PBattlePlayerCamp_PBattlePlayerCamp_Red),
//		//BattlePlayers: players,
//	}
//}
//
//func (t *BattleTeam) formatTeam2Proto() *cmd.LS2C_EnterLevelRes_BattlePlayerTeam {
//	return t.LS2C_EnterLevelRes_BattlePlayerTeam
//}
//
//func FormatTeam2Protos(teams []*BattleTeam) []*cmd.LS2C_EnterLevelRes_BattlePlayerTeam {
//	players := make([]*cmd.LS2C_EnterLevelRes_BattlePlayerTeam, 0)
//
//	for _, eachTeam := range teams {
//		player := eachTeam.formatTeam2Proto()
//		players = append(players, player)
//	}
//
//	return players
//}

package chapter

//
//import (
//	vdata "gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/data"
//	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
//	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
//)
//
//type BattleCard struct {
//	//CardId         uint32   // 英雄id
//	//CardHp         uint32   // hp
//	//CardMaxHp      uint32   // 最大hp
//	//CardLevel      uint32   // 等级
//	//AttackPower    uint32   // 攻击力
//	//DefensivePower uint32   // 防御力
//	//Skills         []uint32 // 技能列表
//	*cmd.PClientCardInfo
//}
//
//func newBattleCard(card *vdata.Card) *BattleCard {
//	cardCfg := data.GetBeastarMgr().GetById(int32(card.BaseId))
//	skillIds := make([]uint32, 0)
//	for _, each := range cardCfg.GetSkillID() {
//		skillIds = append(skillIds, uint32(each))
//	}
//
//	cardInfo := &cmd.PClientCardInfo{}
//	card.FillClientCardData(cardInfo)
//
//	return &BattleCard{
//		PClientCardInfo: cardInfo,
//		//CardId:         card.GetCardId(),
//		//CardHp:         uint32(cardCfg.GetHp()),
//		//CardMaxHp:      uint32(cardCfg.GetHpUp()),
//		//CardLevel:      card.GetCardLevel(),
//		//AttackPower:    uint32(cardCfg.GetAtkVal()),
//		//DefensivePower: uint32(cardCfg.GetDefVal()),
//		//Skills:         skillIds,
//	}
//}
//
//func (c *BattleCard) formatCard2Proto() *cmd.PClientCardInfo {
//	return c.PClientCardInfo
//}
//
//func formatCard2Protos(cards []*BattleCard) []*cmd.PClientCardInfo {
//	protoCards := make([]*cmd.PClientCardInfo, 0)
//
//	for _, card := range cards {
//		card2Proto := card.formatCard2Proto()
//		protoCards = append(protoCards, card2Proto)
//	}
//	return protoCards
//}

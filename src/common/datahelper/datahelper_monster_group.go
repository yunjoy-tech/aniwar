package datahelper

import "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"

// GetMonsterIdsByBattleEventId 根据战斗事件id, 获取怪物id列表
func GetMonsterIdsByBattleEventId(battleEventId int32) []int32 {
	var (
		monsterIds = make([]int32, 0)
	)

	data.GetMonsterGroupMgr().Foreach(func(cfg *data.MonsterGroupCfg) bool {
		if cfg.BattleEventId == battleEventId {
			monsterIds = append(monsterIds, cfg.MonsterId)
		}

		return false

	}, true)

	return monsterIds
}

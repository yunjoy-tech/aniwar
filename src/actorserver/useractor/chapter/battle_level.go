package chapter

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// BattleLevel 关卡
type BattleLevel struct {
	*cmd.LS2DB_LevelInfo
}

func ReloadBattleLevel(info *cmd.LS2DB_LevelInfo) *BattleLevel {
	return &BattleLevel{info}
}

// 创建一个db结构的关卡数据
func NewBattleLevel(
	levelId int32,
	simpleInfos map[int32]*cmd.LevelSummary) (*BattleLevel, *cmd.LevelSummary) {

	stage := &BattleLevel{
		LS2DB_LevelInfo: &cmd.LS2DB_LevelInfo{
			LevelId:  levelId,
			MapInfos: make(map[int32]*cmd.BattleMapInfo),
		},
	}

	var simpleInfo *cmd.LevelSummary
	if _simpleInfo, ok := simpleInfos[levelId]; ok {
		// 有历史摘要数据，直接返回
		simpleInfo = _simpleInfo
	} else {
		// 没有历史摘要数据，创建新的
		simpleInfo = &cmd.LevelSummary{
			LevelId: levelId,
			LevelSimpleInfo: &cmd.LevelSimpleInfo{
				LevelId:            levelId,
				HistoryHadPassed:   cmd.HistoryHadPassed_PLevelStatus_None,
				UnlockedPointInfos: make([]*cmd.UnlockedPointInfo, 0),
				FirstPassedTimeSec: 0,
				LastPassedTimeSec:  0,
				DailyPassedCount:   0,
			},
			MonsterList: nil,
		}
	}

	return stage, simpleInfo
}

func (s *BattleLevel) FormatStage2DB() *cmd.LS2DB_LevelInfo {
	return s.LS2DB_LevelInfo
}

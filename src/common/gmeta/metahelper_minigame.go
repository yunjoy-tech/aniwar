package gmeta

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
)

// MiniGameWinType 小游戏胜利类型
type MiniGameWinType int32

const (
	MiniGameWinTypeClick MiniGameWinType = 1
	MiniGameWinTypeCd    MiniGameWinType = 2
)

// 获取倒计时时间
func GetMiniGameCountdown(cfgId pb.RoomModel) int32 {
	// cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	// return cfg.Countdown
	return 0
}

// 获取胜利条件配置的值
func GetMiniGameWinCondition(cfgId pb.RoomModel, winType MiniGameWinType) int32 {
	// cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	// return cfg.WinCondition[int32(winType)]
	return 0
}

// 获取整个游戏的最大时长
func GetMiniGameTotalSec(cfgId pb.RoomModel) int32 {
	readyCd := GetMiniGameCountdown(cfgId)
	gameSec := GetMiniGameWinCondition(cfgId, MiniGameWinTypeCd)

	return readyCd + gameSec
}

// 获取最大玩家数量
func GetMiniGamePlayerNum(cfgId pb.RoomModel) int32 {
	// cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	// return cfg.PlayerNum
	return 0
}

// 获取最大携带卡牌数量
func GetMiniGameHeroNum(cfgId pb.RoomModel) int32 {
	// cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	// return cfg.HeroNum
	return 0
}

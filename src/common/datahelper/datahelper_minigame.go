package datahelper

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// MiniGameWinType 小游戏胜利类型
type MiniGameWinType int32

const (
	MiniGameWinTypeClick MiniGameWinType = 1
	MiniGameWinTypeCd    MiniGameWinType = 2
)

// 获取倒计时时间
func GetMiniGameCountdown(cfgId cmd.RoomModel) int32 {
	cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	return cfg.Countdown
}

// 获取胜利条件配置的值
func GetMiniGameWinCondition(cfgId cmd.RoomModel, winType MiniGameWinType) int32 {
	cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	return cfg.WinCondition[int32(winType)]
}

// 获取整个游戏的最大时长
func GetMiniGameTotalSec(cfgId cmd.RoomModel) int32 {
	readyCd := GetMiniGameCountdown(cfgId)
	gameSec := GetMiniGameWinCondition(cfgId, MiniGameWinTypeCd)

	return readyCd + gameSec
}

// 获取最大玩家数量
func GetMiniGamePlayerNum(cfgId cmd.RoomModel) int32 {
	cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	return cfg.PlayerNum
}

// 获取最大携带卡牌数量
func GetMiniGameHeroNum(cfgId cmd.RoomModel) int32 {
	cfg := data.GetMiniGameMgr().GetById(int32(cfgId))
	return cfg.HeroNum
}

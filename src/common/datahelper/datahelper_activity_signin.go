package datahelper

import (
	"gitee.com/bychannel/aniwar/src/excel/data"
)

func GetActivitySinginRewards(activityId, dayIndex int32) []*data.ItemReward {
	var (
		rewards = make([]*data.ItemReward, 0)
	)

	data.GetActivitySigninMgr().Foreach(func(cfg *data.ActivitySigninCfg) bool {
		if cfg.ActivityId == activityId && cfg.Days == dayIndex {
			rewards = cfg.SigninReward
			return false
		}
		return true
	}, false)

	return rewards
}

package datahelper

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
)

// 根据资源id掉落奖励
func GetRewardsByResourceId(resourceId int32) []*data.ItemReward {
	resourceCfg := data.GetResourceMgr().GetById(resourceId)
	return GetRewardsByDropId(resourceCfg.GetDropId())
}

package datahelper

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
)

func GetCampaignBaseRewards(levelId int32) map[int32]int32 {
	var (
		tempRewards = make(map[int32]int32)
	)

	campaignCfg := data.GetCampaignMgr().GetById(levelId)
	if campaignCfg == nil {
		return tempRewards
	}

	// 基础奖励
	myUtils.MergeItems(tempRewards, campaignCfg.RewardBase)
	// 玩家经验
	myUtils.MergeItems(tempRewards, map[int32]int32{common.ITEM_ID_ROLE_EXP_1001: campaignCfg.PlayerExp})
	// 随机奖励
	itemRewards := GetRewardsByDropId(campaignCfg.RewardRandom)
	for _, each := range itemRewards {
		myUtils.MergeItems(tempRewards, map[int32]int32{each.ItemId: each.Num})
	}

	return tempRewards
}

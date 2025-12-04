package datahelper

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
)

func GetResourceCampaign(campaignId int32, campaignType common.CAMPAIGN_TYPE) []*data.ResourcecampaignCfg {
	var (
		cfgs = make([]*data.ResourcecampaignCfg, 0)
	)

	data.GetResourcecampaignMgr().Foreach(func(cfg *data.ResourcecampaignCfg) bool {
		if campaignId == cfg.GetId() && int32(campaignType) == cfg.GetCampaignType() {
			cfgs = append(cfgs, cfg)
		}
		return true
	}, false)

	return cfgs
}

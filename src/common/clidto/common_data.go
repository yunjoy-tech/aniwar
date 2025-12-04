package clidto

import (
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

type Comdata struct {
	Data *cmd.CliComData
	Flag bool // 是否清除标记
}

// FixDownComData 获取actor上的comdata数据，并补全全局数据
func (c *Comdata) FixDownComData() *cmd.CliComData {
	// 补全全局字段
	c.Data.ServerTimestamp = time.Now().UnixMilli()
	c.Data.OpenServerTimestamp = time.Now().UnixMilli() // todo 临时值
	c.Data.NextRefreshTime = common.GetNextDailyRefreshTime()
	c.Flag = true
	return c.Data
}

// BuildComData 只初始化comdata结构
func BuildComData() *Comdata {
	return &Comdata{Data: &cmd.CliComData{}}
}

func (c *Comdata) GetBaseData() *cmd.PClientRoleBaseInfo {
	if c.Data.Base == nil {
		c.Data.Base = &cmd.PClientRoleBaseInfo{
			//Common: &cmd.PCommonRoleBaseInfo{},
		}
	}
	return c.Data.Base
}

func (c *Comdata) GetTutorialData() *cmd.PPlayerBeginnerTutorial {
	if c.Data.Tutorial == nil {
		c.Data.Tutorial = &cmd.PPlayerBeginnerTutorial{}
	}
	return c.Data.Tutorial
}

func (c *Comdata) GetDutyData() *cmd.PCommonDutyInfo {
	if c.Data.Duty == nil {
		c.Data.Duty = &cmd.PCommonDutyInfo{}
	}
	return c.Data.Duty
}

func (c *Comdata) GetCampData() *cmd.PPlayerCampList {
	if c.Data.Camp == nil {
		c.Data.Camp = &cmd.PPlayerCampList{}
	}
	return c.Data.Camp
}

func (c *Comdata) GetQuestData() *cmd.PQuestInfo {
	if c.Data.Quest == nil {
		c.Data.Quest = &cmd.PQuestInfo{}
	}
	return c.Data.Quest
}

func (c *Comdata) GetStaminaData() *cmd.PStaminaInfo {
	if c.Data.Stamina == nil {
		c.Data.Stamina = &cmd.PStaminaInfo{}
	}
	return c.Data.Stamina
}

func (c *Comdata) GetCampaignData() *cmd.PClientGeneralCampaign {
	if c.Data.Campaign == nil {
		c.Data.Campaign = &cmd.PClientGeneralCampaign{}
	}
	return c.Data.Campaign
}

func (c *Comdata) GetLevelSummaryData() *cmd.PClientLevelSummary {
	if c.Data.LevelSummary == nil {
		c.Data.LevelSummary = &cmd.PClientLevelSummary{
			TickInfos:        make([]*cmd.LevelMonsterTicketInfo, 0),
			LevelSummaryList: make([]*cmd.LevelSummary, 0),
		}
	}

	return c.Data.LevelSummary
}

func (c *Comdata) GetShopInfos() []*cmd.ShopInfo {
	if c.Data.ShopInfos == nil {
		c.Data.ShopInfos = make([]*cmd.ShopInfo, 0)
	}
	return c.Data.ShopInfos
}

func (c *Comdata) AddShopInfo(addShopInfos ...*cmd.ShopInfo) {
	shopInfos := c.GetShopInfos()
	for _, addEach := range addShopInfos {
		hadFound := false
		for oldIdx, oldEach := range shopInfos {
			if oldEach.ShopId == addEach.ShopId {
				shopInfos[oldIdx] = addEach
				hadFound = true
				break
			}
		}

		if !hadFound {
			shopInfos = append(shopInfos, addEach)
		}
	}

	c.Data.ShopInfos = shopInfos
}

func (c *Comdata) GetAchieveData() []*cmd.PLevelAchieveInfo {
	if c.Data.Achieve == nil {
		c.Data.Achieve = make([]*cmd.PLevelAchieveInfo, 0)
	}
	return c.Data.Achieve
}

func (c *Comdata) GetBlockWayData() []*cmd.PBlockWayEvent {
	if c.Data.BlockWayEvents == nil {
		c.Data.BlockWayEvents = make([]*cmd.PBlockWayEvent, 0)
	}
	return c.Data.BlockWayEvents
}

func (c *Comdata) GetFriendData() *cmd.PClientFriendInfo {
	if c.Data.Friends == nil {
		c.Data.Friends = &cmd.PClientFriendInfo{}
	}
	return c.Data.Friends
}

func (c *Comdata) GetUseLimitData() *cmd.PClientUseLimitInfo {
	if c.Data.UseLimit == nil {
		c.Data.UseLimit = &cmd.PClientUseLimitInfo{}
	}
	return c.Data.UseLimit
}

func (c *Comdata) GetAllianceData() *cmd.PCommonAllianceInfo {
	if c.Data.Alliance == nil {
		c.Data.Alliance = &cmd.PCommonAllianceInfo{}
	}
	return c.Data.Alliance
}

func (c *Comdata) AddTravelLevelData(data *cmd.PassedTravelLevelData) {
	if c.Data.TravelLevelData == nil {
		c.Data.TravelLevelData = &cmd.PUserTravelLevelData{}
	}

	if c.Data.TravelLevelData.PassedTravelLevelDatas == nil {
		c.Data.TravelLevelData.PassedTravelLevelDatas = make([]*cmd.PassedTravelLevelData, 0)
	}

	c.Data.TravelLevelData.PassedTravelLevelDatas = append(c.Data.TravelLevelData.PassedTravelLevelDatas, data)
}

func (c *Comdata) AddActivityData(activityData *cmd.ActivityData) {
	if c.Data.ActivityData == nil {
		c.Data.ActivityData = &cmd.PClientActivity{}
	}

	if c.Data.ActivityData.ActivityDatas == nil {
		c.Data.ActivityData.ActivityDatas = make([]*cmd.ActivityData, 0)
	}

	hadFound := false
	for idx, each := range c.Data.ActivityData.ActivityDatas {
		if each.ActivityId == activityData.ActivityId {
			hadFound = true

			// 替换成最新数据
			c.Data.ActivityData.ActivityDatas[idx] = activityData
		}
	}

	if !hadFound {
		c.Data.ActivityData.ActivityDatas = append(c.Data.ActivityData.ActivityDatas, activityData)
	}
}

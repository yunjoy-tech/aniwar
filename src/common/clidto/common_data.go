package clidto

import (
	"time"

	"gitee.com/aniwar2/aniwar/src/common"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
)

type Comdata struct {
	Data *pb.CliComData
	Flag bool // 是否清除标记
}

// FixDownComData 获取actor上的comdata数据，并补全全局数据
func (c *Comdata) FixDownComData() *pb.CliComData {
	// 补全全局字段
	c.Data.ServerTimestamp = time.Now().UnixMilli()
	c.Data.OpenServerTimestamp = time.Now().UnixMilli() // todo 临时值
	c.Data.NextRefreshTime = common.GetNextDailyRefreshTime()
	c.Flag = true
	return c.Data
}

// BuildComData 只初始化comdata结构
func BuildComData() *Comdata {
	return &Comdata{Data: &pb.CliComData{}}
}

func (c *Comdata) GetBaseData() *pb.PClientRoleBaseInfo {
	if c.Data.Base == nil {
		c.Data.Base = &pb.PClientRoleBaseInfo{
			// Common: &pb.PCommonRoleBaseInfo{},
		}
	}
	return c.Data.Base
}

func (c *Comdata) GetTutorialData() *pb.PPlayerBeginnerTutorial {
	if c.Data.Tutorial == nil {
		c.Data.Tutorial = &pb.PPlayerBeginnerTutorial{}
	}
	return c.Data.Tutorial
}

func (c *Comdata) GetDutyData() *pb.PCommonDutyInfo {
	if c.Data.Duty == nil {
		c.Data.Duty = &pb.PCommonDutyInfo{}
	}
	return c.Data.Duty
}

func (c *Comdata) GetCampData() *pb.PPlayerCampList {
	if c.Data.Camp == nil {
		c.Data.Camp = &pb.PPlayerCampList{}
	}
	return c.Data.Camp
}

func (c *Comdata) GetQuestData() *pb.PQuestInfo {
	if c.Data.Quest == nil {
		c.Data.Quest = &pb.PQuestInfo{}
	}
	return c.Data.Quest
}

func (c *Comdata) GetStaminaData() *pb.PStaminaInfo {
	if c.Data.Stamina == nil {
		c.Data.Stamina = &pb.PStaminaInfo{}
	}
	return c.Data.Stamina
}

func (c *Comdata) GetCampaignData() *pb.PClientGeneralCampaign {
	if c.Data.Campaign == nil {
		c.Data.Campaign = &pb.PClientGeneralCampaign{}
	}
	return c.Data.Campaign
}

func (c *Comdata) GetLevelSummaryData() *pb.PClientLevelSummary {
	if c.Data.LevelSummary == nil {
		c.Data.LevelSummary = &pb.PClientLevelSummary{
			TickInfos:        make([]*pb.LevelMonsterTicketInfo, 0),
			LevelSummaryList: make([]*pb.LevelSummary, 0),
		}
	}

	return c.Data.LevelSummary
}

func (c *Comdata) GetShopInfos() []*pb.ShopInfo {
	if c.Data.ShopInfos == nil {
		c.Data.ShopInfos = make([]*pb.ShopInfo, 0)
	}
	return c.Data.ShopInfos
}

func (c *Comdata) AddShopInfo(addShopInfos ...*pb.ShopInfo) {
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

func (c *Comdata) GetAchieveData() []*pb.PLevelAchieveInfo {
	if c.Data.Achieve == nil {
		c.Data.Achieve = make([]*pb.PLevelAchieveInfo, 0)
	}
	return c.Data.Achieve
}

func (c *Comdata) GetBlockWayData() []*pb.PBlockWayEvent {
	if c.Data.BlockWayEvents == nil {
		c.Data.BlockWayEvents = make([]*pb.PBlockWayEvent, 0)
	}
	return c.Data.BlockWayEvents
}

func (c *Comdata) GetFriendData() *pb.PClientFriendInfo {
	if c.Data.Friends == nil {
		c.Data.Friends = &pb.PClientFriendInfo{}
	}
	return c.Data.Friends
}

func (c *Comdata) GetUseLimitData() *pb.PClientUseLimitInfo {
	if c.Data.UseLimit == nil {
		c.Data.UseLimit = &pb.PClientUseLimitInfo{}
	}
	return c.Data.UseLimit
}

func (c *Comdata) GetAllianceData() *pb.PCommonAllianceInfo {
	if c.Data.Alliance == nil {
		c.Data.Alliance = &pb.PCommonAllianceInfo{}
	}
	return c.Data.Alliance
}

func (c *Comdata) AddTravelLevelData(data *pb.PassedTravelLevelData) {
	if c.Data.TravelLevelData == nil {
		c.Data.TravelLevelData = &pb.PUserTravelLevelData{}
	}

	if c.Data.TravelLevelData.PassedTravelLevelDatas == nil {
		c.Data.TravelLevelData.PassedTravelLevelDatas = make([]*pb.PassedTravelLevelData, 0)
	}

	c.Data.TravelLevelData.PassedTravelLevelDatas = append(c.Data.TravelLevelData.PassedTravelLevelDatas, data)
}

func (c *Comdata) AddActivityData(activityData *pb.ActivityData) {
	if c.Data.ActivityData == nil {
		c.Data.ActivityData = &pb.PClientActivity{}
	}

	if c.Data.ActivityData.ActivityDatas == nil {
		c.Data.ActivityData.ActivityDatas = make([]*pb.ActivityData, 0)
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

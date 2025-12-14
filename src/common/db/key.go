package db

import (
	"fmt"
	"time"

	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/bychannel/aniwar/src/common/conf"
)

// var KeyCacheRedisKeyIdx = func(mongoDbName service.MongoDbType, account string) string {
//	return fmt.Sprintf("%v:cache-key:%s", mongoDbName, account)
// }

// 玩家账号登陆锁
var KeyAccountLoginLock = func(account string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:lock:login:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, account)
	}
	return fmt.Sprintf("lock:login:%s", account)
}

var KeyCacheRedisData = func(uid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:cache-data", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
	}
	return fmt.Sprintf("%v:cache-data", uid)
}

var KeyUserToken = func(uid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:token", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
	}
	return fmt.Sprintf("%v:token", uid)
}

// RoomSession数据块
var KeyUserSession = func(uid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:session", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
	}
	return fmt.Sprintf("%v:session", uid)
}

// // RoomSession数据块
// var KeyRoomSession = func(uid string) string {
//  if conf.Base().IsDBPrefix {
//  	return fmt.Sprintf("%s:%s:%v:roomsession", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
//  }
//	return fmt.Sprintf("%v:roomsession", uid)
// }

// 绑定玩家角色id和roomId
var KeyPlayerUidAndRoomId = func(uid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:roomId", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
	}
	return fmt.Sprintf("%v:roomId", uid)
}

// 玩家长连接离线消息
var KeyOfflineMsg = func(uid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:offlinemsg", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
	}
	return fmt.Sprintf("%v:offlinemsg", uid)
}

// 基于账号的UAID索引
var KeyAccountUAID = func(account string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:uaid", global.RdsCfgNameSpace, global.RdsCfgGroup, account)
	}
	return fmt.Sprintf("%v:uaid", account)
}

// 基于PlayerID的UAID索引
var KeyPlayerUAID = func(playerId uint64) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:uaid", global.RdsCfgNameSpace, global.RdsCfgGroup, playerId)
	}
	return fmt.Sprintf("%v:uaid", playerId)
}

// taptap openId-uid映射
var KeyTaptapOpenId = func(openId string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:taptap:%v", global.RdsCfgNameSpace, global.RdsCfgGroup, openId)
	}
	return fmt.Sprintf("taptap:%v", openId)
}

// KeyHeartBeat 心跳数据
// @param uid 账号id
var KeyHeartBeat = func(uid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:heatbeat", global.RdsCfgNameSpace, global.RdsCfgGroup, uid)
	}
	return fmt.Sprintf("%v:heatbeat", uid)
}

// 玩家账号数据块
var KeyAccountInfo = func(account string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:account", global.RdsCfgNameSpace, global.RdsCfgGroup, account)
	}
	return fmt.Sprintf("%v:account", account)
}

// OrderHandler
var KeyUserOrderInfo = func(account string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:order", global.RdsCfgNameSpace, global.RdsCfgGroup, account)
	}
	return fmt.Sprintf("%v:order", account)
}

// 离线事件
var KeyOfflineEvent = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:offlineevent", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:offlineevent", uaid)
}

// BaseHandler
var KeyUserBaseInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:baseinfo", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:baseinfo", uaid)
}

// TroopHandler
var KeyUserCardTroop = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:cardtroop", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:cardtroop", uaid)
}

// CardHandler
var KeyUserCard = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:card", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:card", uaid)
}

// CampHandler
var KeyUserCamp = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:camp", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:camp", uaid)
}

// BagHandler
var KeyUserItems = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:items", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:items", uaid)
}

// TutorialHandler
var KeyUserTutorial = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:tutorial", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:tutorial", uaid)
}

// CurrencyHandler
var KeyUserCurrency = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:currency", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:currency", uaid)
}

// PoolHandler
var KeyUserCardPool = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:cardpool", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:cardpool", uaid)
}

// PoolHandler
var KeyUserCampPool = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:camppool", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:camppool", uaid)
}

// UseLimitHandler
var KeyUseLimit = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:uselimit", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:uselimit", uaid)
}

// HandBookHandler
var KeyUserHandBook = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:handbook", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:handbook", uaid)
}

// QuestionHandler
var KeyUserQuestion = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:question", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:question", uaid)
}

// ChapterHandler
var KeyUserLevelInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:chapter", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:chapter", uaid)
}

// ShopHandler
var KeyUserShopInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:shop", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:shop", uaid)
}

// MailHandler
var KeyUserMail = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:mail", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:mail", uaid)
}

// FriendHandler
var KeyUserFriend = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:friend", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:friend", uaid)
}

var KeySystemMail = func() string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:systemmail", global.RdsCfgNameSpace, global.RdsCfgGroup)
	}
	return "systemmail"
}

var KeyServerRegisterUsers = func() string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:registerusers", global.RdsCfgNameSpace, global.RdsCfgGroup)
	}
	return "registerusers"
}

// EquipHandler
var KeyUserEquipInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:equip", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:equip", uaid)
}

// DutyHandler
var KeyUserDutyInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:duty", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:duty", uaid)
}

// GuideTaskHandler
var KeyUserGuideTask = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:guidetask", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:guidetask", uaid)
}

// QuestHandler
var KeyUserQuestInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:quest", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:quest", uaid)
}

// CampaignHandler
var KeyCampaign = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:gencampaign", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:gencampaign", uaid)
}

// StoryFlagHandler
var KeyUserStoryFlag = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:storyFlag", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:storyFlag", uaid)
}

// AchieveHandler
var KeyUserAchieve = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:achieve", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:achieve", uaid)
}

// SkinHandler
var KeyUserCardSkin = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:cardskin", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:cardskin", uaid)
}

// SignHandler
var KeyUserSign = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:sign", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:sign", uaid)
}

// TrialHandler
var KeyUserTrial = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:trial", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:trial", uaid)
}

// BlockWayHandler
var KeyUserBlockWay = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:blockway", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:blockway", uaid)
}

// RoleDetailHandler
var KeyRoleDetailInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:roledetail", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:roledetail", uaid)
}

// PlayerLevelHandler
var KeyUserLevelData = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:playerlevel", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:playerlevel", uaid)
}

var KeyGmtLilith = func(ts string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:lilith:%v", global.RdsCfgNameSpace, global.RdsCfgGroup, ts)
	}
	return fmt.Sprintf("lilith:%v", ts)
}

var KeyGmtAniwar = func(ts string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:aniwar:%v", global.RdsCfgNameSpace, global.RdsCfgGroup, ts)
	}
	return fmt.Sprintf("aniwar:%v", ts)
}

var KeyGMTVerify = func() string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:gmt-aniwar-%s", global.RdsCfgNameSpace, global.RdsCfgGroup, time.Now().Month().String())
	}
	return fmt.Sprintf("gmt-aniwar-%s", time.Now().Month().String())
}

// PVP-RoomHandler
var KeyPvpRoomData = func(roomId string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:room", global.RdsCfgNameSpace, global.RdsCfgGroup, roomId)
	}
	return fmt.Sprintf("%v:room", roomId)
}

// PVP-RoomHandler
var KeyGameTugData = func(roomId string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:gameTug", global.RdsCfgNameSpace, global.RdsCfgGroup, roomId)
	}
	return fmt.Sprintf("%v:gameTug", roomId)
}

var KeyAllianceData = func(allianceId string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:alliance", global.RdsCfgNameSpace, global.RdsCfgGroup, allianceId)
	}
	return fmt.Sprintf("%v:alliance", allianceId)
}

var KeyUserAlliance = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:useralliance", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:useralliance", uaid)
}

// KeyUserChatInfo
var KeyUserChatInfo = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:chat", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:chat", uaid)
}

// KeyUserRelation
var KeyUserRelation = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:relation", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:relation", uaid)
}

// KeyUserRelation
var KeyUserCallSys = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:callSys", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:callSys", uaid)
}

// useractor
var KeyUserActor = func(actorType, uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:actor:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, actorType, uaid)
	}
	return fmt.Sprintf("actor:%s:%s", actorType, uaid)
}

// TravelLevelHandler
var KeyUserTravelLevel = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:travelLevel", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:travelLevel", uaid)
}

// ActivityHandler
var KeyUserActivity = func(uaid string) string {
	if conf.Base().IsDBPrefix {
		return fmt.Sprintf("%s:%s:%v:activity", global.RdsCfgNameSpace, global.RdsCfgGroup, uaid)
	}
	return fmt.Sprintf("%v:activity", uaid)
}

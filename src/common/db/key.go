package db

import (
	"fmt"
	"time"
)

// var KeyCacheRedisKeyIdx = func(mongoDbName service.MongoDbType, account string) string {
//	return fmt.Sprintf("%v:cache-key:%s", mongoDbName, account)
// }

// 玩家账号登陆锁
var KeyAccountLoginLock = func(account string) string {
	return fmt.Sprintf("lock:login:%s", account)
}

var KeyCacheRedisData = func(uid string) string {
	return fmt.Sprintf("%v:cache-data", uid)
}

var KeyUserToken = func(uid string) string {
	return fmt.Sprintf("%v:token", uid)
}

// RoomSession数据块
var KeyUserSession = func(uid string) string {
	return fmt.Sprintf("%v:session", uid)
}

// 绑定玩家角色id和roomId
var KeyPlayerUidAndRoomId = func(uid string) string {
	return fmt.Sprintf("%v:roomId", uid)
}

// 玩家长连接离线消息
var KeyOfflineMsg = func(uid string) string {
	return fmt.Sprintf("%v:offlinemsg", uid)
}

// 基于账号的UAID索引
var KeyAccountUAID = func(account string) string {
	return fmt.Sprintf("%v:uaid", account)
}

// 基于PlayerID的UAID索引
var KeyPlayerUAID = func(playerId uint64) string {
	return fmt.Sprintf("%v:uaid", playerId)
}

// taptap openId-uid映射
var KeyTaptapOpenId = func(openId string) string {
	return fmt.Sprintf("taptap:%v", openId)
}

// KeyHeartBeat 心跳数据
// @param uid 账号id
var KeyHeartBeat = func(uid string) string {
	return fmt.Sprintf("%v:heatbeat", uid)
}

// 玩家账号数据块
var KeyAccountInfo = func(account string) string {
	return fmt.Sprintf("%v:account", account)
}

// OrderHandler
var KeyUserOrderInfo = func(account string) string {
	return fmt.Sprintf("%v:order", account)
}

// 离线事件
var KeyOfflineEvent = func(uaid string) string {
	return fmt.Sprintf("%v:offlineevent", uaid)
}

// BaseHandler
var KeyUserBaseInfo = func(uaid string) string {
	return fmt.Sprintf("%v:baseinfo", uaid)
}

// TroopHandler
var KeyUserCardTroop = func(uaid string) string {
	return fmt.Sprintf("%v:cardtroop", uaid)
}

// CardHandler
var KeyUserCard = func(uaid string) string {
	return fmt.Sprintf("%v:card", uaid)
}

// CampHandler
var KeyUserCamp = func(uaid string) string {
	return fmt.Sprintf("%v:camp", uaid)
}

// BagHandler
var KeyUserItems = func(uaid string) string {
	return fmt.Sprintf("%v:items", uaid)
}

// TutorialHandler
var KeyUserTutorial = func(uaid string) string {
	return fmt.Sprintf("%v:tutorial", uaid)
}

// CurrencyHandler
var KeyUserCurrency = func(uaid string) string {
	return fmt.Sprintf("%v:currency", uaid)
}

// PoolHandler
var KeyUserCardPool = func(uaid string) string {
	return fmt.Sprintf("%v:cardpool", uaid)
}

// PoolHandler
var KeyUserCampPool = func(uaid string) string {
	return fmt.Sprintf("%v:camppool", uaid)
}

// UseLimitHandler
var KeyUseLimit = func(uaid string) string {
	return fmt.Sprintf("%v:uselimit", uaid)
}

// HandBookHandler
var KeyUserHandBook = func(uaid string) string {
	return fmt.Sprintf("%v:handbook", uaid)
}

// ChapterHandler
var KeyUserLevelInfo = func(uaid string) string {
	return fmt.Sprintf("%v:chapter", uaid)
}

// ShopHandler
var KeyUserShopInfo = func(uaid string) string {
	return fmt.Sprintf("%v:shop", uaid)
}

// MailHandler
var KeyUserMail = func(uaid string) string {
	return fmt.Sprintf("%v:mail", uaid)
}

// FriendHandler
var KeyUserFriend = func(uaid string) string {
	return fmt.Sprintf("%v:friend", uaid)
}

var KeySystemMail = func() string {
	return "systemmail"
}

var KeyServerRegisterUsers = func() string {
	return "registerusers"
}

// EquipHandler
var KeyUserEquipInfo = func(uaid string) string {
	return fmt.Sprintf("%v:equip", uaid)
}

// DutyHandler
var KeyUserDutyInfo = func(uaid string) string {
	return fmt.Sprintf("%v:duty", uaid)
}

// GuideTaskHandler
var KeyUserGuideTask = func(uaid string) string {
	return fmt.Sprintf("%v:guidetask", uaid)
}

// QuestHandler
var KeyUserQuestInfo = func(uaid string) string {
	return fmt.Sprintf("%v:quest", uaid)
}

// CampaignHandler
var KeyCampaign = func(uaid string) string {
	return fmt.Sprintf("%v:gencampaign", uaid)
}

// StoryFlagHandler
var KeyUserStoryFlag = func(uaid string) string {
	return fmt.Sprintf("%v:storyFlag", uaid)
}

// AchieveHandler
var KeyUserAchieve = func(uaid string) string {
	return fmt.Sprintf("%v:achieve", uaid)
}

// SkinHandler
var KeyUserCardSkin = func(uaid string) string {
	return fmt.Sprintf("%v:cardskin", uaid)
}

// SignHandler
var KeyUserSign = func(uaid string) string {
	return fmt.Sprintf("%v:sign", uaid)
}

// TrialHandler
var KeyUserTrial = func(uaid string) string {
	return fmt.Sprintf("%v:trial", uaid)
}

// BlockWayHandler
var KeyUserBlockWay = func(uaid string) string {
	return fmt.Sprintf("%v:blockway", uaid)
}

// RoleDetailHandler
var KeyRoleDetailInfo = func(uaid string) string {
	return fmt.Sprintf("%v:roledetail", uaid)
}

// PlayerLevelHandler
var KeyUserLevelData = func(uaid string) string {
	return fmt.Sprintf("%v:playerlevel", uaid)
}

var KeyGmtLilith = func(ts string) string {
	return fmt.Sprintf("lilith:%v", ts)
}

var KeyGmtAniwar = func(ts string) string {
	return fmt.Sprintf("aniwar:%v", ts)
}

var KeyGMTVerify = func() string {
	return fmt.Sprintf("gmt-aniwar-%s", time.Now().Month().String())
}

// PVP-RoomHandler
var KeyPvpRoomData = func(roomId string) string {
	return fmt.Sprintf("%v:room", roomId)
}

// PVP-RoomHandler
var KeyGameTugData = func(roomId string) string {
	return fmt.Sprintf("%v:gameTug", roomId)
}

var KeyAllianceData = func(allianceId string) string {
	return fmt.Sprintf("%v:alliance", allianceId)
}

var KeyUserAlliance = func(uaid string) string {
	return fmt.Sprintf("%v:useralliance", uaid)
}

// KeyUserChatInfo
var KeyUserChatInfo = func(uaid string) string {
	return fmt.Sprintf("%v:chat", uaid)
}

// KeyUserRelation
var KeyUserRelation = func(uaid string) string {
	return fmt.Sprintf("%v:relation", uaid)
}

// useractor
var KeyUserActor = func(actorType, uaid string) string {
	return fmt.Sprintf("actor:%s:%s", actorType, uaid)
}

// TravelLevelHandler
var KeyUserTravelLevel = func(uaid string) string {
	return fmt.Sprintf("%v:travelLevel", uaid)
}

// ActivityHandler
var KeyUserActivity = func(uaid string) string {
	return fmt.Sprintf("%v:activity", uaid)
}

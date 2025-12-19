package common

//go:generate stringer -type=ChangeReason
type ChangeReason int

const (
	LanguageDefault = "zh_CN"
)

// 游戏常量
const (
	GAME_DAILY_REFRESH_HOUR = 5  // 游戏刷新时间点
	FIXED_SAVE_DB_TIME      = 10 // 数据延迟写库时间

	USER_ID_BASE = 10000 // 用户ID基数

	ExcelDataVersionLen = 27 // excel-data的版本信息数据长度

	RoomIdSecret = "63c163@00e730387"
)

const (
	ACCOUNT_BANED       = "account_baned"       // 账号已被封禁
	ACCOUNT_MULTI_LOGIN = "account_multi_login" // 账号异地登陆
)

// es索引名称
const (
	ES_ROLE_DETAIL_KEY   = "role_detail"   // 玩家详情数据key
	ES_ALLIANCE_BASE_KEY = "alliance_base" // 联盟基础数据key
)

// 屏蔽字检查类型
const (
	CHECK_TYPE_PLAYERNAME = 1 // 玩家昵称
	CHECK_TYPE_SQUADNAME  = 2 // 编队名称
)

// 道具来源
const (
	CR_NewbieGift                   ChangeReason = 1  // "新手礼包"
	CR_UseItem                      ChangeReason = 2  // "使用道具"
	CR_BuyItem                      ChangeReason = 3  // "购买道具"
	CR_Destroy_EXP_ITEM             ChangeReason = 4  // "销毁过期道具"
	CR_GM                           ChangeReason = 5  // "gm"
	CR_Camp_Building_Create         ChangeReason = 6  // "建筑创建"
	CR_Camp_Building_Get            ChangeReason = 7  // "建筑产出获取"
	CR_Camp_Building_Furnace        ChangeReason = 8  // "道具合成"
	CR_Camp_Building_FoodSupply     ChangeReason = 9  // "食物供应"
	CR_Camp_Building_OpCancel       ChangeReason = 10 // "建筑制造取消"
	CR_Camp_Equip_Fuoundry          ChangeReason = 11 // "装备合成台锻造"
	CR_Camp_CampMaterial_Conversion ChangeReason = 12 // "材料研究所转换"
	CR_Camp_Trader_Exchange         ChangeReason = 13 // "营地商人兑换"
	CR_Battle_collect               ChangeReason = 14 // "战斗中-收集"
	CR_Card_Breakthrough_Upgrade    ChangeReason = 15 // "卡牌突破升级"
	CR_Card_Skill_Upgrade           ChangeReason = 16 // "卡牌技能升级"
	CR_Card_Awaken_Upgrade          ChangeReason = 17 // "卡牌觉醒升级"
	CR_Card_Level_Up                ChangeReason = 18 // "卡牌升级"
	CR_Card_Pool                    ChangeReason = 19 // "抽卡"
	CR_Add_Card                     ChangeReason = 20 // "获得卡牌"
	CR_Card_Character_Break         ChangeReason = 21 // "卡牌性格突破"
	CR_Card_Character_Upgrade       ChangeReason = 22 // "卡牌性格升级"
	CR_Card_Eat_Food                ChangeReason = 23 // "卡牌食物使用"
	CR_Handbook_Single_Reward       ChangeReason = 24 // "图鉴单个奖励"
	CR_Mail_Attachment              ChangeReason = 25 // "领取邮件"
	CR_Currency_Exchange            ChangeReason = 26 // "货币兑换"
	CR_Shop_buy                     ChangeReason = 27 // "商店购买"
	CR_Daily_Sign                   ChangeReason = 28 // "每日签到"
	CR_Quest_Reward                 ChangeReason = 29 // "剧情任务"
	CR_Daily_Task_Reward            ChangeReason = 30 // "每日任务奖励"
	CR_STAMINA_BUY                  ChangeReason = 31 // 耐力购买
	CR_Campaign_Reward              ChangeReason = 32 // "日替或常驻关卡奖励"
	CR_Campaign_battle_97           ChangeReason = 33 // "日替或常驻战斗结算"
	CR_Save_Story_Flag              ChangeReason = 34 // "保存story-flag"
	CR_offline_exec                 ChangeReason = 35 // 离线逻辑
	CR_PASS_LEVEL_ONCE              ChangeReason = 36 // 通关奖励 - 一次性
	CR_PASS_LEVEL_BASE              ChangeReason = 37 // 通关奖励 - 基础的
	CR_PASS_LEVEL_TEMP              ChangeReason = 38 // 通关奖励 - 存储的
	CR_DISCOVERY_UNLOCK_POINT       ChangeReason = 39 // 解锁地标奖励
	CR_ENTER_LEVEL                  ChangeReason = 40 // 进入关卡
	CR_EXEC_EVENT                   ChangeReason = 41 // 执行事件
	CR_REBACK_StAMINA               ChangeReason = 42 // 返还预扣的体力
	CR_Daily_Task_Active_Reward     ChangeReason = 44 // 任务好感度奖励
	CR_Quest_Object_Submit          ChangeReason = 45 // 任务提交材料
	CR_Campaign_battle_98           ChangeReason = 46 // "日替或常驻战斗结算"
	CR_Campaign_battle_99           ChangeReason = 47 // "日替或常驻战斗结算"
	CR_Campaign_battle_100          ChangeReason = 48 // "日替或常驻战斗结算"
	CR_STAMINA_PLAY_LEVEL           ChangeReason = 49 // 玩家升级体力变更
	CR_LEVEL_DAILY_RECOVER          ChangeReason = 50 // 大地图 - 门票每日恢复
	CR_LEVEL_BATTLE_COST            ChangeReason = 50 // 大地图 - 战斗消耗
	CR_LEVEL_INCR_MAX               ChangeReason = 50 // 大地图 - 门票最大值增加
	CR_CHARACTER_UNLOCK             ChangeReason = 51 // 性格解锁
	CR_FAVOR_REWARD                 ChangeReason = 52 // 好感度奖励
	CR_FAVOR_ITEM_USE               ChangeReason = 53 // 好感度道具使用
	CR_ACHIEVE_REWARD               ChangeReason = 54 // 成就奖励
	CR_ADD_CARD_SKIN                ChangeReason = 55 // 获得皮肤
	CR_Shop_Manual_refresh          ChangeReason = 56 // "商店手动刷新"
	CR_TRIAL                        ChangeReason = 57 // 试炼
	CR_RECHARGE                     ChangeReason = 58 // 充值购买
	CR_ROAD_SHOP                    ChangeReason = 59 // 拦路事件
	CR_GM_REDUCE                    ChangeReason = 60 // GM 扣除
	CR_ADD_FRIEND_POINT             ChangeReason = 61 // 领取友情点
	CR_CAMP_HOME_COIN               ChangeReason = 62 // 营地家装库
	CR_CAMP_POOL                    ChangeReason = 63 // 营地扭蛋
	CR_CHANGE_NICKNAME              ChangeReason = 64 // 改名
	CR_ALLIANCE_CREATE              ChangeReason = 65 // 联盟创建
	CR_CAMP_LAYOUT_EDIT             ChangeReason = 66 // 营地扩建、或者购买扩建次数
	CR_Finish_Event_Submit          ChangeReason = 67 // 完成事件时提交材料
	CR_COST_BY_BATTLE_TRAVEL_LEVEL  ChangeReason = 68 // 旅途关卡战斗消耗
	CR_PASS_TRAVEL_LEVEL_ONCE       ChangeReason = 69 // 通关奖励 - 一次性
	CR_PASS_TRAVEL_LEVEL_BASE       ChangeReason = 70 // 通关奖励 - 基础的
	CR_FINISH_GUIDE_TASK            ChangeReason = 71 // 引导任务
	CR_EQUIP_LEVEL_UP               ChangeReason = 72 // 装备升级
	CR_FINISH_ACTIVITY              ChangeReason = 73 // 活动奖励
)

// 道具常量
const (
	ITEM_ID_ROLE_EXP_1001 = 1001 // 玩家经验
	ITEM_ID_STAMINA_1004  = 1004 // 体力
)

// 货币道具常量
const (
	CURRENCY_ITEM_ID_2001 = 2001 // type=4 电力 			Electric
	CURRENCY_ITEM_ID_2005 = 2005 // type=1 免费灵之砂 	Diamond
	CURRENCY_ITEM_ID_2006 = 2006 // type=1 付费灵之砂 	Diamond
	CURRENCY_ITEM_ID_2007 = 2007 // type=5 活力 			Vitality
	CURRENCY_ITEM_ID_2008 = 2008 // type=3 灵感 			Collect
	CURRENCY_ITEM_ID_2009 = 2009 // type=6 数据片段 		GeneFragment
	CURRENCY_ITEM_ID_2010 = 2010 // type=7 精英怪挑战券 	LevelEliteTicket
	CURRENCY_ITEM_ID_2011 = 2011 // type=8 boss挑战券 	LevelBossTicket
	CURRENCY_ITEM_ID_2012 = 2012 // type=9 家装币 		HomeIcon
	CURRENCY_ITEM_ID_2013 = 2013 // type=10 友情点 		FriendPoint
)

// 日替关卡类型
type CAMPAIGN_TYPE int

const (
	CAMPAIGN_TYPE_97 CAMPAIGN_TYPE = 97 // 日替
	CAMPAIGN_TYPE_98 CAMPAIGN_TYPE = 98 // 日替

	CAMPAIGN_TYPE_99  CAMPAIGN_TYPE = 99  // 开车小游戏
	CAMPAIGN_TYPE_100 CAMPAIGN_TYPE = 100 // 金币副本
)

// 关卡事件刷新条件类型
type MAPPOINT_EVENT_UPDATE_TYPE int

const (
	MAPPOINT_EVENT_UPDATE_TYPE_0 MAPPOINT_EVENT_UPDATE_TYPE = 0 // 常规刷新
	MAPPOINT_EVENT_UPDATE_TYPE_2 MAPPOINT_EVENT_UPDATE_TYPE = 2 // 表示跟随任务刷新，对应参数为任务id，任务完成后，该事件才会常态刷新
	MAPPOINT_EVENT_UPDATE_TYPE_5 MAPPOINT_EVENT_UPDATE_TYPE = 5 // 表示跟随一次性事件完成刷新，对应参数为地图事件的id，该事件完成后，新的事件才会常态刷新

)

// 关卡类型
type LEVEL_TYPE int32

var NIWA_EVENT_GROUP_CD = int64(-1) // 地图事件组-cd倒计时 (9999-01-01 00:00:00)

const (
	CHAPTER_LEVEL_TYPE_MAIN   LEVEL_TYPE = 1   // 大关卡
	CHAPTER_LEVEL_TYPE_SUB    LEVEL_TYPE = 2   // 子关卡
	CHAPTER_LEVEL_TYPE_TRAVEL LEVEL_TYPE = 101 // 旅途关卡

	CHAPTER_LEVEL_ISFINISH = int32(1) // 关卡是否完成
)

// gm指令名
const (
	GM_ACTOR_SHOW = "user.test.actorShow" // 打印actor信息
	GM_ACTOR_DEL  = "user.test.actorDel"  // 删除actor

	// add组cmd
	GM_ADD_ITEM           = "user.add.item"
	GM_ADD_ITEM_ALL       = "user.add.itemall"
	GM_ADD_ITEM_BY_TYPE   = "user.add.itembytype"
	GM_ADD_CARD_EXP       = "user.add.cardexp"
	GM_ADD_FAVORITE_EXP   = "user.add.favoriteexp"
	GM_ADD_PLAYER_EXP     = "user.add.playerexp"
	GM_ADD_CARDS_RELATION = "user.add.relation"

	GM_CLEAN_ITEM = "user.clean.item"

	// del组cmd
	GM_DEL_MONEY      = "user.del.money"
	GM_DEL_STAMINA    = "user.del.stamina"
	GM_DEL_ITEM_BY_ID = "user.del.itembybaseid"

	// set组cmd
	GM_SET_CARD_STRENGTH            = "user.set.cardphysicalstrength"
	GM_SET_LIGHTING_COMPOSE_TREE_TS = "user.set.lightingcomposetreets"
	GM_SET_SUPER_CARD               = "user.set.supercard"

	// reset组cmd
	GM_RESET_LEVEL         = "user.reset.level"       // 重置玩家等级
	GM_RESET_CARD_POOL_LOG = "user.reset.cardpoollog" // 重置抽卡保底次数记录
	GM_RESET_DUTY_TASK     = "user.reset.dutytask"    // 重置值日生任务

	// test组cmd
	GM_TEST_PROTO                  = "user.test.proto"
	GM_TEST_SIGN                   = "user.test.sign"
	GM_TEST_MAIL                   = "user.test.mail"
	GM_TEST_UGC                    = "user.test.ugc"                    // 测试ugc字符串
	GM_TEST_SENSITIVE              = "user.test.sensitive"              // 测试屏蔽词库
	GM_TEST_Battle_chapter         = "user.test.battlechapter"          // 测试checkBattle-chapter接口
	GM_CHECKBATTLE_RELOAD_EXCEL    = "user.test.checkBattleReloadExcel" // battleServer热更
	GM_TEST_CARD                   = "user.test.card"
	GM_TEST_GEN_CODE               = "user.test.gencode"                // 测试生成礼包码
	GM_TEST_USE_CODE               = "user.test.usecode"                // 测试使用礼包码
	GM_TEST_DROP                   = "user.test.drop"                   // 测试掉落
	GM_TEST_GUID                   = "user.test.guid"                   // 测试GUID
	GM_TEST_DB                     = "user.test.db"                     // 测试数据库
	GM_ERR_CODE                    = "user.test.errCode"                // 返回一个errorCode
	GM_TEST_ACHIEVE                = "user.test.achievement"            // 测试完成成就
	GM_TEST_PVP_ROOM               = "user.test.room"                   // 房间测试
	GM_CLOSE_BATTKE_CHECK          = "user.test.closeBattleCheck"       // 关闭战斗校验
	GM_TEST_RECOMMEND              = "user.test.recommend"              // 测试好友推荐
	GM_TEST_ChangeCampHomeIconTime = "user.test.changeCampHomeIconTime" // 测试好友推荐
	GM_TEST_CampDouble             = "user.test.camp_double"            // 测试营地双倍
	GM_TEST_Card_Broad             = "user.test.card_broad"             // 测试营地双倍
	GM_TEST_Del_Camp_Build         = "user.test.del.camp_build"         // 测试营地双倍
	GM_TEST_Test_Cfg_Hot           = "user.test.test.cfg_hot"           // 测试热更文件

	// 其他cmd
	GM_DIRECT_LEVEL_UP           = "user.directlevelup"
	GM_WEAR_EQUIP                = "user.wearequip"
	GM_DIRECT_COMPLETE_OBJECT    = "user.directcompleteobject"
	GM_DIRECT_COMPLETE_QUEST     = "user.directcompletequest"
	GM_SAVE_STORY_FLAG           = "user.savestoryflag"          // 保存story-flag
	GM_DIRECT_COMPLETE_DUTY_TASK = "user.directcompletedutytask" // 直接完成值日生任务
	GM_LEVEL_FINISH              = "user.levelfinish"            // 直接完成关卡(第一个参数指定关卡id, 第二个参数传1表示胜利(不传默认胜利)
	GM_KICKOUT                   = "user.kickout"
	GM_BANNED                    = "user.banned"
)

// 卡牌稀有度
const (
	POTENTIAL_R   int32 = 3 // R
	POTENTIAL_SR  int32 = 4 // SR
	POTENTIAL_SSR int32 = 5 // SSR
	POTENTIAL_SP  int32 = 6 // SP
)

// 抽卡类型
const (
	CARD_POOL_TYPE_ONE     = 1     // 单抽
	CARD_POOL_TYPE_TEN     = 2     // 十抽
	CARD_POOL_TYPE_NORMAL  = 0     // 普通池
	CARD_POOL_TYPE_SPECIAL = 1     // 限定池
	CARD_POOL_NEWBIE       = 0     // 新手池
	CARD_POOL_SPECIAL      = 1     // up付费池
	CARD_POOL_NORMAL       = 2     // 普通付费池
	CARD_POOL_FRIEND       = 3     // 友情池
	CARD_POOL_MUST_VALUE   = 10000 // 抽卡保底值
)

// 道具品质
const (
	ITEM_QUALITY_1 = 1 // 白色 (装备)
	ITEM_QUALITY_2 = 2 // 浅绿色
	ITEM_QUALITY_3 = 3 // 深绿色
	ITEM_QUALITY_4 = 4 // 蓝色 (装备)
	ITEM_QUALITY_5 = 5 // 黄色 (装备)
	ITEM_QUALITY_6 = 6 // 红色 (装备)
)

// 装备属性类型
const (
	EQUIP_ATTR_8  = 8  // 暴击crit
	EQUIP_ATTR_9  = 9  // 暴伤critInjury
	EQUIP_ATTR_10 = 10 // 抵抗resist
	EQUIP_ATTR_11 = 11 // 韧性toughness
)

// 玩家周边数据类型
const (
	PLAYER_HEAD  = 1 // 玩家头像
	PLAYER_TITLE = 2 // 玩家称号
)

// 玩家头像解锁类型
const (
	HEAD_UNLOCK_TYPE_1 = 1 // 获得角色
	HEAD_UNLOCK_TYPE_2 = 2 // 觉醒角色
	HEAD_UNLOCK_TYPE_3 = 3 // 默认解锁
	HEAD_UNLOCK_TYPE_4 = 4 // 性别选择
)

// 关卡中更新卡牌血量方式类型
type UpdateCardHpType int32

const (
	UpdateCardHpType_FULL UpdateCardHpType = 1 // 恢复满血
	UpdateCardHpType_ADD  UpdateCardHpType = 2 // 增加指定血量
	UpdateCardHpType_SET  UpdateCardHpType = 3 // 设置血量
)

// 邮件模板id
const (
	MAIL_TEMPLATE_1 = 1 // 节日奖励邮件
	MAIL_TEMPLATE_2 = 2 // 背包已满邮件
	MAIL_TEMPLATE_3 = 3 // 每日任务奖励邮件
	MAIL_TEMPLATE_4 = 4 // 体力硬上限邮件
	MAIL_TEMPLATE_5 = 5 // 问卷奖励邮件
	MAIL_TEMPLATE_6 = 6 // 月卡每日奖励邮件
)

// 邮件常量定义
const (
	MAIL_RECEIVE_ALL = 0 // 领取所有
	MAIL_RECEIVE_ONE = 1 // 领取单个
	MAIL_DELETE_ALL  = 0 // 删除所有
	MAIL_DELETE_ONE  = 1 // 删除单个
	MAIL_READ_ALL    = 0 // 已读所有
	MAIL_READ_ONE    = 1 // 已读单个

	MAIL_STATUS_NOT_RECEIVE = 0 // 不可领取
	MAIL_STATUS_UNRECEIVE   = 1 // 未领取
	MAIL_STATUS_RECEIVED    = 2 // 已领取

	MAIL_STATUS_UNREAD = 0 // 未读
	MAIL_STATUS_READ   = 1 // 已读

	MAIL_TYPE_1 = 1 // 公告
	MAIL_TYPE_2 = 2 // 节日奖励
	MAIL_TYPE_3 = 3 // 道具溢出
	MAIL_TYPE_4 = 4 // 每日任务奖励代领
	MAIL_TYPE_5 = 5 // gmt邮件

	MAIL_SEND_TYPE_SYSTEM = 0 // 系统
	MAIL_SEND_TYPE_USER   = 1 // 自定义

	MAIL_REWARD_TYPE_NONE  = 0 // 无奖励
	MAIL_REWARD_TYPE_MONEY = 1 // 付费货币奖励
	MAIL_REWARD_TYPE_OTHER = 2 // 其他奖励

	QUESTION_LANG_TYPE_SINGLE = 1 // 单语言
	QUESTION_LANG_TYPE_MULTI  = 2 // 多语言
	QUESTION_STATE_READ       = 0 // 已回答
	QUESTION_STATE_UNREAD     = 1 // 未回答
)

// 联盟
const (
	ALLIANCE_CONTRIBUTION_1 = 1 // 贡献度类型 登录
	ALLIANCE_CONTRIBUTION_2 = 2 // 贡献度类型 每日活动
)

// 战斗中校验特殊规则类型（配置表中按位配置）
const (
	CheckBattleBitPos_doNot_checkBattle int32 = 1 // 第1位 - 不做战斗校验
	CheckBattleBitPos_doNot_saveHp      int32 = 2 // 第2位 - 不做血量更新
	CheckBattleBitPos_doNot_checkFood   int32 = 3 // 第3位 - 不做校验食物消耗
	CheckBattleBitPos_DO_hpFull         int32 = 4 // 第4位 - 恢复满血(仅前端表现, 后端不做逻辑)
)

// Broadcast_type 广播类型
const (
	Broadcast_type int32 = 1 // 抽卡
)

const (
	Camp_Building_Type_Food int32 = 90075 // 食品加工厂
)

// 增加羁绊值的类型
const (
	Realtion_type_win       int32 = 1 // 战斗胜利
	Realtion_type_camp_life int32 = 2 // 营地生活区
)

// 营地技能效果
const (
	Building_Product_Add   int32 = 1 // 建筑产量提升
	Produce_Power_Cost_Sub int32 = 2 // 生产消耗电力减少
	Produce_Time_Cost_Sub  int32 = 3 // 生产耗时减少
	Produce_Double         int32 = 4 // 生产双倍产出
)

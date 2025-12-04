package taptap

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// 系统字段定义
type SystemFieldInfo struct {
	ClientId   string      `json:"client_id"` // 项目的 TapTap Client ID，在 TapTap 开发者中心可以查看该 ID
	Type       string      `json:"type"`      // 数据类型，上报事件时传入 track
	DeviceId   string      `json:"device_id"` // 事件发生时的设备 ID
	UserId     string      `json:"user_id"`   // 事件发生时的账号 ID
	Name       string      `json:"name"`      // 事件名，可传入预置事件或自定义事件
	Properties interface{} `json:"properties"`
}

// 预置属性字段
type PropertyFieldInfo struct {
	OS          string `json:"os"`           // 操作系统（支持传入 Android、iOS、Windows、Mac）
	DeviceId1   string `json:"device_id1"`   // 预留设备 ID 槽位（iOS SDK 传入 IDFA，Android SDK 传入 IMEI）
	DeviceId2   string `json:"device_id2"`   // 预留设备 ID 槽位（Android SDK 传入 Google 广告 ID）
	DeviceId3   string `json:"device_id3"`   // 预留设备 ID 槽位（Android SDK 传入 Android ID）
	DeviceId4   string `json:"device_id4"`   // 预留设备 ID 槽位（Android SDK 传入 OAID）
	Width       int32  `json:"width"`        // 屏幕宽度
	Height      int32  `json:"height"`       // 屏幕高度
	DeviceModel string `json:"device_model"` // 设备型号（设备厂商+设备型号）
	OsVersion   string `json:"os_version"`   // 操作系统版本
	Network     string `json:"network"`      // 网络类型（WiFi传入2，未知传入3，2G传入4，3G传入5，4G传入6）
	Channel     string `json:"channel"`      // 分包渠道
	AppVersion  string `json:"app_version"`  // App版本
	SdkVersion  string `json:"sdk_version"`  // SDK版本
	ServerId    string `json:"#server_id"`   // 服务器ID
	ServerName  string `json:"#server_name"` // 服务器名
	CreateTS    int64  `json:"#create_ts"`   // 创建时间戳
}

// 创建游戏角色(rolecreate)
type RoleCreate struct {
	*PropertyFieldInfo
	RoleId string `json:"#role_id"` // 角色唯一id
}

// 游戏角色登录(rolelogin)
type RoleLogin struct {
	*PropertyFieldInfo
	RoleId   string  `json:"#role_id"`   // 角色唯一id
	Level    int32   `json:"#level"`     // 等级
	VipLevel int32   `json:"#vip_level"` // vip等级
	Recharge float32 `json:"#recharge"`  // 累计充值金额
	Language string  `json:"#language"`  // 游戏内语言，中文简体=zh_CN,中文繁体=zh_TW
}

// 游戏角色退出(rolelogout)
type RoleLogout struct {
	*PropertyFieldInfo
	RoleId   string  `json:"#role_id"`   // 角色唯一id
	GameTime string  `json:"#game_time"` // 本次游戏时长，单位秒
	Level    int32   `json:"#level"`     // 等级
	VipLevel int32   `json:"#vip_level"` // vip等级
	Recharge float32 `json:"#recharge"`  // 累计充值金额
}

// 在线人数，每分钟上报一次(online)
type Online struct {
	*PropertyFieldInfo
	OnlineCount int64  `json:"online"`      // 当前在线人数
	ServerTag   string `json:"#server_tag"` // 服务器标识符，可以通过描述找到服务器
}

// 角色充值行为(purchase)
type Purchase struct {
	*PropertyFieldInfo
	RoleId   string  `json:"#role_id"`   // 角色唯一id
	ItemId   string  `json:"#item_id"`   // 商品id 优先id 其次名称
	Level    int32   `json:"#level"`     // 角色等级
	VipLevel int32   `json:"#vip_level"` // vip等级
	IsTest   int32   `json:"#is_test"`   // 测试订单标记 测试订单为1 ，正常订单为0 ，订阅订单为2
	OrderId  string  `json:"order_id"`   // 充值订单号
	Recharge float32 `json:"#recharge"`  // 累计充值金额，美金单位
	Currency string  `json:"#currency"`  // 充值货币单位
	Price    float32 `json:"#price"`     // 充值货币金额
	Iap      string  `json:"#iap"`       // 商店后台购买项ID
	PayType  string  `json:"#pay_type"`  // 支付方式 SDK支付接口的"pay_type" unisdk支付接口的"plat"
}

// 角色退款
type Refund struct {
	*PropertyFieldInfo
	RoleId   string  `json:"#role_id"`   // 角色唯一id
	OrderId  string  `json:"#order_id"`  // 充值订单号
	Level    int32   `json:"#level"`     // 角色等级
	VipLevel int32   `json:"#vip_level"` // vip等级
	Recharge float32 `json:"#recharge"`  // 累计充值金额，美金单位
}

// 角色一级代币变化
type MoneyFlow struct {
	*PropertyFieldInfo
	RoleId      string  `json:"#role_id"`      // 角色唯一id
	MoneyBefore int64   `json:"#money_before"` // 变化前货币数值
	MoneyAfter  int64   `json:"#money_after"`  // 变化后货币数值
	Flow        string  `json:"#flow"`         // 货币流向 获得为"in" 消耗为"out" 退款，坏账导致的钻石扣除 需上报一条货币流向为"in" 的消息，变化量为负
	Action      string  `json:"#action"`       // 触发货币变动的行为，我们需要编写一份文档表示对应的行为
	Level       int32   `json:"#level"`        // 角色等级
	VipLevel    int32   `json:"#vip_level"`    // vip等级
	MoneyType   string  `json:"#money_type"`   // 非必须 货币类型，如果存在多种获得的货币类型则需要
	Item        string  `json:"#item"`         // 非必须 如果行为的结果是获得道具资源，则需要填写资源id
	Recharge    float32 `json:"#recharge"`     // 累计充值金额，美金单位
}

// 角色资源变化
type ResourceFlow struct {
	*PropertyFieldInfo
	RoleId         string  `json:"#role_id"`         // 角色唯一id
	ResourceId     string  `json:"#resource_id"`     // 资源id
	ResourceBefore int64   `json:"#resource_before"` // 变化前资源数值
	ResourceAfter  int64   `json:"#resource_after"`  // 变化后资源数值
	Flow           string  `json:"#flow"`            // 资源流向 获得为"in" 消耗为"out"
	Action         string  `json:"#action"`          // 触发资源变动的行为，我们需要编写一份文档表示对应的行为
	Level          int32   `json:"#level"`           // 角色等级
	VipLevel       int32   `json:"#vip_level"`       // vip等级
	Recharge       float32 `json:"#recharge"`        // 累计充值金额，美金单位
}

// 角色物品变化
type ItemFlow struct {
	*PropertyFieldInfo
	RoleId     string  `json:"#role_id"`     // 角色唯一id
	ItemFlow   string  `json:"#item_flow"`   // 物品流向 获得为"in" 消耗为"out"
	ItemId     string  `json:"#item_id"`     // 物品id
	ItemCount  int32   `json:"#item_count"`  // 变化的物品数量
	Level      int32   `json:"#level"`       // 角色等级
	VipLevel   int32   `json:"#vip_level"`   // vip等级
	Action     string  `json:"#action"`      // 触发物品变动的行为，我们需要编写一份文档表示对应的行为
	ItemBefore int64   `json:"#item_before"` // 非必须 变化前资源数值
	ItemAfter  int64   `json:"#item_after"`  // 非必须 变化后资源数值
	Recharge   float32 `json:"#recharge"`    // 累计充值金额，美金单位
}

// 角色升级
type LevelUp struct {
	*PropertyFieldInfo
	RoleId   string  `json:"#role_id"`   // 角色唯一id
	Level    int32   `json:"#level"`     // 角色等级
	VipLevel int32   `json:"#vip_level"` // vip等级
	Recharge float32 `json:"#recharge"`  // 累计充值金额，美金单位
}

// 角色封禁
type BanRole struct {
	*PropertyFieldInfo
	RoleId     string `json:"#role_id"`     // 角色唯一id
	UnlockTime string `json:"#unlock_time"` // 解封时间
	Reason     string `json:"#reason"`      // 处理原因
	BanSource  string `json:"#ban_source"`  // 封禁来源 例如: GMT 手动
	BanReason  string `json:"#ban_reason"`  // 封禁原因 （内部展示 涩涩，涉政）
}

// 角色解封
type UnBanRole struct {
	*PropertyFieldInfo
	RoleId string `json:"#role_id"` // 角色唯一id
	Reason string `json:"#reason"`  // 处理原因
}

// 抽卡结果(cardextract)
type CardExtract struct {
	*PropertyFieldInfo
	PoolId      int32  `json:"#pool_id"`         // 抽卡卡池id
	ExtractType int32  `json:"#extract_type"`    // 抽卡类型，单抽为1，十连抽为2
	Cost        string `json:"#cost,omitempty"`  // 道具消耗 map[int32]int32
	Cards       string `json:"#cards,omitempty"` // 抽卡结果 []int32
}

// 玩家从全局邮件中领取到私人邮件(glotopersonalmail)
type GlobalToPersonalMail struct {
	*PropertyFieldInfo
	MailIds string `json:"#mail_ids,omitempty"` // 待领取的邮件id列表 []int64
}

// 邮件领取(mailreceive)
type MailReceive struct {
	*PropertyFieldInfo
	MailIds string `json:"#mail_ids,omitempty"` // 待领取的邮件id列表 []int64
	Reward  string `json:"#reward,omitempty"`   // 领取的奖励列表 map[int32]int32
}

// 邮件删除(maildelete)
type MailDelete struct {
	*PropertyFieldInfo
	MailIds string `json:"#mail_ids,omitempty"` // 已删除的邮件id列表 []int64
}

// 完成剧情物件交互(completequestobject)
type CompleteQuestObject struct {
	*PropertyFieldInfo
	ObjectId       int32  `json:"#object_id"`                 //物件配置id
	StepId         int32  `json:"#step_id"`                   //对应的步骤id
	QuestId        int32  `json:"#quest_id"`                  //对应的任务id
	CompleteQuest  string `json:"#complete_quest,omitempty"`  //完成的任务id列表 []int32
	CompleteStep   string `json:"#complete_step,omitempty"`   //当前任务完成的步骤id列表 []int32
	CompleteObject string `json:"#complete_object,omitempty"` //当前步骤完成的物件id列表 []int32
}

// 卡牌突破
type CardBreakThrough struct {
	*PropertyFieldInfo
	CardId   uint32 `json:"#card_id"`   // 突破卡牌id
	BeforeLv uint32 `json:"#before_lv"` // 突破前等级
	AfterLv  uint32 `json:"#after_lv"`  // 突破后等级
}

// 技能升级
type CardSkillUpgrade struct {
	*PropertyFieldInfo
	CardId   uint32 `json:"#card_id"`   // 卡牌id
	Index    uint32 `json:"#index"`     // 升级的技能位
	BeforeLv int32  `json:"#before_lv"` // 升级前的技能等级
	AfterLv  int32  `json:"#after_lv"`  // 升级后的技能等级
}

// 卡牌觉醒
type CardCompound struct {
	*PropertyFieldInfo
	CardId   uint32 `json:"#card_id"`   // 突破卡牌id
	BeforeLv uint32 `json:"#before_lv"` // 觉醒前等级
	AfterLv  uint32 `json:"#after_lv"`  // 觉醒后等级
}

// 性格突破
type CardCharacterBreak struct {
	*PropertyFieldInfo
	CardId   uint32 `json:"#card_id"`   // 突破卡牌id
	BeforeLv uint32 `json:"#before_lv"` // 升级前等级
	AfterLv  uint32 `json:"#after_lv"`  // 升级后等级
}

// 体力恢复
type CardEatFood struct {
	*PropertyFieldInfo
	CardId   uint32              `json:"#card_id"`   // 突破卡牌id
	Items    []*cmd.KeyValueItem `json:"#items"`     // 使用的食物道具列表
	BeforeHp uint32              `json:"#before_hp"` // 喂养前的体力值
	AfterHp  uint32              `json:"#after_hp"`  // 喂养后的体力值
}

// 卡牌升级
type CardLevelUp struct {
	*PropertyFieldInfo
	CardId   int32  `json:"#card_id"`         // 突破卡牌id
	Items    string `json:"#items,omitempty"` // 使用的食物道具列表 []*cmd.KeyValueItem
	AddExp   int32  `json:"#add_exp"`         // 本次增加的经验值
	BeforeLv uint32 `json:"#before_lv"`       // 升级前等级
	AfterLv  uint32 `json:"#after_lv"`        // 升级后等级
}

// 签到
type DaySign struct {
	*PropertyFieldInfo
	GroupId  int32  `json:"#group_id"`         // 签到组id
	Params   int32  `json:"#params"`           // 特殊签到参数，无参数则为0
	Category int32  `json:"#category"`         // 签到类型
	Counter  int32  `json:"#counter"`          // 当前签到次数
	Reward   string `json:"#reward,omitempty"` // 签到奖励 map[int32]int32
}

// 记录引导点
type TutorialAddRecord struct {
	*PropertyFieldInfo
	TutorialType uint32 `json:"#tutorial_type"` // 引导点类型，1=主引导，2=功能引导
	TutorialId   uint32 `json:"#tutorial_id"`   // 引导点id
}

// 装备升级
type EquipLevelUp struct {
	*PropertyFieldInfo
	EquipId     uint64 `json:"#equip_id"`                // 升级的装备唯一id
	ConfigId    int32  `json:"#config_id"`               // 装备配表id
	EquipDelIds string `json:"#equip_del_ids,omitempty"` // 待合并的装备id列表 []uint64
	AddExp      int32  `json:"#add_exp"`                 // 增加的经验值
	BeforeLv    int32  `json:"#before_lv"`               // 升级前等级
	AfterLv     int32  `json:"#after_lv"`                // 升级后等级
}

// 装备穿戴
type EquipWear struct {
	*PropertyFieldInfo
	EquipId  uint64 `json:"#equip_id"`  // 装备唯一id
	ConfigId int32  `json:"#config_id"` // 装备配表id
	CardId   uint32 `json:"#card_id"`   // 穿戴的目标卡牌id
}

// 装备卸下
type EquipUnWear struct {
	*PropertyFieldInfo
	EquipId  int64  `json:"#equip_id"`  // 装备唯一id
	ConfigId int32  `json:"#config_id"` // 装备配表id
	CardId   uint32 `json:"#card_id"`   // 卸下的目标卡牌id
}

// 装备删除
type EquipDelete struct {
	*PropertyFieldInfo
	EquipId  uint64 `json:"#equip_id"`  // 装备唯一id
	ConfigId int32  `json:"#config_id"` // 配表id
	Lv       int32  `json:"#level"`     // 当前等级
	Exp      int32  `json:"#exp"`       // 当前经验
	SkillId  int32  `json:"#skill_id"`  // 技能id
	MainAttr int32  `json:"#main_attr"` // 主属性类型
	SubAttr  int32  `json:"#sub_attr"`  // 副属性类型
}

// 装备获得
type EquipCreate struct {
	*PropertyFieldInfo
	EquipId  int64 `json:"#equip_id"`  // 装备唯一id
	ConfigId int32 `json:"#config_id"` // 配表id
	Lv       int32 `json:"#level"`     // 当前等级
	Exp      int32 `json:"#exp"`       // 当前经验
	SkillId  int32 `json:"#skill_id"`  // 技能id
	MainAttr int32 `json:"#main_attr"` // 主属性类型
	SubAttr  int32 `json:"#sub_attr"`  // 副属性类型
}

// 穿戴皮肤(skindress)
type SkinDress struct {
	*PropertyFieldInfo
	CardId int32 `json:"#card_id"` // 卡牌id
	SkinId int32 `json:"#skin_id"` // 皮肤id
}

// 获得皮肤(skincreate)
type SkinCreate struct {
	*PropertyFieldInfo
	CardId int32 `json:"#card_id"` // 卡牌id
	SkinId int32 `json:"#skin_id"` // 皮肤id
}

// 编队(troopoperate)
type TroopOperate struct {
	*PropertyFieldInfo
	TroopType       int32  `json:"#troop_type"`                 // 玩法类型id
	TroopId         int32  `json:"#troop_id"`                   // 队伍编号
	BeforePositions string `json:"#before_positions,omitempty"` // 修改前队伍卡牌站位数据 []int32
	AfterPositions  string `json:"#after_positions,omitempty"`  // 修改后队伍卡牌站位数据 []int32
	SubType         int32  `json:"#sub_type"`                   // 玩法子类型编号
}

// 食物编辑(foodoperate)
type FoodOperate struct {
	*PropertyFieldInfo
	TroopType   int32  `json:"#troop_type"`             // 玩法类型id
	BeforeFoods string `json:"#before_foods,omitempty"` // 修改前的食物列表数据 []int32
	AfterFoods  string `json:"#after_foods,omitempty"`  // 修改后的食物列表数据 []int32
}

// 使用道具(useitem)
type UseItem struct {
	*PropertyFieldInfo
	ItemId  int32 `json:"#com_item_id"` // 使用道具的id
	ItemNum int32 `json:"#item_num"`    // 使用数量
}

// 销毁过期道具(destroyexpireitem)
type DestroyExpireItem struct {
	*PropertyFieldInfo
	Id       int64  `json:"#id"`                 //唯一id
	ItemId   int32  `json:"#com_item_id"`        //销毁道具配置id
	ItemNum  int32  `json:"#item_num"`           //销毁数量
	Expire   int64  `json:"#expire"`             //过期时间戳
	Exchange string `json:"#exchange,omitempty"` //补偿道具列表 map[int32]int32
}

// 购买道具(itembuy)
type ItemBuy struct {
	*PropertyFieldInfo
	ItemId    int32 `json:"#item_id"`    // 购买的道具id
	ItemNum   int32 `json:"#item_num"`   // 购买数量
	MoneyType int32 `json:"#money_type"` // 消耗的货币类型
	MoneyNum  int32 `json:"#money_num"`  // 消耗货币数量
}

// 道具兑换货币(currencyexchange)
type CurrencyExchange struct {
	*PropertyFieldInfo
	MoneyType   int32  `json:"#money_type"`     //兑换的货币类型
	Cost        string `json:"#cost,omitempty"` //消耗道具列表 map[uint64]uint32
	ExchangeNum uint64 `json:"#exchange_num"`   //可兑换货币数量
}

// 二级货币购买(currencybuy)
type CurrencyBuy struct {
	*PropertyFieldInfo
	MoneyType   int32 `json:"#money_type"`   //购买的货币类型
	ExchangeNum int32 `json:"#exchange_num"` //获得二级货币数量
	CostType    int32 `json:"#cost_type"`    //消耗一级货币类型
	CostNum     int32 `json:"#cost_num"`     //消耗一级货币数量
}

// 修改值日生(changedutycard)
type ChangeDutyCard struct {
	*PropertyFieldInfo
	BeforeCard int32 `json:"#before_card_id"` // 修改前卡牌
	AfterCard  int32 `json:"#after_card_id"`  // 修改后卡牌
}

// 领取任务奖励(receivedailyreward)
type ReceiveDailyReward struct {
	*PropertyFieldInfo
	TaskId   int32  `json:"#task_id"`          //任务id
	TaskType int32  `json:"#task_type"`        //任务类型
	CondId   int32  `json:"#cond_id"`          //完成条件类型
	Target   int32  `json:"#target"`           //目标值
	Active   int32  `json:"#active"`           //领取的活跃度
	Reward   string `json:"#reward,omitempty"` //领取的奖励 map[int32]int32
}

// 领取活跃度奖励(receiveactivereward)
type ReceiveActiveReward struct {
	*PropertyFieldInfo
	ActiveNode int32  `json:"#active_node"`      //待领取的活跃度节点值
	ActiveType int32  `json:"#active_type"`      //活跃度类型
	Reward     string `json:"#reward,omitempty"` //领取的奖励 map[int32]int32
}

// 修建建筑
type MakeFuncBuilding struct {
	*PropertyFieldInfo
	ItemId       int32  `json:"#com_item_id"`    //功能建筑道具id
	Id           int32  `json:"#id"`             //建筑唯一id
	BuildingId   int64  `json:"#building_id"`    //建筑配置id
	Lv           int32  `json:"#level"`          //建筑等级
	BuildingType int32  `json:"#building_type"`  //建筑类型id
	Costs        string `json:"#cost,omitempty"` //建造消耗材料 map[int32]int32
}

// 建筑升级
type BuildingLevelUp struct {
	*PropertyFieldInfo
	Id           int32  `json:"#id"`             //建筑唯一id
	BuildingId   int64  `json:"#building_id"`    //建筑配置id
	BuildingType int32  `json:"#building_type"`  //建筑类型id
	Costs        string `json:"#cost,omitempty"` //建造消耗材料 map[int32]int32
	BeforeLv     int32  `json:"#before_lv"`      //升级前等级
	AfterLv      int32  `json:"#after_lv"`       //升级后等级
}

// 熔炉熔炼
type CampFurnaceoPerate struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`                //建筑唯一id
	BuildingId int64  `json:"#building_id"`       //建筑id
	Lv         int32  `json:"#level"`             //建筑等级
	Formula    string `json:"#formula,omitempty"` //熔炼产出材料 []*cmd.PPlayerCampFunctionBuildingFormula
	StartTs    int64  `json:"#start_ts"`          //队列开始时间戳
	EndTs      int64  `json:"#end_ts"`            //队列结束时间戳
	Costs      string `json:"#cost,omitempty"`    //建造消耗材料   map[int32]int32
	QueueId    int64  `json:"#queue_id"`          //队列id
}

// 熔炉队列取消
type CancelFurnaceQueue struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`                //建筑唯一id
	BuildingId int64  `json:"#building_id"`       //建筑id
	Lv         int32  `json:"#level"`             //建筑等级
	QueueId    int64  `json:"#queue_id"`          //队列id
	Formula    string `json:"#formula,omitempty"` //消耗材料 []*cmd.PPlayerCampFunctionBuildingFormula
	Costs      string `json:"#cost,omitempty"`    //建造消耗材料 map[uint32]uint32
}

// 领取队列奖励
type GetqueueReward struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`               //建筑唯一id
	BuildingId int64  `json:"#building_id"`      //建筑id
	Lv         int32  `json:"#level"`            //建筑等级
	QueueId    int64  `json:"#queue_id"`         //队列id
	Reward     string `json:"#reward,omitempty"` //产出奖励 map[uint32]uint32
}

// 光合树收获
type CamptreeReward struct {
	*PropertyFieldInfo
	Id          int32  `json:"#id"`               //建筑唯一id
	BuildingId  int64  `json:"#building_id"`      //建筑配置id
	Reward      string `json:"#reward,omitempty"` //领取的奖励 map[int32]int32
	Lv          int32  `json:"#level"`            //光合树当前等级
	BeforeEndTs int64  `json:"#before_end_ts"`    //领取前的结束时间戳
	AfterEndTs  int64  `json:"#after_end_ts"`     //领取后的结束时间戳
	UseTime     int64  `json:"#use_time"`         //奖励的折算时间
}

// 材料研究转换
type CampmaterialConver struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`                //建筑唯一id
	BuildingId int64  `json:"#building_id"`       //建筑id
	Lv         int32  `json:"#level"`             //建筑等级
	Formula    string `json:"#formula,omitempty"` //消耗材料 map[int32]int32
	Reward     string `json:"#reward,omitempty"`  //产出奖励 map[uint32]uint32
}

// 食物制作
type CampMakeFood struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`          //建筑唯一id
	BuildingId int64  `json:"#building_id"` //建筑id
	Lv         int32  `json:"#level"`       //建筑等级
	FoodId     int32  `json:"#food_id"`     //食谱id
	Cost       string `json:"#cost"`        //消耗材料
	Reward     string `json:"#reward"`      //食物产出
}

// 装备打造
type CampequipFoundry struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`                //建筑唯一id
	BuildingId int64  `json:"#building_id"`       //建筑id
	Lv         int32  `json:"#level"`             //建筑等级
	Formula    string `json:"#formula,omitempty"` //消耗材料 map[int32]int32
	Equips     string `json:"#equips,omitempty"`  //装备唯一id列表 []int64
}

// 营地角色上阵
type CamproleChange struct {
	*PropertyFieldInfo
	Count      int    `json:"#counter"`               //当前上阵数量上限
	BeforeCard string `json:"#before_card,omitempty"` //上阵前卡牌列表 []int32
	AfterCard  string `json:"#after_card,omitempty"`  //上阵后卡牌列表 []uint32
}

// 家具打造
type CampBuildingFoundry struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`             //建筑唯一id
	BuildingId int64  `json:"#building_id"`    //建筑配置id
	Lv         int32  `json:"#level"`          //建筑等级
	ItemId     int32  `json:"#com_item_id"`    //待制造的家具id
	Num        int32  `json:"#counter"`        //待制造的数量
	Cost       string `json:"#cost,omitempty"` //消耗材料 map[int32]int32
}

// 商人兑换奖励
type CampTraderExchange struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`               //建筑唯一id
	BuildingId int64  `json:"#building_id"`      //建筑id
	Lv         int32  `json:"#level"`            //建筑等级
	TraderId   int32  `json:"#trader_id"`        //兑换清单id
	Category   int32  `json:"#category"`         //清单分类
	Quality    int32  `json:"#quality"`          //清单品质
	Costs      string `json:"#cost,omitempty"`   //消耗 map[int32]int32
	Reward     string `json:"#reward,omitempty"` //奖励 cmd.KeyValueItem
}

// 商人清单刷新
type CampTraderRefresh struct {
	*PropertyFieldInfo
	Id         int32  `json:"#id"`                    //建筑唯一id
	BuildingId int64  `json:"#building_id"`           //建筑id
	Lv         int32  `json:"#level"`                 //建筑等级
	BeforeList string `json:"#before_data,omitempty"` //刷新前的清单列表 []*cmd.PPlayerCampTraderList
	AfterList  string `json:"#after_data,omitempty"`  //刷新后的清单列表 []*cmd.PPlayerCampTraderList
}

// PPlayerCampTraderListTemp 商人清单刷新列表
type PPlayerCampTraderListTemp struct {
	Id       int32  `json:"#id"`               //建筑唯一id
	Category int32  `json:"#category"`         //1=今日强推，2=今日杂货
	Status   int32  `json:"#status"`           //1=未兑换，2=已兑换
	Quality  int32  `json:"#quality"`          //品质
	Costs    string `json:"#cost,omitempty"`   //消耗物 []*cmd.KeyValueItem
	Reward   string `json:"#reward,omitempty"` //奖励 KeyValueItem
}

func NewPPlayerCampTraderListTemp(id, category, status, quality int32, costs, reward string) *PPlayerCampTraderListTemp {
	return &PPlayerCampTraderListTemp{
		Id:       id,
		Category: category,
		Status:   status,
		Quality:  quality,
		Costs:    costs,
		Reward:   reward,
	}
}

// 建筑卡牌驻守
type CampBuildingUpcard struct {
	*PropertyFieldInfo
	Id         int32 `json:"#id"`             //建筑唯一id
	BuildingId int64 `json:"#building_id"`    //建筑id
	Lv         int32 `json:"#level"`          //建筑等级
	BeforeCard int32 `json:"#before_card_id"` //上阵前的卡牌
	AfterCard  int32 `json:"#after_card_id"`  //上阵后的卡牌
}

// 建筑卡牌下阵
type CampBuildingDownCard struct {
	*PropertyFieldInfo
	Id         int32 `json:"#id"`             //建筑唯一id
	BuildingId int64 `json:"#building_id"`    //建筑配置id
	Lv         int32 `json:"#level"`          //建筑等级
	BeforeCard int32 `json:"#before_card_id"` //上阵前的卡牌
	AfterCard  int32 `json:"#after_card_id"`  //上阵后的卡牌
}

// 布局修改
type CampSaveLayout struct {
	*PropertyFieldInfo
	BeforeAtmosphere int32 `json:"#before_value"` //切换前氛围值
	AfterAtmosphere  int32 `json:"#after_value"`  //切换后氛围值
}

// 切换布局方案
type CampSwitchLayout struct {
	*PropertyFieldInfo
	CampId           int32 `json:"#camp_id"`      //营地id
	LayoutId         int32 `json:"#layout_id"`    //布局id
	BeforeAtmosphere int32 `json:"#before_value"` //切换前氛围值
	AfterAtmosphere  int32 `json:"#after_value"`  //切换后氛围值
}

// 大地图-进入
type LevelEnter struct {
	*PropertyFieldInfo
	LevelId int32  `json:"#level_id"` // 关卡id
	TroopId uint32 `json:"#troop_id"` // 队伍id
}

// 大地图-退出
type LevelExit struct {
	*PropertyFieldInfo
	BattleResult int64  `json:"#result_type"`     // 战斗结果 cmd.BattleResult
	LevelId      int32  `json:"#level_id"`        // 关卡id
	BattleCards  string `json:"#cards,omitempty"` // 卡牌信息 []*cmd.PPlayerBattleCard
	Foods        string `json:"#items,omitempty"` // 食物itemId列表 []*int32
	Collection   uint32 `json:"#cost_value"`      // 消耗采集点数
}

// 大地图-开始战斗
type LevelBattleStart struct {
	*PropertyFieldInfo
	BattleId         uint64 `json:"#id"`      // 战斗id
	BattleRandomSeed uint32 `json:"#seed_id"` // 战斗随机种子
}

// 大地图-结束战斗
type LevelBattleEnd struct {
	*PropertyFieldInfo
	NiwaId        int32  `json:"#map_id"`            // 地图id
	EventId       int32  `json:"#event_id"`          // 事件Id
	QuestObjectId uint32 `json:"#object_id"`         // 物件id
	Monster       string `json:"#monster,omitempty"` // 怪物 []*cmd.PClientLevelBattleEventMonster
	BattleResult  int64  `json:"#result_type"`       // 战斗结果 cmd.BattleResult
	BattleCards   string `json:"#cards,omitempty"`   // 卡牌信息 []*cmd.PPlayerBattleCard
	Foods         string `json:"#items,omitempty"`   // 食物itemId列表 []*int32
	CostFoods     string `json:"#cost,omitempty"`    // 消耗食物信息 []*cmd.KeyValueItem
}

// 大地图-事件处理
type LevelMapEvent struct {
	*PropertyFieldInfo
	NiwaId        int32  `json:"#map_id"`    // 地图id
	EventId       int32  `json:"#event_id"`  // 事件Id
	QuestObjectId uint32 `json:"#object_id"` // 物件id
}

// 大地图-选择路径
type LevelChooseNiwaPath struct {
	*PropertyFieldInfo
	PathId uint32 `json:"#map_id"` // 地图id
}

// 大地图-解锁传送点
type LevelUnlockPoint struct {
	*PropertyFieldInfo
	//NiwaId        int32  `json:"niwa_id"`         // 地图id
	//EventId       int32  `json:"event_id"`        // 事件Id
	UnlockedPointId string `json:"#ids"` // 节点id []int32
}

// 大地图-返回大本营
type LevelBackToBC struct {
	*PropertyFieldInfo
	//NiwaId        int32  `json:"niwa_id"`         // 地图id
	//EventId       int32  `json:"event_id"`        // 事件Id
	//UnlockedPointId []int32 `json:"unlocked_point_id"` // 节点id
}

// 日替-获取日替副本列表
type CampaignList struct {
	*PropertyFieldInfo
	OpenCampaigns string `json:"#ids"` // 日替关卡当日开放列表 []int32
	//GeneralCampaigns *cmd.PClientGeneralCampaign `json:"general_campaigns"` // 日替关卡数据
}

// 日替-获取日替副本列表
type CampaignEnter struct {
	*PropertyFieldInfo
	CampaignId    int32  `json:"#level_id"` // 副本id
	SubCampaignId int32  `json:"#sub_type"` // 副本子类型
	Teams         string `json:"#teams"`    // 队伍信息 []*cmd.GeneralCampaignTeam
}

// 日替-开始战斗
type CampaignBattleStart struct {
	*PropertyFieldInfo
	BattleId         uint64 `json:"#id"`      // 战斗id
	BattleRandomSeed uint32 `json:"#seed_id"` // 战斗随机种子
}

// 日替-结束战斗
type CampaignBattleEnd struct {
	*PropertyFieldInfo
	CampaignId    int32 `json:"#level_id"`    // 关卡id
	SubCampaignId int32 `json:"#sub_type"`    // 子关卡id
	BattleScore   int32 `json:"#score"`       // 得分
	BattleResult  int64 `json:"#result_type"` // 战斗结果 cmd.BattleResult
}

// 商店-列表
type ShopList struct {
	*PropertyFieldInfo
	ShopIds string `json:"#ids"` // 商店列表 []uint32
}

// 商店-列表
type ShopInfo struct {
	*PropertyFieldInfo
	ShopId        int32  `json:"#id"`              // id
	ShopLayer     int32  `json:"#layout_id"`       // 商店层级
	ShopGoodsInfo string `json:"#items,omitempty"` // 商品ids []*cmd.ShopGoodsInfo
	ExpireTimeSec int64  `json:"#expire"`          // 过期时间戳
}

// 商店-购买物品
type ShopBuy struct {
	*PropertyFieldInfo
	ShopId    int32  `json:"#id"`       // 商店id
	GoodsId   int32  `json:"#goods_id"` // 商品id
	GoodsInfo string `json:"#items"`    // 商品信息 *cmd.ShopGoodsInfo
}

// 玩家体力变化
type StaminaChange struct {
	*PropertyFieldInfo
	Action    int32  `json:"#action_id"`    // 变化来源
	Num       int32  `json:"#counter"`      // 变化数量
	BeforeNum int32  `json:"#before_value"` // 变化前数量
	AfterNum  int32  `json:"#after_value"`  // 变化后数量
	Flow      string `json:"#flow"`         // 流向，获得为"in" 消耗为"out"
	Level     uint32 `json:"#level"`        // 玩家等级
}

// 新增全局邮件
type GlobalMailAddInfo struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
	MailId         int64  `json:"#mail_id"`                  // 新增的全局邮件id
}

// 删除全局邮件
type GlobalMailDelInfo struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
	MailId         int64  `json:"#mail_id"`                  // 删除的全局邮件id
}

// 服务启动
type ServerStart struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
}

// 服务退出
type ServerStop struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
	LiveTime       int64  `json:"#livetime"`                 // 生存时间
}

// 服务定时器
type ServerHour struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
	LiveTime       int64  `json:"#livetime"`                 // 生存时间
}

// 服务配置事件
type Confevent struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
	EventId        string `json:"#id_str"`                   // 事件id
	EventData      string `json:"#event_data,omitempty"`     // 配置事件内容 map[string]string
}

// 服务热更新
type ServeReload struct {
	*PropertyFieldInfo
	AppId          string `json:"#appid"`                    // 服务类型标识
	AppVersion     string `json:"#app_version"`              // 程序版本号
	ClientVersion  string `json:"#client_version,omitempty"` // 客户端版本
	RollingVersion string `json:"#rolling_version"`          // 滚动更新版本
	Files          string `json:"#file_list"`                // 热更文件列表 []string
	Fails          string `json:"#fail_list"`                // 热更失败文件列表 []string
}

// GM指令
type GmCmd struct {
	*PropertyFieldInfo
	Cmd          string `json:"#key"`                // 命令字符串
	Param        string `json:"#params"`             // 命令参数 []string
	GmUser       string `json:"#user_str,omitempty"` // gm用户
	User         int    `json:"#user,omitempty"`     // useractor id
	Ip           string `json:"#ip,omitempty"`       // 请求源IP地址
	ResultStatus int    `json:"#status"`             // 执行状态
	Result       string `json:"#result,omitempty"`   // 执行结果
}

// 用户容器上线下
type UserActor struct {
	*PropertyFieldInfo
	Id       string `json:"#id_str"`   // id
	Type     int64  `json:"#type_int"` // 1：active; 0: deactive
	LiveTime int64  `json:"#livetime"` // 生存时间
}

// DB读写失效
type DbFail struct {
	*PropertyFieldInfo
	Key  string `json:"#key"`      // 数据key
	Db   string `json:"#db"`       // redis/mongo
	Type string `json:"#type_str"` // read/write
}

// login网络延迟
type LoginDelay struct {
	*PropertyFieldInfo
	MsgId int32 `json:"#msg_id"` // 消息id
	Delay int64 `json:"#delay"`  // 延迟时间
}

// gate网络延迟
type GateDelay struct {
	*PropertyFieldInfo
	MsgId int32 `json:"#msg_id"` // 消息id
	Delay int64 `json:"#delay"`  // 延迟时间
}

// 获取版本信息(getversion)
type GetVersion struct {
	*PropertyFieldInfo
	Version  string `json:"#version_info"`
	Platform string `json:"#platform"`
}

// loginServer登录(accountlogin)
type AccountLogin struct {
	*PropertyFieldInfo
	Channel int32  `json:"#login_chan"`
	Extra   string `json:"#extra"`
}

// 长链进入(tcpenter)
type TcpEnter struct {
	*PropertyFieldInfo
}

// 断线重连(reconnect)
type Reconnect struct {
	*PropertyFieldInfo
}

// 重复登录(repeatedlogin)
type RepeatedLogin struct {
	*PropertyFieldInfo
}

package logic

const (
	RET_CODE_SUCCESS     = 0
	RET_CODE_SUCCESS_200 = 200
	RET_CODE_FAIL_ALL    = -1
	RET_CODE_FAIL        = -2
)

// 请求类型定义
const (
	REQUEST_TYPE_QUERY_USER               = "query_user"
	REQUEST_TYPE_MODIFY_WHITE_LIST        = "gmt_player_white_list"
	REQUEST_TYPE_SEND_USER_MAIL           = "send_mail"
	REQUEST_TYPE_SEND_SYS_MAIL            = "send_global_mail"
	REQUEST_TYPE_SEND_MULTI_LANG_MAIL     = "send_multi_lang_mail"
	REQUEST_TYPE_SEND_MULTI_LANG_SYS_MAIL = "send_multi_lang_global_mail"
	REQUEST_TYPE_ADD_SINGLE_RESOURCE      = "gm_add_resource"
	REQUEST_TYPE_SUB_SINGLE_RESOURCE      = "gm_sub_resource"
	REQUEST_TYPE_ADD_BATCH_RESOURCE       = "gm_add_batch_resource"
	REQUEST_TYPE_SUB_BATCH_RESOURCE       = "gm_sub_batch_resource"
	REQUEST_TYPE_SEND_GIFT_PACKAGE        = "give_iap_package"
	REQUEST_TYPE_SEND_MULTI_RESOURCE      = "send_laotie_fuli"
	REQUEST_TYPE_RESET_USER_NAME          = "init_user_name"
	REQUEST_TYPE_QUERY_USER_GM_LIST       = "query_cmd_list"
	REQUEST_TYPE_QUERY_GLOBAL_GM_LIST     = "query_cmd_list2"
	REQUEST_TYPE_QUERY_USER_EXCUTE_CMD    = "query_excute_cmd"
	REQUEST_TYPE_QUERY_GLOBAL_EXCUTE_CMD  = "query_excute_cmd2"
)

// 错误提示文本
const (
	SUCCESS               = "success"
	FAIL                  = "fail"
	IP_LIMIT              = "IP限制"
	Internal_Error        = "内部错误"
	Sign_Check_Error      = "签名验证失败"
	Unrealized_Type_Error = "未实现的接口"
	Param_Error           = "参数错误"
)

// 通用返回
type CommonRet struct {
	Ret  int32       `json:"ret"`  // 返回状态标识，成功时为0，失败时为错误码
	Info interface{} `json:"info"` // 错误信息，如果为一般错误信息则为string(没有info.rets)，如果操作结果为部分成功则为rets数组
}

// 通用返回的数组对象
type RetItems struct {
	SvrId  int32  `json:"svr_id"`  // 返回失败的服务器ID
	UserId int32  `json:"user_id"` // 返回失败的玩家ID
	Ret    int32  `json:"ret"`     // 返回错误码
	Info   string `json:"info"`    // 返回信息描述
}

// 通用基础的返回数组对象
type RetBaseItems struct {
	Ret  int32  `json:"ret"`  // 返回状态标识，成功时为0，失败时为错误码
	Info string `json:"info"` // 错误信息
}

// 通用user结构
type CommonUser struct {
	SvrId             int32         `json:"svr_id"`             // 查询的目标服务器ID
	BornsvrId         int32         `json:"bornsvr_id"`         // 出生服ID
	UserId            int           `json:"user_id"`            // 游戏内分配的角色唯一ID
	UserName          string        `json:"user_name"`          // 游戏内角色名称
	OpenId            string        `json:"open_id"`            // sdk厂商提供的账号唯一ID
	Plat              string        `json:"plat"`               // 平台信息，比如ios，国内安卓官方，360……
	UserLevel         int32         `json:"user_level"`         // 角色等级
	Vip               int32         `json:"vip"`                // VIP等级
	RechargeSum       int32         `json:"recharge_sum"`       // 充值金额
	Currency          int           `json:"currency"`           // 玩家当前的一级货币数量(从商店购买得到的)
	Coins             []*CommonCoin `json:"coins"`              // 其他游戏内次级货币coin（非商店购买）数组, coin定义如下
	Items             []*CommonItem `json:"items"`              // 玩家道具item数组, item定义如下
	MonthcardLeftdays string        `json:"monthcard_leftdays"` // 月卡剩余天数，如果多张月卡用“，”分隔
	LastLoginTime     int64         `json:"last_login_time"`    // 上次登录时间，unix时间戳
	LastLoginIp       string        `json:"last_login_ip"`      // 上次登录IP
	CreateTime        int64         `json:"create_time"`        // 账号创建时间，unix时间戳
	UnlockTime        int64         `json:"unlock_time"`        // 封号到期时间，unix时间戳，未被封号该值为0
	UnsilenceTime     int64         `json:"unsilence_time"`     // 禁言到期时间，unix时间戳，未被禁言该值为0
	IsShield          bool          `json:"is_shield"`          // 玩家角色屏蔽状态, true 已屏蔽 false 未屏蔽
}

// 通用货币结构
type CommonCoin struct {
	CoinName  string `json:"coin_name"`  // 游戏自己定义的货币名称
	CoinValue int    `json:"coin_value"` // 货币数量
}

// 通用道具结构
type CommonItem struct {
	ItemId    string `json:"item_id"`    // 道具ID
	ItemCount int32  `json:"item_count"` // 道具数量
}

// 通用装备结构
type CommonEquip struct {
	EquipName   string `json:"equip_name"`   // 装备名称
	EquipCount  int    `json:"equip_count"`  // 装备数量
	EquipCustom string `json:"equip_custom"` // 自定义属性
}

// 通用英雄结构
type CommonHero struct {
	Name    string `json:"name"`    // 英雄名称
	Level   int    `json:"level"`   // 英雄等级
	Quality string `json:"quality"` // 英雄装备(品质)
}

// 通用问卷结构
type CommonQuestion struct {
	QuestionType string `json:"type"`     //'0' 常规 '1' 问卷
	Sid          string `json:"sid"`      //单语言sid
	SourceId     string `json:"sourceid"` //多语言sourceid
}

// 通用礼包结构
type CommonGiftPackage struct {
	SvrId     int `json:"svr_id"`     // 服务器id
	PlayerId  int `json:"player_id"`  // 玩家id
	PackageId int `json:"package_id"` // 礼包id
	ChargeId  int `json:"charge_id"`  // 充值id，可选字段，目前仅hgame有
}

// 通用自定义资源
type CommonMultiResource struct {
	Uid      int    `json:"uid"`      // 玩家id
	Items    string `json:"items"`    // 道具字段（若只发礼包码，无此字段，{ l_coin: 1,coin : 2,xuezuan: 3}, map[string]int解析
	GiftCode string `json:"giftCode"` // 礼包码（若只发道具，无此字段）
}

package lilith

//// 公共头数据
//type SystemFieldInfo struct {
//	LogType     string `json:"log_type"`     // 每种类型日志输出不同字符串，具体见各类型数据定义
//	Version     int32  `json:"version"`      // 数据版本号，目前版本号为1
//	EventTime   string `json:"event_time"`   // 事件发生时间（见附注2）
//	GameId      string `json:"game_id"`      // 一款游戏对应的唯一ID，由莉莉丝提供
//	Pkg         string `json:"pkg"`          // 在某个平台上架的唯一游戏包名（见附注1）
//	Channel     string `json:"channel"`      // 分发渠道ID,IOS客户端务必请填self-lilith-0.7, Android客户端务必请使用获取渠道号章节所述接口,PC客户端务必请填self-lilith-1
//	Idfa        string `json:"idfa"`         // Ios设备广告标示符，IOS客户端必填
//	AndroidId   string `json:"android_id"`   // 安卓设备id，安卓客户端必填
//	GoogleAid   string `json:"google_aid"`   // Google广告平台账号，安装了google play的设备可取到
//	Ip          string `json:"ip"`           // 玩家设备的IP地址
//	Os          string `json:"os"`           // 玩家设备操作系统（见附注）
//	OsVersion   string `json:"os_version"`   // 玩家设备系统版本号（见附注）
//	AppVersion  string `json:"app_version"`  // 应用版本号（见附注）
//	DeviceModel string `json:"device_model"` // 设备产品型号（见附注）
//	OpenId      string `json:"open_id"`      // 平台id
//	UserId      string `json:"user_id"`      // 游戏内的账号唯一ID，如果直接使用了Open id则可不填
//	ServerId    string `json:"server_id"`    // 服务器ID，命名规则见附注
//}
//
//// 首次登录游戏(usercreate)
//type UserCreate struct {
//	*SystemFieldInfo
//}
//
//// 每次账号登录(userlogin)
//type UserLogin struct {
//	*SystemFieldInfo
//}
//
//// 账号登出游戏(userlogout)
//type UserLogout struct {
//	*SystemFieldInfo
//	GameTime string `json:"game_time"` // 本次游戏时长，单位秒
//}
//
//// 创建游戏角色(rolecreate)
//type RoleCreate struct {
//	*SystemFieldInfo
//	RoleId string `json:"role_id"` // 角色唯一id
//}
//
//// 游戏角色登录(rolelogin)
//type RoleLogin struct {
//	*SystemFieldInfo
//	RoleId   string  `json:"role_id"`   // 角色唯一id
//	Level    int32   `json:"level"`     // 等级
//	VipLevel int32   `json:"vip_level"` // vip等级
//	Recharge float32 `json:"recharge"`  // 累计充值金额
//	Language string  `json:"language"`  // 游戏内语言，中文简体=zh_CN,中文繁体=zh_TW
//}
//
//// 游戏角色退出(rolelogout)
//type RoleLogout struct {
//	*SystemFieldInfo
//	RoleId   string  `json:"role_id"`   // 角色唯一id
//	GameTime string  `json:"game_time"` // 本次游戏时长，单位秒
//	Level    int32   `json:"level"`     // 等级
//	VipLevel int32   `json:"vip_level"` // vip等级
//	Recharge float32 `json:"recharge"`  // 累计充值金额
//}
//
//// 在线人数，每分钟上报一次(online)
//type Online struct {
//	LogType     string `json:"log_type"`     // 记录类型 该条为 “online”
//	Version     string `json:"version"`      // 数据版本号
//	EventTime   string `json:"event_time"`   // 事件时间戳
//	GameId      string `json:"game_id"`      // 游戏唯一id 莉莉丝提供
//	OnlineCount int64  `json:"online_count"` // 当前在线人数
//	ServerTag   string `json:"server_tag"`   // 服务器标识符，可以通过描述找到服务器
//}
//
//// 角色充值行为(purchase)
//type Purchase struct {
//	*SystemFieldInfo
//	RoleId   string  `json:"role_id"`   // 角色唯一id
//	ItemId   string  `json:"item_id"`   // 商品id 优先id 其次名称
//	Level    int32   `json:"level"`     // 角色等级
//	VipLevel int32   `json:"vip_level"` // vip等级
//	IsTest   int32   `json:"is_test"`   // 测试订单标记 测试订单为1 ，正常订单为0 ，订阅订单为2
//	OrderId  string  `json:"order_id"`  // 充值订单号
//	Recharge float32 `json:"recharge"`  // 累计充值金额，美金单位
//	Currency string  `json:"currency"`  // 充值货币单位
//	Price    float32 `json:"price"`     // 充值货币金额
//	Iap      string  `json:"iap"`       // 商店后台购买项ID
//	PayType  string  `json:"pay_type"`  // 支付方式 SDK支付接口的"pay_type" unisdk支付接口的"plat"
//}
//
//// 角色退款
//type Refund struct {
//	*SystemFieldInfo
//	RoleId   string  `json:"role_id"`   // 角色唯一id
//	OrderId  string  `json:"order_id"`  // 充值订单号
//	Level    int32   `json:"level"`     // 角色等级
//	VipLevel int32   `json:"vip_level"` // vip等级
//	Recharge float32 `json:"recharge"`  // 累计充值金额，美金单位
//}
//
//// 角色一级代币变化
//type MoneyFlow struct {
//	*SystemFieldInfo
//	RoleId      string  `json:"role_id"`      // 角色唯一id
//	MoneyBefore int64   `json:"money_before"` // 变化前货币数值
//	MoneyAfter  int64   `json:"money_after"`  // 变化后货币数值
//	Flow        string  `json:"flow"`         // 货币流向 获得为"in" 消耗为"out" 退款，坏账导致的钻石扣除 需上报一条货币流向为"in" 的消息，变化量为负
//	Action      string  `json:"action"`       // 触发货币变动的行为，我们需要编写一份文档表示对应的行为
//	Level       int32   `json:"level"`        // 角色等级
//	VipLevel    int32   `json:"vip_level"`    // vip等级
//	MoneyType   string  `json:"money_type"`   // 非必须 货币类型，如果存在多种获得的货币类型则需要
//	Item        string  `json:"item"`         // 非必须 如果行为的结果是获得道具资源，则需要填写资源id
//	Recharge    float32 `json:"recharge"`     // 累计充值金额，美金单位
//}
//
//// 角色资源变化
//type ResourceFlow struct {
//	*SystemFieldInfo
//	RoleId         string  `json:"role_id"`         // 角色唯一id
//	ResourceId     string  `json:"resource_id"`     // 资源id
//	ResourceBefore int64   `json:"resource_before"` // 变化前资源数值
//	ResourceAfter  int64   `json:"resource_after"`  // 变化后资源数值
//	Flow           string  `json:"flow"`            // 资源流向 获得为"in" 消耗为"out"
//	Action         string  `json:"action"`          // 触发资源变动的行为，我们需要编写一份文档表示对应的行为
//	Level          int32   `json:"level"`           // 角色等级
//	VipLevel       int32   `json:"vip_level"`       // vip等级
//	Recharge       float32 `json:"recharge"`        // 累计充值金额，美金单位
//}
//
//// 角色物品变化
//type ItemFlow struct {
//	*SystemFieldInfo
//	RoleId     string  `json:"role_id"`     // 角色唯一id
//	ItemFlow   string  `json:"item_flow"`   // 物品流向 获得为"in" 消耗为"out"
//	ItemId     string  `json:"item_id"`     // 物品id
//	ItemCount  int32   `json:"item_count"`  // 变化的物品数量
//	Level      int32   `json:"level"`       // 角色等级
//	VipLevel   int32   `json:"vip_level"`   // vip等级
//	Action     string  `json:"action"`      // 触发物品变动的行为，我们需要编写一份文档表示对应的行为
//	ItemBefore int64   `json:"item_before"` // 非必须 变化前资源数值
//	ItemAfter  int64   `json:"item_after"`  // 非必须 变化后资源数值
//	Recharge   float32 `json:"recharge"`    // 累计充值金额，美金单位
//}
//
//// 角色升级
//type LevelUp struct {
//	*SystemFieldInfo
//	RoleId   string  `json:"role_id"`   // 角色唯一id
//	Level    int32   `json:"level"`     // 角色等级
//	VipLevel int32   `json:"vip_level"` // vip等级
//	Recharge float32 `json:"recharge"`  // 累计充值金额，美金单位
//}
//
//// 玩家点击游戏内广告变现输出
//type VideoAds struct {
//	*SystemFieldInfo
//	RoleId    string `json:"role_id"`   // 角色唯一id
//	Level     int32  `json:"level"`     // 角色等级
//	VipLevel  int32  `json:"vip_level"` // vip等级
//	EventType string `json:"event_type"`
//	TargetId  string `json:"target_id"` // 区分玩家点击视频广告位置
//}
//
//// 网络心跳延迟
//type NetMonitor struct {
//	*SystemFieldInfo
//	HbCnt         int32  `json:"hb_cnt"`         // 心跳次数
//	HbDelay       int32  `json:"hb_delay"`       // 平均心跳延迟 单位ms
//	TimeoutCnt    int32  `json:"timeout_cnt"`    // 心跳超时次数
//	TimeoutLength int32  `json:"timeout_length"` // 平均心跳超时时长
//	Line          string `json:"line"`           // 非必须 线路标识
//}
//
//// 角色封禁
//type BanRole struct {
//	LogType    string `json:"log_type"`   // 记录类型 该条为 "banrole"
//	Version    string `json:"version"`    // 数据版本号
//	EventTime  string `json:"event_time"` // 事件发生时间
//	GameId     string `json:"game_id"`    // 游戏唯一id 莉莉丝提供
//	OpenId     string `json:"open_id"`
//	UserId     string `json:"user_id"`     // 非必须 账号唯一id
//	ServerId   string `json:"server_id"`   // 服务器id 填0（世界服填0）
//	RoleId     string `json:"role_id"`     // 角色唯一id
//	UnlockTime string `json:"unlock_time"` // 解封时间
//	Reason     string `json:"reason"`      // 处理原因
//	BanSource  string `json:"ban_source"`  // 封禁来源 例如: GMT 手动
//	BanReason  string `json:"ban_reason"`  // 封禁原因 （内部展示 涩涩，涉政）
//}
//
//// 角色解封
//type UnBanRole struct {
//	LogType   string `json:"log_type"`   // 记录类型 该条为 "unbanrole"
//	Version   string `json:"version"`    // 数据版本号
//	EventTime string `json:"event_time"` // 事件发生时间
//	GameId    string `json:"game_id"`    // 游戏唯一id 莉莉丝提供
//	OpenId    string `json:"open_id"`
//	UserId    string `json:"user_id"`   // 非必须 账号唯一id
//	ServerId  string `json:"server_id"` // 服务器id 填0（世界服填0）
//	RoleId    string `json:"role_id"`   // 角色唯一id
//	Reason    string `json:"reason"`    // 处理原因
//}
//
//// 玩家社区举报
//type Complain struct {
//	LogType          string `json:"log_type"`   // 记录类型 该条为 "complain"
//	Version          string `json:"version"`    // 数据版本号
//	EventTime        string `json:"event_time"` // 事件发生时间
//	GameId           string `json:"game_id"`    // 游戏唯一id 莉莉丝提供
//	ReporterOpenId   string `json:"reporter_open_id"`
//	ReporterRoleId   string `json:"reporter_role_id"`   // 举报者 角色唯一id
//	ReporterServerId string `json:"reporter_server_id"` // 举报者 服务器id 填0（世界服填0）
//	OpenId           string `json:"open_id"`
//	RoleId           string `json:"role_id"`        // 被举报者 角色唯一id
//	ServerId         string `json:"server_id"`      // 被举报者 服务器id 填0（世界服填0）
//	NickName         string `json:"nick_name"`      // 被举报者 昵称
//	Level            string `json:"level"`          // 被举报者 等级
//	VipLevel         string `json:"vip_level"`      // vip等级
//	Icon             string `json:"icon"`           // 被举报者 头像
//	Signature        string `json:"signature"`      // 被举报者 签名
//	TotalRecharge    string `json:"total_recharge"` // 被举报者 累计充值金额
//	Reason           string `json:"reason"`         // 被举报原因
//	Param_1          string `json:"param_1"`        // 自定义字段
//}
//
//// 角色信息日志
//type RoleInfo struct {
//	*SystemFieldInfo
//	RoleId      string `json:"role_id"`      // 被举报者 角色唯一id
//	InfoDetails string `json:"info_details"` // 玩家状态信息列表 json格式
//}

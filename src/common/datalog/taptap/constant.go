package taptap

// 通用日志类型
const (
	LogType_RoleCreate   = "#rolecreate"   // 创建游戏角色输出
	LogType_RoleLogin    = "#rolelogin"    // 角色登录输出
	LogType_RoleLogout   = "#rolelogout"   // 角色登出输出
	LogType_Online       = "#online"       // 服务器在线人数 /min
	LogType_Purchase     = "#purchase"     // 角色充值行为发生时输出
	LogType_Refund       = "#refund"       // 角色退款发生时输出
	LogType_MoneyFlow    = "#moneyflow"    // 角色的一级代币变化
	LogType_ResourceFlow = "#resourceflow" // 角色资源变化
	LogType_ItemFlow     = "#itemflow"     // 角色物品变化
	LogType_LevelUp      = "#levelup"      // 角色升级
	LogType_BanRole      = "#banrole"      // 角色封禁
	LogType_UnBanRole    = "#unbanrole"    // 角色解封
)

// 自定义日志类型
const (
	LogType_MailReceive         = "#mailreceive"
	LogType_MailDelete          = "#maildelete"
	LogType_GloToPersonalMail   = "#globaltopersonalmail" // 玩家领取全服邮件到私人邮件
	LogType_CardExtract         = "#cardextract"
	LogType_CompleteQuestObject = "#completequestobject"
	LogType_CardEreakThrough    = "#cardbreakthrough"   // 卡牌突破
	LogType_CardSkillUpgrade    = "#cardskillupgrade"   // 技能升级
	LogType_CardCompound        = "#cardcompound"       // 卡牌觉醒
	LogType_CardCharacterBreak  = "#cardcharacterbreak" // 性格突破
	LogType_CardEatFood         = "#cardeatfood"        // 体力恢复
	LogType_CardLevelUp         = "#cardlevelup"        // 卡牌升级
	LogType_DaySign             = "#daysign"            // 签到
	LogType_TutorialDddRecord   = "#tutorialaddrecord"  // 记录引导点
	LogEquipLevelUp             = "#equiplevelup"       // 装备升级
	LogEquipWear                = "#equipwear"          // 装备穿戴
	LogEquipunWear              = "#equipunwear"        // 装备卸下
	LogEquipDelete              = "#equipdelete"        // 装备删除
	LogEquipCreate              = "#equipcreate"        // 装备获得
	LogType_SkinDress           = "#skindress"
	LogType_SkinCreate          = "#skincreate"
	LogType_TroopOperate        = "#troopoperate"
	LogType_FoodOperate         = "#foodoperate"
	LogType_UseItem             = "#useitem"
	LogType_DestroyExpireItem   = "#destroyexpireitem"
	LogType_ItemBuy             = "#itembuy"
	LogType_CurrencyExchange    = "#currencyexchange"
	LogType_CurrencyBuy         = "#currencybuy"
	LogType_ChangeDutyCard      = "#changedutycard"
	LogType_ReceiveDailyReward  = "#receivedailyreward"
	LogType_ReceiveActiveReward = "#receiveactivereward"

	// 营地埋点类型定义
	LogType_MakeFuncBuilding     = "#makefuncbuilding"     //修建建筑
	LogType_BuildingLevelUp      = "#buildinglevelup"      //建筑升级
	LogType_CampFurnaceoPerate   = "#campfurnaceoperate"   //熔炉熔炼
	LogType_CancelFurnaceQueue   = "#cancelfurnacequeue"   //熔炉队列取消
	LogType_GetqueueReward       = "#getqueuereward"       //领取队列奖励
	LogType_CamptreeReward       = "#camptreereward"       //光合树收获
	LogType_CampmaterialConver   = "#campmaterialconver"   //材料研究转换
	LogType_CampequipFoundry     = "#campequipfoundry"     //装备打造
	LogType_CampMakeFood         = "#campmakefood"         //食物制作
	LogType_CamproleChange       = "#camprolechange"       //营地角色上阵
	LogType_CampBuildingFoundry  = "#campbuildingfoundry"  //家具打造
	LogType_CampTraderExchange   = "#camptraderexchange"   //商人兑换奖励
	LogType_CampTraderRefresh    = "#camptraderrefresh"    //商人清单刷新
	LogType_CampBuildingUpcard   = "#campbuildingupcard"   //建筑卡牌驻守
	LogType_CampBuildingDownCard = "#campbuildingdowncard" //建筑卡牌下阵
	LogType_CampSaveLayout       = "#campsavelayout"       //布局修改
	LogType_CampSwitchLayout     = "#campswitchlayout"     //切换布局方案

	// 大地图
	LogType_Level_enter            = "#enterlevel"          //进入大地图/副本
	LogType_Level_exit             = "#exitlevel"           //退出大地图/副本
	LogType_Level_start_battle     = "#levelstartbattle"    //大地图开始战斗
	LogType_Level_end_battle       = "#levelendbattle"      //大地图结束战斗
	LogType_Level_map_event        = "#levelmapevent"       //大地图事件
	LogType_Level_choose_niwa_path = "#levelchooseniwapath" //选择路径
	LogType_Level_unlockpoint      = "#levelunlockpoint"    //解锁传送点
	LogType_Level_backtobc         = "#backtobc"            //返回大本营

	// 日替
	LogType_Campaign_list         = "#campaignlist"        //获取日替副本列表
	LogType_Campaign_enter        = "#campaignenter"       //进入日替副本
	LogType_Campaign_start_battle = "#campaignstartbattle" //日替战斗开始
	LogType_Campaign_end_battle   = "#campaignendbattle"   //日替战斗结束

	// 商店
	LogType_shop_list = "#shoplist" //商店列表
	LogType_shop_info = "#shopinfo" //获取商店信息
	LogType_shop_buy  = "#shopbuy"  //商店购买物品

	// 玩家
	LogType_StaminaChange = "#staminachange" //玩家体力变化

	// 全局
	LogType_ServiceStart  = "#serverstart"     // 服务启动
	LogType_ServiceStop   = "#serverStop"      // 服务退出
	LogType_ServerHour    = "#serverhour"      // 服务定时器
	LogType_Confevent     = "#serverconfevent" // 服务配置事件
	LogType_ServeReload   = "#servereload"     // 服务热更新
	LogType_GmCmd         = "#gmcmd"           // GM指令
	LogType_UserActor     = "#useractor"       // 用户容器上线下
	LogType_DbFail        = "#dbfail"          // DB读写失效
	LogType_LoginDelay    = "#logindelay"      // login网络延迟
	LogType_GateDelay     = "#gatedelay"       // gate网络延迟
	LogType_GetVersion    = "#getversion"      // 版本获取
	LogType_AccountLogin  = "#accountlogin"    // 账号loginServer登录
	LogType_TcpEnter      = "#tcpenter"        // 长链进入
	LogType_Reconnect     = "#reconnect"       // 断线重连
	LogType_RepeatedLogin = "#repeatedlogin"   // 重复登录
	LogType_GlobalMailAdd = "#globalmailadd"   // 新增全局邮件
	LogType_GlobalMailDel = "#globalmaildel"   // 删除全局邮件
)

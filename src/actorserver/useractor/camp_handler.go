package useractor

import (
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"strconv"
	"time"

	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	_ "google.golang.org/genproto/googleapis/cloud/dataproc/v1"
	"google.golang.org/protobuf/proto"
)

const (
	LEVEL_UPGRADE_CHANGE_1 = 1 // 队列数量
	LEVEL_UPGRADE_CHANGE_2 = 2 // 上阵角色数量
	//LEVEL_UPGRADE_CHANGE_3  = 3     // 区域解锁
	LEVEL_UPGRADE_CHANGE_4  = 4     // 食材品质
	DEFAULT_CAMP_ID         = 9001  // 默认营地ID
	LAYOUT_MAXIMUM          = 4     // 每个营地最大布局数量
	BUILDING_FURNACE_ID     = 90074 // 熔炉配置id
	BUILDING_FOOD_SUPPLY_ID = 90075 // 食物加工厂配置id
	TRADER_TYPE_1           = 1     // 商人类型 固定池
	TRADER_TYPE_2           = 2     // 商人类型 随机池
	TRADER_STATUS_1         = 1     // 商人清单状态 未兑换
	TRADER_STATUS_2         = 2     // 商人清单状态 已兑换
	DEFAULT_FOOD_ID         = 5001  // 默认食物<焦炭>
	DEFAULT_FOOD_COST_NUM   = 4     // 默认食材最大数量
)

type CampHandler struct {
	*UABaseHandler
	Buildings map[cmd.PlayerCampBuildingType]IBuilding
	LifeSkill map[int32]map[int32]*excel.LifeSkillCfg
}

func NewCampHandler(actor *UserActor) *CampHandler {
	h := &CampHandler{
		UABaseHandler: NewUABaseHandler(actor, "CampHandler"),
		Buildings:     nil,
	}
	h.ChildHandler = h
	h.initBuildFactory()

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampInfoReq), h.CampInfoReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampLayoutReq), h.CampLayoutReq) //布局请求

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampSwitchReq), h.CampSwitchReq)                     // 切换布局方案 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampLayoutSaveReq), h.CampLayoutSaveReq)             // 布局修改 工作区家装摆设 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampLayoutModifyNameReq), h.CampLayoutModifyNameReq) // 布局名称修改
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampRoleChangeReq), h.CampRoleChangeReq)             // 营地角色上阵 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampGetHomeCoinReq), h.CampGetHomeCoinReq)           // 领取家装币
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampBuyEditBlockTimesReq), h.CampBuyEditBlockTimes)  // 购买编辑地块次数
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampGetThumbReq), h.CampGetThumbReq)                 // 获取缩略图

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampMakeFunctionBuildingReq), h.CampMakeFunctionBuildingReq) // 修建建筑 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampFunctionBuildingLvUpReq), h.CampFunctionBuildingLvUpReq) // 功能建筑升级 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampBuildingFoundryReq), h.CampBuildingFoundryReq)           // 家具打造 1

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampMakeFoodReq), h.CampMakeFoodReq)                                   // 自主烹饪 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampFuncBuildingOpCancelReq), h.CampFuncBuildingOpCancelReq)           // 食物或者熔炉取消建造
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampFuncBuildingGetRewardReq), h.CampFuncBuildingGetRewardReq)         // 领取队列奖励 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampLightingComposeTreeRewardReq), h.CampLightingComposeTreeRewardReq) // 光合树收获

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampTraderExchangeReq), h.PlayerCampTraderExchangeReq) // 商人兑换奖励 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampTraderRefreshReq), h.PlayerCampTraderRefreshReq)   // 商人清单刷新 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampFuncUpCardReq), h.PlayerCampFuncUpCardReq)         // 建筑卡牌驻守 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampFuncDownCardReq), h.PlayerCampFuncDownCardReq)     // 建筑卡牌下阵 1
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampBuildFunOpReq), h.CampBuildFunReq)                 // 材料转化，熔炼、装备制作

	return h
}

func (h *CampHandler) Init() error {
	// 初始化
	h.actor.Data.Camp = &cmd.PPlayerCampBlob{
		Createtime:         time.Now().Unix(),
		CurrentCampId:      0,
		CurrentLayoutId:    0,
		DecorationBuilding: make(map[int64]*cmd.PPlayerCampDecorationBuilding),
		Camp:               make(map[int32]*cmd.PPlayerCampServerCamp),
		BuildingUnlockList: make(map[int32]int32),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}
	h.Debug("init camp data success. player: %s", h.actor.ID())
	return nil
}

func (h *CampHandler) EnterGame() error {
	// 尝试修正食物制造所数据
	data := h.actor.GetCampData()
	for _, camp := range data.Camp {
		// 是否解锁
		buildingId := int64(0)
		for _, layout := range camp.Layout {
			for _, building := range layout.Building {
				if building.Building.ItemId == BUILDING_FOOD_SUPPLY_ID {
					buildingId = building.Building.BuildingId
					break
				}
			}
		}
		// 解锁了才处理
		if buildingId > 0 && camp.Foods == nil {
			// 取初始化配置
			buildLevelId := BUILDING_FOOD_SUPPLY_ID*100 + 1
			buildLevelConfig := excel.GetBuildingLevelMgr().GetById(int32(buildLevelId))
			if buildLevelConfig == nil {
				return fmt.Errorf("config not found")
			}

			camp.Foods = &cmd.PPlayerCampMakeFood{
				BuildingId:  buildingId,
				Level:       1,
				UnlockFoods: buildLevelConfig.UpgradeFood,
				IsNew:       buildLevelConfig.UpgradeFood,
			}
			camp.Foods.UnlockFoods = append(camp.Foods.UnlockFoods, DEFAULT_FOOD_ID)
			h.Infof("fix camp food data %d", buildingId)
		}
	}
	// 预处理配置表
	h.ProcessExcel()
	return nil
}

func (h *CampHandler) ProcessExcel() {
	if h.LifeSkill == nil {
		h.LifeSkill = make(map[int32]map[int32]*excel.LifeSkillCfg)
	}
	excel.GetLifeSkillMgr().Foreach(func(cfg *excel.LifeSkillCfg) bool {
		var item map[int32]*excel.LifeSkillCfg
		var ok bool
		scope, _ := strconv.Atoi(cfg.Scope)
		if item, ok = h.LifeSkill[int32(scope)]; !ok {
			item = make(map[int32]*excel.LifeSkillCfg, 0)
		}
		item[cfg.Id] = cfg
		h.LifeSkill[int32(scope)] = item
		return true
	}, true)
}

func (h *CampHandler) DailyRefresh() error {
	return nil
}

func (h *CampHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PPlayerCampBlob); ok {
		h.actor.Data.Camp = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *CampHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCamp(h.actor.ID()), h.actor.Data.Camp
}

func (h *CampHandler) init(campId int32) bool {
	camp := h.actor.GetCampData()
	camp.CurrentCampId = campId
	camp.CurrentLayoutId = 1
	// 目前上来就初始化4个营地
	layout := make(map[int32]*cmd.PPlayerCampServerLayout, LAYOUT_MAXIMUM)
	for i := 1; i <= LAYOUT_MAXIMUM; i++ {
		if i == 1 {
			// 默认家具初始化
			layout[int32(1)] = h.DefaultLayout()
			continue
		}
		layout[int32(i)] = h.NewPlayerCampServerLayout(int32(i))
	}

	camp.Camp[campId] = &cmd.PPlayerCampServerCamp{CampId: campId,
		Layout:        layout,
		RoleList:      make([]int32, 0),
		BlockMaxTimes: h.initBlockMaxTimes(),
	}
	h.SaveDB(true)
	return true
}
func (h *CampHandler) initBlockMaxTimes() int32 {
	times, _ := h.GetMaxEditTimes()
	if len(times) == 2 {
		return times[0] //初始最大次数
	}
	return 0
}
func (h *CampHandler) NewPlayerCampServerLayout(layoutId int32) *cmd.PPlayerCampServerLayout {
	themeCfg := excel.GetBuildMainMgr().GetById(90189)
	if themeCfg == nil {
		h.Warnf("build main config not found, id: %d", 90189)
	}
	layout := &cmd.PPlayerCampServerLayout{
		LayoutId:        layoutId,
		LayoutName:      "",
		ThemeId:         90189,
		AtmosphereValue: themeCfg.Ambience,
		Building:        make(map[int64]*cmd.PPlayerCampCommonBuilding),
	}

	return layout
}

func (h *CampHandler) getCamp(campId int32) *cmd.PPlayerCampServerCamp {
	campMap := h.actor.GetCampData().Camp
	if camp, ok := campMap[campId]; ok {
		return camp
	}
	return nil
}

func (h *CampHandler) getLayout(campId, layoutId int32) *cmd.PPlayerCampServerLayout {
	camp := h.getCamp(campId)
	if camp == nil || camp.Layout == nil {
		return nil
	}
	if layout, ok := camp.Layout[layoutId]; ok {
		return layout
	}
	return nil
}

func (h *CampHandler) getCurCamp() *cmd.PPlayerCampServerCamp {
	return h.getCamp(h.getCurrentCampId())
}

func (h *CampHandler) getCurLayout() *cmd.PPlayerCampServerLayout {
	return h.getLayout(h.getCurrentCampId(), h.getCurrentLayoutId())
}

func (h *CampHandler) getCurrentCampId() int32 {
	return h.actor.Data.Camp.CurrentCampId
}

func (h *CampHandler) getCurrentLayoutId() int32 {
	return h.actor.Data.Camp.CurrentLayoutId
}

func (h *CampHandler) getWorkQueueBuilding(buildingId int64) *cmd.PPlayerCampBuildingWorkQueue {
	camp := h.getCurCamp()
	if camp == nil {
		return nil
	}

	if workQueue, ok := camp.WorkQueue[buildingId]; ok {
		h.Debugf("camp handler getWorkQueueBuilding queues:%+v, work:%+v, buildId:%d", camp.WorkQueue, workQueue, buildingId)
		return workQueue
	}
	return nil
}

// 获取指定好友的露营地数据
func (h *CampHandler) getFriendCampInfo(uaid string) *cmd.PFriendCampInfo {
	res := &cmd.PFriendCampInfo{}
	if err, _ := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
		return res
	}

	data := &cmd.PPlayerCampBlob{}
	_, err := h.actor.GetCache(service.MongoDbType_MongoGame, db.KeyUserCamp(uaid), data)
	if err != nil {
		return nil
	}
	if data == nil || data.Camp == nil {
		return nil
	}
	//获取当前营地
	curCamp := data.Camp[data.CurrentCampId]
	if curCamp == nil || curCamp.Layout == nil {
		return nil
	}
	curLayout, ok := curCamp.Layout[data.CurrentLayoutId]
	if !ok {
		return nil
	}
	dataCard := &cmd.PCardData{}
	_, err = h.actor.GetCache(service.MongoDbType_MongoGame, db.KeyUserCard(uaid), dataCard)
	if err != nil {
		return nil
	}

	for _, id := range curCamp.RoleList {
		cardInfo := dataCard.Card[uint32(id)]
		if cardInfo == nil {
			h.Debug("get friend camp info cardInfo err:", id)
			continue
		}
		res.RoleList = append(res.RoleList, int32(cardInfo.SkinId))
	}
	res.Layout = ServerLayout2ClientLayout(data.CurrentLayoutId, curLayout, true)
	return res
}

func (h *CampHandler) NewPPlayerCampDecorationBuilding(itemId, level, campId int32) *cmd.PPlayerCampDecorationBuilding {

	decorationBuilding := &cmd.PPlayerCampDecorationBuilding{
		BuildingId:    int64(h.actor.Srv.GenGUID(guid.GUID_BUILDING)),
		ItemId:        itemId,
		BuildingLevel: level,
		CampId:        campId,
		CardId:        0,
		FuncType:      cmd.PlayerCampBuildingFunType_Type_building,
	}
	cfg := excel.GetBuildMainMgr().GetById(itemId)
	if cfg.Category == int32(cmd.CampBuildItemType_Build_Item_Type_Six) {
		decorationBuilding.FuncType = cmd.PlayerCampBuildingFunType_Type_theme
	}
	return decorationBuilding
}

func (h *CampHandler) NewPPlayerCampCommonBuilding(x, y, parentId, parentGridId, edge int32, flip bool, building *cmd.PPlayerCampDecorationBuilding) *cmd.PPlayerCampCommonBuilding {
	return &cmd.PPlayerCampCommonBuilding{
		X:            x,
		Y:            y,
		ParentId:     parentId,
		ParentGridId: parentGridId,
		Flip:         flip,
		Edge:         edge,
		Building:     building,
	}
}

// IsBuildingUpCard 指定建筑中是否是指定卡牌驻守
func (h *CampHandler) IsBuildingUpCard(buildingId int32, cardId int32) bool {
	for _, v := range h.actor.GetCampData().DecorationBuilding {
		if v.ItemId == buildingId && v.CardId == cardId {
			return true
		}
	}
	return false
}

// GetBuildingCountByLevel 获取指定等级的营地建筑数量
func (h *CampHandler) GetBuildingCountByLevel(level int32) int32 {
	var count int32
	for _, v := range h.actor.GetCampData().DecorationBuilding {
		if v.BuildingLevel >= level {
			count++
		}
	}
	return count
}

// GetBuildingLevel 获取指定建筑的等级
func (h *CampHandler) GetBuildingLevel(buildingId int32) int32 {
	for _, v := range h.actor.GetCampData().DecorationBuilding {
		if v.ItemId == buildingId {
			return v.BuildingLevel
		}
	}
	return 0
}

// BuildingExist 判断是定建筑是否存在
func (h *CampHandler) BuildingExist(buildingId int32) bool {
	for _, v := range h.actor.GetCampData().DecorationBuilding {
		if v.ItemId == buildingId {
			return true
		}
	}
	return false
}

// BuildingItem 功能建筑ID、等级
type BuildingItem struct {
	itemId int32
	level  int32
}

// EquipItemIdMap 用于随机
type EquipItemIdMap struct {
	cfg map[interface{}]int32
}

// OutputParams 传出参数
type OutputParams struct {
	isFunctionBuilding bool
	buildingId         int64
	incrLevel          int32
	messageId          int32
	buildType          int32
	layout             *cmd.PPlayerCampServerLayout
	camp               *cmd.PPlayerCampServerCamp
	buildLevelConfig   *excel.BuildingLevelCfg
	building           *cmd.PPlayerCampDecorationBuilding
}

func NewOutputParams(isFunctionBuilding bool, buildingId int64, incrLevel, messageId int32) *OutputParams {
	return &OutputParams{
		isFunctionBuilding: isFunctionBuilding,
		buildingId:         buildingId,
		incrLevel:          incrLevel,
		messageId:          messageId}
}

func checkBuildType(messageId, buildType int32) (error, int32) {
	switch messageId {
	case int32(cmd.Protocols_PC2LS_PlayerCampFunctionBuildingLvUpReq),
		int32(cmd.Protocols_PC2LS_PlayerCampFuncUpCardReq),
		int32(cmd.Protocols_PC2LS_PlayerCampFuncDownCardReq),
		int32(cmd.Protocols_PC2LS_PlayerCampFuncBuildingGetRewardReq),
		int32(cmd.Protocols_PC2LS_PlayerCampFuncBuildingOpCancelReq):
		if buildType <= int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_None) ||
			buildType >= int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_Max) {
			return fmt.Errorf("buildType error, must be match function building"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
		}

	case int32(cmd.Protocols_PC2LS_PlayerCampLightingComposeTreeRewardReq):
		if buildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_LightingComposeTree) {
			return fmt.Errorf("buildType error, must be lighting compose tree"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
		}
	case int32(cmd.Protocols_PC2LS_PlayerCampTraderExchangeReq), int32(cmd.Protocols_PC2LS_PlayerCampTraderRefreshReq):
		if buildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_Trader) {
			return fmt.Errorf("buildType error, must be trader"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
		}
	case int32(cmd.Protocols_PC2LS_PlayerCampMakeFoodReq):
		if buildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_FoodSupplyStation) {
			return fmt.Errorf("buildType error, must be food supply station"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
		}
	case int32(cmd.Protocols_PC2LS_PlayerCampBuildFunOpReq):
		//if buildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_Furnace) || buildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_MaterialInstitute) || buildType != int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_EquipmentFoundry) {
		//	return fmt.Errorf("buildType error, must be food supply station"), int32(cmd.ErrorCode_CampBuildingTypeMismatch)
		//}

	default:
		return fmt.Errorf("interface not implemented"), int32(cmd.ErrorCode_InvalidParam)
	}
	return nil, int32(cmd.ErrorCode_Success)
}

// DelBuilding GM删除指定的建筑
func (h *CampHandler) DelBuilding(itermId, num int32, comdata *clidto.Comdata) {
	// 获取建筑的数量
	count := int32(0)
	for k, v := range h.actor.GetCampData().DecorationBuilding {
		// 在布局中没有使用
		if v.ItemId == itermId && !h.LayoutUse(v.BuildingId) {
			delete(h.actor.GetCampData().DecorationBuilding, k)
			h.Debugf("GM删除建筑[%d-%d]", v.BuildingId, v.ItemId)
			count += 1
			if count == num {
				break
			}
		}
	}
	h.Debugf("GM共删除删除建筑[%d],[%d]ge", itermId, count)
	h.SaveDB()
}

func (h *CampHandler) LayoutUse(buildingId int64) bool {
	for _, camp := range h.actor.GetCampData().Camp {
		for _, layout := range camp.Layout {
			if _, ok := layout.Building[buildingId]; ok {
				return true
			}
		}
	}
	return false
}

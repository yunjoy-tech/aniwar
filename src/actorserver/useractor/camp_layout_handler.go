package useractor

import (
	"context"
	"encoding/json"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"google.golang.org/protobuf/proto"
	"math"
	"time"
)

// CampLayoutReq 布局请求
func (h *CampHandler) CampLayoutReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampLayoutReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	res := &cmd.LS2C_PlayerCampLayoutRes{}
	layout := h.getCurLayout()
	camp := h.getCurCamp()
	if camp.HomeCoinStartTime == 0 {
		camp.HomeCoinStartTime = time.Now().Unix()
	}
	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	if layout != nil {
		res.Layout = ServerLayout2ClientLayout(layout.LayoutId, layout, true)
	}
	// 更新羁绊值
	h.actor.UserRelationHandler.CampUpdateRelation(camp.RoleList, h.actor.comData, common.Realtion_type_camp_life, false)
	h.actor.comData.Data.Camp = &cmd.PPlayerCampList{HomeCoinStartTime: camp.HomeCoinStartTime}
	res.CommonData = h.actor.comData.FixDownComData()

	return res, nil, int32(cmd.ErrorCode_Success)
}

// CampSwitchReq 切换布局
func (h *CampHandler) CampSwitchReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	if err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampSwitchReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	// 切换前氛围值
	curLayout := h.getCurLayout()
	beforeAtmosphere := curLayout.AtmosphereValue

	// 获取要切换的方案
	layout := h.getLayout(req.CampId, req.LayoutId)
	camp := h.getCamp(req.GetCampId())
	if layout == nil || camp == nil {
		return nil, fmt.Errorf("camp layout not exist"), int32(cmd.ErrorCode_CampLayoutNotExist)
	}

	h.setCurrentCampId(req.CampId)
	h.setCurrentLayoutId(req.LayoutId)
	// 修改入住人数
	roleList, code := h.ChangeRoleList()
	if code != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("get Ambience cfg by value is nil"), code
	}

	res := &cmd.LS2C_PlayerCampSwitchRes{
		CampId:   req.CampId,
		RoleList: roleList,
	}
	// 计算切换前可以产出多少 计算获得的币 个数
	dropChange, code := h.ChangeLayoutReturnHomeIcon(beforeAtmosphere, camp.HomeCoinStartTime)
	if code != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("C2LS_PlayerCampSwitchReq ChangeLayoutReturnHomeIcon err"), code
	}
	res.DropChange = dropChange
	camp.HomeCoinStartTime = time.Now().Unix()
	h.actor.comData.Data.Camp = &cmd.PPlayerCampList{HomeCoinStartTime: camp.HomeCoinStartTime}
	res.CommonData = h.actor.comData.FixDownComData()
	res.Layout = ServerLayout2ClientLayout(req.LayoutId, layout, true)

	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 切换布局方案 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampSwitchLayout{
	//		CustomHeadInfo:   lilith.BuildCustomHeadInfo(lilith.LogType_CampSwitchLayout, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		CampId:           req.CampId,             //营地id
	//		LayoutId:         req.LayoutId,           //布局id
	//		BeforeAtmosphere: beforeAtmosphere,       //切换前氛围值
	//		AfterAtmosphere:  layout.AtmosphereValue, //切换后氛围值
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampSwitchLayout{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CampId:            req.CampId,             //营地id
			LayoutId:          req.LayoutId,           //布局id
			BeforeAtmosphere:  beforeAtmosphere,       //切换前氛围值
			AfterAtmosphere:   layout.AtmosphereValue, //切换后氛围值
		}
		taptap.WriteDataLog(taptap.LogType_CampSwitchLayout, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return res, nil, 0
}

// CampLayoutSaveReq 布局保存请求
func (h *CampHandler) CampLayoutSaveReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	msgId, _, _ := in.MsgId, in.UserId, in.Data

	var req cmd.C2LS_PlayerCampLayoutSaveReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	// 公共检查
	commonParams := NewOutputParams(false, 0, 0, msgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}
	if commonParams.layout.X+commonParams.layout.Y > commonParams.camp.BlockMaxTimes {
		return nil, fmt.Errorf("block times not enough"), int32(cmd.ErrorCode_CampBlockTimesNotEnough)
	}

	//检测参数
	if codeErr := h.CheckoutXY(req, commonParams); codeErr != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("save layout CheckoutXY err"), codeErr
	}

	// 获取当前营地ID，用于确认建筑是否属于该营地
	currentCampId := h.getCurrentCampId()
	buildsMap := h.actor.GetCampData().DecorationBuilding

	funBuildingMap, atmosphereValMap := h.getFuncBuildingFromLayout(commonParams.layout) //
	var atmosphereValue int32

	// 历史布局信息
	// 当前营地所有布局的建筑使用次数统计
	curCampBuildUseCount, historyLayoutBuilds := h.buildingLayoutCount(currentCampId, commonParams.layout.LayoutId)

	buildMainMgr := excel.GetBuildMainMgr()

	builds := map[int64]*cmd.PPlayerCampCommonBuilding{}
	for _, v := range req.Building {
		if v.Building != nil {

			building, ok := buildsMap[v.Building.BuildingId] // 判断家具建筑是否已经建造
			if !ok {
				// 无中生有的建筑
				return nil, fmt.Errorf("building not exist"), int32(cmd.ErrorCode_CampBuildingNotExist)
			}
			_, ok = builds[building.BuildingId]
			if ok {
				// 存在重复数据
				return nil, fmt.Errorf("invalid param"), int32(cmd.ErrorCode_InvalidParam)
			}
			// 检查归属权问题，目前多营地不能摆放同建筑
			if !checkBuildingOwnership(currentCampId, building) {
				return nil, fmt.Errorf("building not exist"), int32(cmd.ErrorCode_CampBuildingNotExist)
			}

			// 功能建筑存在
			if len(funBuildingMap) > 0 {
				if _, exist := funBuildingMap[building.BuildingId]; exist {
					delete(funBuildingMap, building.BuildingId)
				}
			}

			if value, ok := atmosphereValMap[building.BuildingId]; ok {
				atmosphereValue += value
			} else {
				cfg := buildMainMgr.GetById(building.ItemId)
				if cfg == nil {
					h.Warnf("build main config not found, id: %d", building.ItemId)
				} else {
					atmosphereValue += cfg.Ambience
				}
			}

			building.CampId = currentCampId
			buildingData := h.NewPPlayerCampCommonBuilding(v.X, v.Y, v.ParentId, v.ParentGridId, v.Edge, v.Flip, building)
			builds[building.BuildingId] = buildingData
		}
	}

	//替换成新的家具
	commonParams.layout.Building = map[int64]*cmd.PPlayerCampCommonBuilding{}
	for _, v := range builds {
		delete(historyLayoutBuilds, v.Building.BuildingId)
		commonParams.layout.Building[v.Building.BuildingId] = v
	}

	// 释放掉删除的家具
	for k := range historyLayoutBuilds {
		useCount, ok := curCampBuildUseCount[k]
		if ok && useCount <= 1 {
			building, exist := buildsMap[k]
			if exist {
				building.CampId = 0
			}
		}
	}
	// 添加主题
	if req.GetThemeId() > 0 {
		commonParams.layout.ThemeId = req.GetThemeId() // 布局主题
		cfg := buildMainMgr.GetById(req.GetThemeId())
		if cfg == nil {
			h.Warnf("build main config not found, id: %d", req.GetThemeId())
		}
		atmosphereValue += cfg.Ambience
	}

	beforeAtmosphere := commonParams.layout.AtmosphereValue // 切换前氛围值
	commonParams.layout.AtmosphereValue = atmosphereValue
	commonParams.layout.Thumb = req.GetThumb() //缩略图
	h.Infof("保存布局[%d]缩略图:%+v", commonParams.layout.LayoutId, req.GetThumb())

	// 修改入住人数
	if _, codeErr := h.ChangeRoleList(); codeErr != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("save layout ChangeRoleList err"), codeErr
	}
	// 判断时候编辑了地块
	if req.GetX() >= 0 {
		h.Debug("跟新X边界，原来是值[%d],更新后的值[%d]", commonParams.layout.X, req.GetX())
		commonParams.layout.X = req.GetX()

	}
	if req.GetY() >= 0 {
		h.Debug("跟新Y边界，原来是值[%d],更新后的值[%d]", commonParams.layout.Y, req.GetY())
		commonParams.layout.Y = req.GetY()
	}

	res := &cmd.LS2C_PlayerCampLayoutSaveRes{}
	// 布局保存返还囤积的家装币
	dropChange, codeErr := h.ChangeLayoutReturnHomeIcon(beforeAtmosphere, commonParams.camp.HomeCoinStartTime)
	if codeErr != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("C2LS_PlayerCampSwitchReq ChangeLayoutReturnHomeIcon err"), codeErr
	}

	res.DropChange = dropChange
	commonParams.camp.HomeCoinStartTime = time.Now().Unix()
	h.actor.comData.Data.Camp = ToCommonData(commonParams)
	res.CommonData = h.actor.comData.FixDownComData()
	h.SaveDB()
	// 布局修改 埋点
	threading.RunSafe(func() {
		e := &taptap.CampSaveLayout{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			BeforeAtmosphere:  beforeAtmosphere,                    //切换前氛围值
			AfterAtmosphere:   commonParams.layout.AtmosphereValue, //切换后氛围值
		}
		taptap.WriteDataLog(taptap.LogType_CampSaveLayout, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
	return res, nil, 0
}

// CampLayoutModifyNameReq 布局名称修改
func (h *CampHandler) CampLayoutModifyNameReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampLayoutModifyNameReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	res := &cmd.LS2C_PlayerCampLayoutModifyNameRes{LayoutId: req.LayoutId, LayoutName: req.LayoutName}

	if !h.setLayoutName(req.LayoutId, req.LayoutName) {
		res.LayoutName = ""
	}

	return res, nil, 0
}

// CampRoleChangeReq  营地角色上阵
func (h *CampHandler) CampRoleChangeReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampRoleChangeReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	num := len(req.RoleList)
	roleSet := make(map[int32]interface{}, len(req.RoleList))
	cards := h.actor.Data.GetCards()
	if num > 0 {

		// 入住角色上限修改
		if num > h.GetRoleListLimit() {
			return nil, fmt.Errorf("camp role num limit, request num: %d", num), int32(cmd.ErrorCode_CampRoleNumLimit)
		}
		if cards == nil || cards.Card == nil {
			return nil, fmt.Errorf("player have nothing card"), int32(cmd.ErrorCode_CardNotExist)
		}
		for _, roleId := range req.RoleList {
			if _, exist := roleSet[roleId]; exist {
				return nil, fmt.Errorf("role duplicate,roleId:%d", roleId), int32(cmd.ErrorCode_CampRoleDuplicate)
			}
			roleSet[roleId] = struct{}{}
			if _, ok := cards.Card[uint32(roleId)]; !ok {
				return nil, fmt.Errorf("card not exist, request cardId: %d", roleId), int32(cmd.ErrorCode_CardNotExist)
			}
		}
	}

	beforeCard := h.getCurCamp().RoleList
	h.getCurCamp().RoleList = req.RoleList

	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	// 更新羁绊值
	h.actor.UserRelationHandler.CampUpdateRelation(req.GetRoleList(), h.actor.comData, common.Realtion_type_camp_life, true)
	threading.RunSafe(func() {
		e := &taptap.CamproleChange{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Count:             num,                                  //当前上阵数量上限
			BeforeCard:        taptap.ConvertList2Str(beforeCard),   //上阵前卡牌列表
			AfterCard:         taptap.ConvertList2Str(req.RoleList), //上阵后卡牌列表
		}
		taptap.WriteDataLog(taptap.LogType_CamproleChange, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &cmd.LS2C_PlayerCampRoleChangeRes{
		CommonData: h.actor.comData.FixDownComData(),
	}, nil, int32(cmd.ErrorCode_Success)
}

func (h *CampHandler) CampGetHomeCoinReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampGetHomeCoinReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	camp := h.getCamp(req.GetCampId())
	if camp == nil {
		return nil, fmt.Errorf("CampGetHomeCoinReq camp is nil"), int32(cmd.ErrorCode_InvalidParam)
	}
	nowTime := time.Now().Unix()
	if nowTime < camp.HomeCoinStartTime {
		return nil, fmt.Errorf(" CampGetHomeCoinReq the get time has not arrived"), int32(cmd.ErrorCode_InternalError)
	}

	// 根据氛围值获取配置
	cfg := h.GetCurAmbienceCfg()
	if cfg == nil {
		return nil, fmt.Errorf("CampGetHomeCoinReq get Ambience Cfg is err"), int32(cmd.ErrorCode_InternalError)
	}

	var rewards = make(map[int32]int32)
	// 计算获得的币 个数
	num := h.ComputeHomeCoinNum(h.getCurLayout().AtmosphereValue, camp.HomeCoinStartTime)
	if num <= 0 {
		return nil, fmt.Errorf("no home icon can get "), int32(cmd.ErrorCode_CampNoEnoughHomeIcon)
	}

	// 是否超过上限
	if num > cfg.BuildcurrencyProducelimit {
		num = cfg.BuildcurrencyProducelimit
	}
	h.Infof("领取家装币,开始时间[%d],领取数量[%d]", camp.HomeCoinStartTime, num)
	rewards[common.CURRENCY_ITEM_ID_2012] = num
	dropChange, err := GetDropMgr(h.actor).DropList2(rewards, true, nil, h.actor.comData, common.CR_CAMP_HOME_COIN)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	camp.HomeCoinStartTime = nowTime
	h.actor.comData.Data.Camp = &cmd.PPlayerCampList{HomeCoinStartTime: camp.HomeCoinStartTime}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	res := &cmd.LS2C_PlayerCampGetHomeCoinRes{
		DropChange: dropChange,
		CommonData: h.actor.comData.FixDownComData(),
	}
	return res, nil, int32(cmd.ErrorCode_Success)
}

func (h *CampHandler) CampBuyEditBlockTimes(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	_, uid, _ := in.MsgId, in.UserId, in.Data
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampBuyEditBlockTimesReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	//获取当前营地
	curCamp := h.getCurCamp()
	if curCamp == nil {
		return nil, fmt.Errorf("edit block get curCamp err "), int32(cmd.ErrorCode_InternalError)
	}

	//判断是否达到最大次数
	maxTimes, code := h.GetMaxEditTimes()
	if code != cmd.ErrorCode_Success {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	if curCamp.BlockMaxTimes+1 > maxTimes[1] {
		return nil, fmt.Errorf("edit block over max times "), int32(cmd.ErrorCode_CampOverMaxTimes)
	}

	//获取购买消耗
	cost, code := h.GetEditTimesCost(curCamp.BlockMaxTimes + 1)
	if code != cmd.ErrorCode_Success {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	//扣除消耗
	//检查道具是否充足，如果充足直接消费掉
	if !h.consumerItem(uid, cost, h.actor.comData, common.CR_CAMP_LAYOUT_EDIT) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}

	curCamp.BlockMaxTimes += 1
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	h.Info("成功购买一次营地编辑次数,最大次数为[%d]", curCamp.BlockMaxTimes)
	res := &cmd.LS2C_PlayerCampBuyEditBlockTimesRes{
		CommonData:    h.actor.comData.FixDownComData(),
		BlockMaxTimes: curCamp.BlockMaxTimes,
	}
	return res, nil, int32(cmd.ErrorCode_Success)
}

func (h *CampHandler) CampGetThumbReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampGetThumbReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	//获取当前营地
	camp := h.getCamp(req.GetCampId())
	if camp == nil {
		return nil, err, int32(cmd.ErrorCode_ParamError)
	}
	h.Infof("获取营地缩略图:%d,%+v", len(camp.GetLayout()), camp.GetLayout())
	res := &cmd.LS2C_PlayerCampGetThumbRes{}
	for id, layout := range camp.GetLayout() {
		res.Thumbs = append(res.Thumbs, &cmd.PlayerCampThumb{
			LayoutId: id,
			Thumb:    layout.Thumb,
		})
	}
	return res, nil, int32(cmd.ErrorCode_Success)
}

////////////////////////////////////////////内部调用方法

// 指定营地所有布局的建筑使用次数统计
func (h *CampHandler) buildingLayoutCount(campId, layoutId int32) (map[int64]int32, map[int64]bool) {
	camp := h.getCamp(campId)
	buildingsUseCount := make(map[int64]int32)
	historyLayoutBuilds := make(map[int64]bool)
	if camp == nil || camp.Layout == nil {
		return buildingsUseCount, historyLayoutBuilds
	}

	for _, v := range camp.Layout { // 所有布局
		if v.Building != nil {
			for k := range v.Building {
				if v.LayoutId == layoutId {
					historyLayoutBuilds[k] = true
				}
				buildingsUseCount[k]++
			}
		}
	}
	return buildingsUseCount, historyLayoutBuilds
}

func (h *CampHandler) setLayoutName(layoutId int32, layoutName string) bool {
	for _, v := range h.actor.GetCampData().Camp {
		if v.CampId == h.getCurrentLayoutId() && v.Layout != nil {
			if layout, ok := v.Layout[layoutId]; ok {
				layout.LayoutName = layoutName
				return true
			}
		}
	}
	return false
}

func (h *CampHandler) setCurrentCampId(campId int32) {
	h.actor.GetCampData().CurrentCampId = campId
}

func (h *CampHandler) setCurrentLayoutId(layoutId int32) {
	h.actor.GetCampData().CurrentLayoutId = layoutId
}

// GetRoleListLimit 获取入住角色上限
func (h *CampHandler) GetRoleListLimit() int {
	// 获取当前营地的氛围值
	curLayout := h.getCurLayout()
	if curLayout == nil {
		return 0
	}
	// 更具氛围值去表里查找氛围值等级
	cfg := h.GetAmbienceByValue(curLayout.AtmosphereValue)
	if cfg == nil {
		h.Debug("get Ambience cfg by value is nil:", curLayout.AtmosphereValue)
		return 0
	}
	return int(cfg.RoleLimit)
}
func (h *CampHandler) GetCurAmbienceCfg() *excel.AmbienceCfg {
	return h.GetAmbienceByValue(h.getCurLayout().AtmosphereValue)
}

func (h *CampHandler) GetAmbienceByValue(value int32) *excel.AmbienceCfg {
	maxValue := int32(0)
	var cfgAm *excel.AmbienceCfg
	excel.GetAmbienceMgr().Foreach(func(cfg *excel.AmbienceCfg) bool {
		if value < cfg.AmbienceNum {
			return true
		}
		if cfg.AmbienceNum >= maxValue {
			cfgAm = cfg
			maxValue = cfg.AmbienceNum
		}
		return true
	}, true)
	return cfgAm
}
func (h *CampHandler) ComputeHomeCoinNum(ambienceValue int32, homeCoinStartTime int64) int32 {
	nowTime := time.Now().Unix()
	cfg := h.GetAmbienceByValue(ambienceValue)
	if cfg == nil {
		return 0
	}

	//计算时间插值
	timeDiff := float64((nowTime - homeCoinStartTime)) / float64(3600)

	// 计算获得的币 个数
	num := timeDiff * float64(cfg.BuildcurrencyProduce)
	h.Debugf("计算要领取的家装币 开始时间[%d] 领取时间[%d] 时间插值[%f] 获取效率[%d] 计算的数量[%f]", homeCoinStartTime, nowTime, timeDiff, cfg.BuildcurrencyProduce, num)
	return int32(math.Floor(num))
}
func (h *CampHandler) ChangeLayoutReturnHomeIcon(beforeAtmosphere int32, startTime int64) (*cmd.DropChange, int32) {
	var rewards = make(map[int32]int32)
	num := h.ComputeHomeCoinNum(beforeAtmosphere, startTime)
	if num > 0 {
		cfg := h.GetAmbienceByValue(beforeAtmosphere)
		if cfg == nil {
			return nil, int32(cmd.ErrorCode_InternalError)
		}
		// 是否超过上限
		if num > cfg.BuildcurrencyProducelimit {
			num = cfg.BuildcurrencyProducelimit
		}
		h.Debugf("领取家装币返还,开始时间[%d],领取数量[%d]", startTime, num)
		rewards[common.CURRENCY_ITEM_ID_2012] = num
		dropChange, err := GetDropMgr(h.actor).DropList2(rewards, true, nil, h.actor.comData, common.CR_CAMP_HOME_COIN)
		if err != nil {
			return nil, int32(cmd.ErrorCode_InternalError)
		}
		return dropChange, int32(cmd.ErrorCode_Success)
	}
	return nil, int32(cmd.ErrorCode_Success)
}

// 获取建筑氛围值
func (h *CampHandler) getAtmosphereValue(buildItemId int32) int32 {
	cfg := excel.GetBuildMainMgr().GetById(buildItemId)
	if cfg == nil {
		h.Warnf("buildMain config not found, id:%d", buildItemId)
		return 0
	}
	return cfg.Ambience
}

func (h *CampHandler) ChangeRoleList() ([]int32, int32) {
	curLayout := h.getCurLayout()
	cfg := h.GetAmbienceByValue(curLayout.AtmosphereValue)
	if cfg == nil {
		return nil, int32(cmd.ErrorCode_NotFoundConfig)
	}
	camp := h.getCamp(h.getCurrentCampId())
	if camp == nil {
		return nil, int32(cmd.ErrorCode_InternalError)
	}
	if len(camp.RoleList) > int(cfg.RoleLimit) {
		camp.RoleList = camp.RoleList[:cfg.RoleLimit]
	}
	return camp.RoleList, int32(cmd.ErrorCode_Success)
}

func (h *CampHandler) CheckoutXY(req cmd.C2LS_PlayerCampLayoutSaveReq, commonParams *OutputParams) int32 {
	if req.GetX() < 0 || req.GetY() < 0 {
		h.Debug("save layout x ,y lt 0:", req.GetX(), req.GetY())
		return int32(cmd.ErrorCode_ParamError)
	}
	//maxTimes, code := h.GetMaxEditTimes()
	//if code != cmd.ErrorCode_Success {
	//	return int32(cmd.ErrorCode_ParamError)
	//}
	if req.GetX()+req.GetY() > commonParams.camp.BlockMaxTimes {
		h.Debug("save layout x ,y limit max times:", req.GetX(), req.GetY())
		return int32(cmd.ErrorCode_ParamError)
	}

	return int32(cmd.ErrorCode_Success)
}

// 检查建筑归属，目前同建筑不用于摆放在不同的营地
func checkBuildingOwnership(currentCampId int32, svrBuilding *cmd.PPlayerCampDecorationBuilding) bool {
	// 建筑无归属权或者建筑归属权属于该营地
	if svrBuilding.CampId == 0 || currentCampId == svrBuilding.CampId {
		return true
	}
	return false
}

// 功能建筑不允许撤回
func (h *CampHandler) getFuncBuildingFromLayout(layout *cmd.PPlayerCampServerLayout) (map[int64]bool, map[int64]int32) {
	funBuildingMap := make(map[int64]bool)
	atmosphereValMap := make(map[int64]int32)
	if layout == nil {
		return funBuildingMap, atmosphereValMap
	}
	mgr := excel.GetBuildMainMgr()
	for k, v := range layout.Building {
		cfg := mgr.GetById(v.Building.ItemId)
		if cfg == nil {
			h.Warnf("build main config not found, id: %d", v.Building.ItemId)
			continue
		}
		atmosphereValMap[k] = cfg.Ambience
		if cfg.BuildType > int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_None) &&
			cfg.BuildType < int32(cmd.PlayerCampBuildingType_PlayerCampBuildingType_Max) {
			funBuildingMap[k] = true
		}
	}
	return funBuildingMap, atmosphereValMap
}

func (h *CampHandler) DefaultLayout() *cmd.PPlayerCampServerLayout {
	defaultLayout := excel.GetConfigMgr().GetCfg().CAMP_LAYOUT_DEFAULT
	layout := &cmd.PPlayerCampServerLayout{}
	if err := json.Unmarshal([]byte(defaultLayout), layout); err != nil {
		h.Debug("Unmarshal defaultLayout err:", err)
		return nil
	}
	layout.AtmosphereValue = 0
	for _, v := range layout.Building {
		themeCfg := excel.GetBuildMainMgr().GetById(v.Building.ItemId)
		h.Debugf("默认布局ItemId[%d],氛围值[%d]", v.Building.ItemId, themeCfg.Ambience)
		layout.AtmosphereValue += themeCfg.Ambience
		//按照配置生成配置
		buildData := h.NewPPlayerCampDecorationBuilding(v.Building.ItemId, 1, 0)
		h.actor.GetCampData().DecorationBuilding[buildData.BuildingId] = buildData
		v.Building.BuildingId = buildData.BuildingId
	}
	h.Debugf("除主题之外的氛围总值[%d]", layout.AtmosphereValue)
	themeCfg := excel.GetBuildMainMgr().GetById(90189)
	if themeCfg == nil {
		h.Warnf("build main config not found, id: %d", 90189)
	}
	layout.AtmosphereValue += themeCfg.Ambience

	return layout
}

func (h *CampHandler) ChangeHomeIconTime(startTime int64) {
	camp := h.getCurCamp()
	camp.HomeCoinStartTime = startTime
	h.SaveDB()
}

// GetMaxEditTimes 最大次数,扩建消耗
func (h *CampHandler) GetMaxEditTimes() ([]int32, cmd.ErrorCode) {
	times := excel.GetConfigMgr().GetCfg().CAMP_EXPAND_LIMIT
	if len(times) < 2 {
		return nil, cmd.ErrorCode_ConfigError
	}

	return times, cmd.ErrorCode_Success
}

// GetEditTimesCost 获取增加扩建次数消耗
func (h *CampHandler) GetEditTimesCost(maxTimes int32) (map[int32]int32, cmd.ErrorCode) {
	times, err := h.GetMaxEditTimes()
	if err != cmd.ErrorCode_Success {
		return nil, err
	}

	cfgCost := excel.GetConfigMgr().GetCfg().CAMP_EXPAND_COST
	cost := make(map[int32]int32, 0)
	cost[common.CURRENCY_ITEM_ID_2012] = cfgCost[int(maxTimes)-int(times[0])-1]

	return cost, cmd.ErrorCode_Success
}

func ServerLayout2ClientLayout(layoutId int32, serverLayout *cmd.PPlayerCampServerLayout, build bool) *cmd.PPlayerCampClientLayout {
	resLayout := &cmd.PPlayerCampClientLayout{
		LayoutId:        layoutId,
		LayoutName:      serverLayout.LayoutName,
		AtmosphereValue: serverLayout.AtmosphereValue,
		ThemeId:         serverLayout.ThemeId,
		X:               serverLayout.X,
		Y:               serverLayout.Y,
	}
	if build {
		for _, v := range serverLayout.Building {
			resLayout.Building = append(resLayout.Building, v)
		}
	}
	return resLayout
}

func ToCommonData(commonParams *OutputParams) *cmd.PPlayerCampList {
	campList := &cmd.PPlayerCampList{
		HomeCoinStartTime: commonParams.camp.HomeCoinStartTime,
		Camp:              make([]*cmd.PPlayerCamp, 0),
	}

	layout := &cmd.PPlayerCampClientLayout{
		LayoutId:        commonParams.layout.LayoutId,
		AtmosphereValue: commonParams.layout.AtmosphereValue,
	}

	campList.Camp = append(campList.Camp, &cmd.PPlayerCamp{Layout: []*cmd.PPlayerCampClientLayout{layout}})
	return campList
}

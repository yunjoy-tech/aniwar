// 常规副本
// 1.资源副本，每日轮替
// 2.常驻副本

package useractor

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"

	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type CampaignHandler struct {
	*UABaseHandler
}

func NewCampaignHandler(actor *UserActor) *CampaignHandler {
	h := &CampaignHandler{UABaseHandler: NewUABaseHandler(actor, "CampaignHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerGeneralCampaignListReq), h.CampaignListReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampaignEnterLevelReq), h.CampaignEnterLevelReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CampaignBattleStartReq), h.BattleStartReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CampaignBattleEndReq), h.BattleEndReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerCampaignDriveSettleReq), h.GameEndReq)

	return h
}

// Init 角色登录游戏时调用
// 每次都读取配置是为了应对配置更新
func (h *CampaignHandler) Init() error {
	h.actor.Data.CampaignInfo = &cmd.PPlayerGeneralCampaign{
		Createtime: time.Now().Unix(),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init CampaignHandler data success. player: %s", h.actor.ID())
	return nil
}

func (h *CampaignHandler) EnterGame() error {
	return nil
}

func (h *CampaignHandler) DailyRefresh() error {
	// 清除日替中的通关次数
	h.resetCompletedCount()

	return nil
}

func (h *CampaignHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PPlayerGeneralCampaign); ok {
		h.actor.Data.CampaignInfo = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *CampaignHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyCampaign(h.actor.ID()), h.actor.Data.CampaignInfo
}

func (h *CampaignHandler) getCampaignInfo() *cmd.PPlayerGeneralCampaign {
	return h.actor.GetCampaignData()
}

func (h *CampaignHandler) getCurrCampaign() *cmd.PCommonGeneralCampaignInfo {
	return h.actor.GetCampaignData().CurCampaign
}

// 重置通关次数
func (h *CampaignHandler) resetCompletedCount() {
	now := time.Now()
	t := time.Unix(h.getLastUpdateTimestamp(), 0)
	isSameDay := common.IsSameDayByOffset(t, now, common.GAME_DAILY_REFRESH_HOUR)
	if isSameDay {
		return
	}

	dbCampaignData := h.getCampaignInfo().ServerGeneralCampaign
	// 日替本次数共用
	dbCampaignData.Res97CampaignCompleteCount = 0
	dbCampaignData.Res98CampaignCompleteCount = 0
	// 常驻本次数不共用
	for _, v := range dbCampaignData.DailyCampaigns {
		v.CompleteCount = 0
	}

	//更新时间戳
	h.actor.Data.CampaignInfo.LastUpdateTimestamp = now.Unix()

	h.Warnf("玩家%s, 每日重置通关次数, %s", h.actor.ID(), common.FormatDateByUnix(now.Unix()))
	err := h.SaveDB()
	if err != nil {
		h.Warnf("玩家%s, 每日重置通关次数 报错, %v", h.actor.ID(), err.Error())
	}
}

func (h *CampaignHandler) IsDailyCampaignFirstPass(campaignId, subId int32) bool {
	dailyCampaign := h.getDailyCampaign(campaignId)
	// 首通
	if !Contains(dailyCampaign.CompletedStage, subId) {
		h.Debugf("玩家%s, Daily关卡:%d - %d, 首次通过", h.actor.ID(), campaignId, subId)
		return true
	}
	h.Debugf("玩家%s, Daily关卡:%d - %d, 通过", h.actor.ID(), campaignId, subId)

	return false
}

func (h *CampaignHandler) updateDriveMaxScore(campaignId, subCampaignId, score int32) {
	dailyCampaign := h.getDailyCampaign(campaignId)

	hadFound := false
	for _, each := range dailyCampaign.DriveScore {
		if each.SubCampaignId == subCampaignId { // 有历史数据
			if each.Score < score {
				each.Score = score // 更新最高分数
			}

			hadFound = true
			break
		}
	}
	if !hadFound {
		// 最高分数据
		campaignScoreData := &cmd.DriveCampaiginScore{
			SubCampaignId: subCampaignId,
			Score:         score,
		}
		dailyCampaign.DriveScore = append(dailyCampaign.DriveScore, campaignScoreData)
	}
}

// 更新日替副本的基础信息
func (h *CampaignHandler) IsResCampaignFirstPass(campaignId, subId int32) bool {
	resCampaign := h.getResourceCampaigns()
	data := resCampaign[campaignId]
	if !Contains(data.CompletedStage, subId) {
		//data.CompletedStage = append(data.CompletedStage, subId)
		h.Debugf("玩家%s, Resource关卡:%d - %d, 首次通过", h.actor.ID(), campaignId, subId)
		return true
	}
	h.Debugf("玩家%s, Resource关卡:%d - %d, 通过", h.actor.ID(), campaignId, subId)
	return false
}

// 构建客户端数据
func (h *CampaignHandler) buildClientCampaignData() *cmd.PClientGeneralCampaign {
	campaignInfo := h.getCampaignInfo().ServerGeneralCampaign
	clientCampaign := &cmd.PClientGeneralCampaign{
		//ResCampaignCompleteCount: campaignInfo.GetResCampaignCompleteCount(),
		Res97CampaignCompleteCount: campaignInfo.Res97CampaignCompleteCount,
		Res98CampaignCompleteCount: campaignInfo.Res98CampaignCompleteCount,
		DailyCampaign:              make([]*cmd.PDailyCampaignBase, 0, 2),
		ResourceCampaign:           make([]*cmd.PResourceCampaignBase, 0, 4),
	}

	for _, v := range campaignInfo.DailyCampaigns {
		clientCampaign.DailyCampaign = append(clientCampaign.DailyCampaign, v)
	}
	for _, v := range campaignInfo.ResCampaigns {
		clientCampaign.ResourceCampaign = append(clientCampaign.ResourceCampaign, v)
	}
	return clientCampaign
}

func Contains[T comparable](s []T, v T) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

// 开车、金币副本数据
func (h *CampaignHandler) getDailyCampaigns() map[int32]*cmd.PDailyCampaignBase {
	dailyCampaigns := h.getCampaignInfo().ServerGeneralCampaign.DailyCampaigns
	return dailyCampaigns
}

// 开车、金币副本数据
func (h *CampaignHandler) getDailyCampaign(campaignId int32) *cmd.PDailyCampaignBase {
	campaigns := h.getDailyCampaigns()

	if campaign, ok := campaigns[campaignId]; !ok {
		campaign = &cmd.PDailyCampaignBase{CampaignId: campaignId,
			CompleteCount:  0,
			CompletedStage: make([]int32, 0),
			DriveScore:     make([]*cmd.DriveCampaiginScore, 0)}
		campaigns[campaignId] = campaign
		return campaign
	} else {
		return campaign
	}
}

// 日替副本数据
func (h *CampaignHandler) getResourceCampaigns() map[int32]*cmd.PResourceCampaignBase {
	resCampaigns := h.getCampaignInfo().ServerGeneralCampaign.ResCampaigns
	return resCampaigns
}

// 日替副本数据
func (h *CampaignHandler) getResourceCampaign(campaignId int32) *cmd.PResourceCampaignBase {
	campaigns := h.getResourceCampaigns()

	if campaign, ok := campaigns[campaignId]; !ok {
		campaign = &cmd.PResourceCampaignBase{
			CampaignId:     campaignId,
			CompletedStage: make([]int32, 0),
		}
		campaigns[campaignId] = campaign
		return campaign
	} else {
		return campaign
	}
}

func (h *CampaignHandler) getLastUpdateTimestamp() int64 {
	return h.getCampaignInfo().LastUpdateTimestamp
}

// 检查进入条件
func (h *CampaignHandler) checkEnterCondition(req *cmd.C2LS_PlayerCampaignEnterLevelReq) (error, cmd.ErrorCode) {
	campaignCfg := excel.GetCampaignMgr().GetById(req.SubCampaignId)
	if campaignCfg == nil {
		return fmt.Errorf("未找到配置, subCampaignId=%d", req.SubCampaignId), cmd.ErrorCode_NotFoundConfig
	}
	if h.actor.LoginHandler.getRoleLevel() < uint32(campaignCfg.MinRoleLv) {
		return fmt.Errorf("玩家等级不足, 当前等级:%d, 需要:%d", h.actor.LoginHandler.getRoleLevel(), campaignCfg.MinRoleLv),
			cmd.ErrorCode_CampaignLvNotMeet
	}

	campaignType := common.CAMPAIGN_TYPE(campaignCfg.CampaignType)

	//cfgs := datahelper.GetResourceCampaign(req.CampaignId, campaignType)

	switch campaignType {
	case common.CAMPAIGN_TYPE_97, common.CAMPAIGN_TYPE_98: // 日替
		//dbCampaign := h.getCampaignInfo().ServerGeneralCampaign

		now := time.Now()
		_, subCampaignIds := getCampaignOpenedList(common.GetWeekday(&now))
		if !Contains(subCampaignIds, req.SubCampaignId) {
			return fmt.Errorf("关卡当前日期不开放, 当天开放关卡:%v, 请求的关卡:%d", subCampaignIds, req.SubCampaignId),
				cmd.ErrorCode_CampaignDateError
		}
		resCfg := excel.GetResourcecampaignMgr().GetById(req.CampaignId)
		if resCfg == nil {
			return fmt.Errorf("未找到配置, subCampaignId=%d", req.CampaignId),
				cmd.ErrorCode_NotFoundConfig
		}
		if !Contains(resCfg.Includecampaign, req.SubCampaignId) {
			return fmt.Errorf("配置中未包含改关卡, subCampaignId=%d", req.SubCampaignId),
				cmd.ErrorCode_NotFoundConfig
		}

		resCampaign := h.getResourceCampaign(req.CampaignId)

		// 前置条件检查
		preLevelId := campaignCfg.PreCampaignId
		if preLevelId != 0 && !Contains(resCampaign.CompletedStage, preLevelId) {
			return fmt.Errorf("前置关卡未完成, preLevelId=%d", preLevelId),
				cmd.ErrorCode_CampaignRequirePreLevel
		}
		// 次数检查 -- 剔除次数限制 2023年8月14日14:19:04
		/*hadCompleteCount, err := getResCompleteCount(dbCampaign, campaignCfg.CampaignType) // 已用次数
		if err != nil {
			return err, cmd.ErrorCode_InternalError
		}
		limitCount, err := getResCfgCount(campaignCfg.CampaignType) // 限制次数
		if err != nil {
			return err, cmd.ErrorCode_ConfigError
		}
		if hadCompleteCount >= limitCount {
			return fmt.Errorf("当天已经次数用完, 已用次数=%d, 限制次数=%d", hadCompleteCount, limitCount),
				cmd.ErrorCode_CampaignNoTimes
		}*/

	case common.CAMPAIGN_TYPE_99, common.CAMPAIGN_TYPE_100: // 开车	// 金币
		dailyCfg := excel.GetDailycampaignMgr().GetById(req.CampaignId)
		//dailyCfg := excel.GetResourcecampaignMgr().GetById(req.CampaignId)
		if dailyCfg == nil {
			return fmt.Errorf("未找到配置, subCampaignId=%d", req.SubCampaignId),
				cmd.ErrorCode_NotFoundConfig
		}
		if !Contains(dailyCfg.Includecampaign, req.SubCampaignId) {
			return fmt.Errorf("配置中未包含改关卡, subCampaignId=%d", req.SubCampaignId),
				cmd.ErrorCode_NotFoundConfig
		}

		dailyCampaign := h.getDailyCampaign(req.CampaignId)

		// 前置关卡是否完成
		preLevelId := campaignCfg.PreCampaignId
		if preLevelId != 0 && !Contains(dailyCampaign.CompletedStage, preLevelId) {
			return fmt.Errorf("前置关卡未完成, preLevelId=%d", preLevelId),
				cmd.ErrorCode_CampaignRequirePreLevel
		}
		// 次数检查
		if dailyCampaign.CompleteCount >= dailyCfg.Enterlimit {
			return fmt.Errorf("当天已经次数用完, 已用次数=%d, 限制次数=%d", dailyCampaign.CompleteCount, dailyCfg.Enterlimit),
				cmd.ErrorCode_CampaignNoTimes
		}

	default:
		return fmt.Errorf("不存在的关卡类型, %d", campaignCfg.CampaignType), cmd.ErrorCode_NotFoundConfig
	}

	return nil, cmd.ErrorCode_Success
}

// resource副本 - 获取配置的最大可完成次数
func getResCfgCount(campaignType int32) (int32, error) {
	switch common.CAMPAIGN_TYPE(campaignType) {
	case common.CAMPAIGN_TYPE_97:
		return excel.GetConfigMgr().GetCfg().DAILY2TICKETLIMIT, nil
	case common.CAMPAIGN_TYPE_98: // 日替
		return excel.GetConfigMgr().GetCfg().DAILYTICKETLIMIT, nil
	default:
		return 0, fmt.Errorf("未实现的类型, %v", campaignType)
	}
}

// resource副本 - 获取完成次数
func getResCompleteCount(campaign *cmd.PServerGeneralCampaign, campaignType int32) (int32, error) {
	switch common.CAMPAIGN_TYPE(campaignType) {
	case common.CAMPAIGN_TYPE_97:
		return campaign.Res97CampaignCompleteCount, nil
	case common.CAMPAIGN_TYPE_98: // 日替
		return campaign.Res98CampaignCompleteCount, nil
	default:
		return 0, fmt.Errorf("未实现的类型, %v", campaignType)
	}
}

/*// resource副本 - 累计完成次数
func incrResCompleteCount(campaign *cmd.PServerGeneralCampaign, campaignType int32, incr int32) error {
	switch common.CAMPAIGN_TYPE(campaignType) {
	case common.CAMPAIGN_TYPE_97:
		campaign.Res97CampaignCompleteCount += incr
	case common.CAMPAIGN_TYPE_98: // 日替
		campaign.Res98CampaignCompleteCount += incr
	default:
		return fmt.Errorf("未实现的类型, %v", campaignType)
	}

	return nil
}*/

// CampaignListReq 获取副本列表
func (h *CampaignHandler) CampaignListReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	//_, _, data := in.MsgId, in.UserId, in.Data
	var req cmd.C2LS_PlayerGeneralCampaignListReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}
	now := time.Now()
	resCampaignIds, _ := getCampaignOpenedList(common.GetWeekday(&now))
	res := &cmd.LS2C_PlayerGeneralCampaignListRes{OpenCampaigns: resCampaignIds,
		GeneralCampaigns: h.buildClientCampaignData()}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampaignList{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Campaign_list, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		OpenCampaigns:  lilith.ConvertList2Str(res.OpenCampaigns),
	//		//GeneralCampaigns: res.GeneralCampaigns,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampaignList{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			OpenCampaigns:     taptap.ConvertList2Str(res.OpenCampaigns),
			//GeneralCampaigns: res.GeneralCampaigns,
		}
		taptap.WriteDataLog(taptap.LogType_Campaign_list, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return res, nil, 0
}

// CampaignEnterLevelReq 进入副本请求
func (h *CampaignHandler) CampaignEnterLevelReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err     error
		errCode cmd.ErrorCode
	)

	err, errCode = h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1006)
	if err != nil {
		return nil, err, int32(errCode)
	}

	var req cmd.C2LS_PlayerCampaignEnterLevelReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 检查挑战条件
	err, errCode = h.checkEnterCondition(&req)
	if err != nil {
		return nil, err, int32(errCode)
	}

	err, errCode = h.checkTeam(req.SubCampaignId, req.Teams)
	if err != nil {
		return nil, err, int32(errCode)
	}

	now := time.Now().Unix()
	// 构建当前打的关卡数据
	curCampaignInfo := &cmd.PCommonGeneralCampaignInfo{
		CampaignId: req.CampaignId,
		//CampaignType:    req.CampaignType,
		SubCampaignId:   req.SubCampaignId,
		StartTimestamp:  now,
		ExpireTimestamp: now + 900, // 无需求, 不做验证
		Teams:           req.Teams,
	}
	h.getCampaignInfo().CurCampaign = curCampaignInfo
	err = h.SaveDB()
	if err != nil {
		h.Warnf("玩家%s, 进入副本 报错, %v", h.actor.ID(), err.Error())
	}

	h.actor.comData.Data.Campaign = h.buildClientCampaignData()

	res := &cmd.LS2C_PlayerCampaignEnterLevelRes{
		CurCampaign: curCampaignInfo,
		CommonData:  h.actor.comData.FixDownComData(),
	}

	// 队伍信息
	teamList := make([]*cmd.GeneralTeamTemp, 0)
	for _, value := range res.CurCampaign.Teams {
		teamTemp := &cmd.GeneralTeamTemp{}
		teamTemp.TeamNumber = int64(value.TeamNumber)        // 队伍编号
		teamTemp.Cards = taptap.ConvertList2Str(value.Cards) // 上阵卡牌ID列表
		teamList = append(teamList, teamTemp)                // 队伍信息
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampaignEnter{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Campaign_enter, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		CampaignId:     res.CurCampaign.CampaignId,             // 副本id
	//		SubCampaignId:  res.CurCampaign.SubCampaignId,          // 副本子类型
	//		Teams:          lilith.ConvertListStruct2Str(teamList), // 队伍信息
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampaignEnter{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CampaignId:        res.CurCampaign.CampaignId,             // 副本id
			SubCampaignId:     res.CurCampaign.SubCampaignId,          // 副本子类型
			Teams:             taptap.ConvertListStruct2Str(teamList), // 队伍信息
		}
		taptap.WriteDataLog(taptap.LogType_Campaign_enter, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return res, nil, 0
}

// 按分数档位获取奖励
func getRewardsByScore(campaignId, subCampaignId, score int32) (bool, map[int32]int32) {
	var (
		pass      = false // 是否胜利
		rewardMap = make(map[int32]int32, 0)
	)

	driveRewardCfg := excel.GetCampaignrewardMgr().GetById(subCampaignId)
	if driveRewardCfg == nil {
		return pass, rewardMap
	}

	// 按分数档位获取奖励
	if score >= driveRewardCfg.SRankScore {
		datahelper.MergeKeyVal(rewardMap, driveRewardCfg.SReward)
	} else if score >= driveRewardCfg.ARankScore {
		datahelper.MergeKeyVal(rewardMap, driveRewardCfg.AReward)
	} else if score >= driveRewardCfg.BRankScore {
		datahelper.MergeKeyVal(rewardMap, driveRewardCfg.BReward)
	} else if score >= driveRewardCfg.CRankScore {
		datahelper.MergeKeyVal(rewardMap, driveRewardCfg.CReward)
	} else if score >= driveRewardCfg.DRankScore {
		datahelper.MergeKeyVal(rewardMap, driveRewardCfg.DReward)
	}

	// 达到最低档位，算胜利
	if score >= driveRewardCfg.DRankScore {
		pass = true
	}

	return pass, rewardMap
}

// 发放其他奖励，货币奖励和道具奖励
func (h *CampaignHandler) commonAddReward(roleList []int32, rewardMap map[int32]int32, commonData *clidto.Comdata, reason common.ChangeReason) /*([]*cmd.ItemReward, []*cmd.CommonCardExpReward) */ {
	dropChange, err := GetDropMgr(h.actor).DropList2(rewardMap, true, roleList, commonData, common.CR_Camp_Building_Get)
	h.Debugf("commonAddReward dropchange:%+v rewardMap:%v roleList:%v", dropChange, rewardMap, roleList)
	if err != nil {
		h.Error("Campaign commAddReward DropListByItems, error", h.actor.ID(), err)
	}
}

func (h *CampaignHandler) BattleStartReq(uid context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req cmd.C2LS_CampaignBattleStartReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	battleId := uint64(utils.GenIntUUID())
	rseed := utils.GenIntUUID()

	campaignCfg := excel.GetCampaignMgr().GetById(req.SubCampaignId)
	if campaignCfg == nil {
		return nil, err, int32(cmd.ErrorCode_NotFoundConfig)
	}
	// 验证体力
	enough := GetConsumeMgr(h.actor).CheckEnough(common.ITEM_ID_STAMINA_1004, campaignCfg.StaminaCost)
	if !enough {
		return nil, err, int32(cmd.ErrorCode_StaminaValueNotEnough)
	}

	currCampaign := h.getCurrCampaign()
	if currCampaign == nil || currCampaign.SubCampaignId != req.SubCampaignId {
		return nil, fmt.Errorf("数据不存在, reqSubCampaignId=%d", req.SubCampaignId), int32(cmd.ErrorCode_CampaignNotExist)
	}

	currCampaign.BattleId = battleId
	currCampaign.BattleRandomSeed = rseed

	err = h.SaveDB()
	if err != nil {
		h.Warnf("玩家%s, 开始战斗 报错, %v", h.actor.ID(), err.Error())
	}

	rsp := &cmd.LS2C_CampaignBattleStartRes{
		BattleId:         battleId,
		BattleRandomSeed: rseed,
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampaignBattleStart{
	//		CustomHeadInfo:   lilith.BuildCustomHeadInfo(lilith.LogType_Campaign_start_battle, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		BattleId:         rsp.BattleId,
	//		BattleRandomSeed: rsp.BattleRandomSeed,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampaignBattleStart{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			BattleId:          rsp.BattleId,
			BattleRandomSeed:  rsp.BattleRandomSeed,
		}
		taptap.WriteDataLog(taptap.LogType_Campaign_start_battle, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
	return rsp, nil, int32(cmd.ErrorCode_Success)
}

func (h *CampaignHandler) BattleEndReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_CampaignBattleEndReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	endData, cliCommonData, err, errCode := h.handleBattleEnd(in.UserId, &req)
	if err != nil {
		return nil, err, errCode
	}

	rsp := &cmd.LS2C_CampaignBattleEndRes{
		EndData: endData,
	}
	if cliCommonData != nil {
		rsp.CommonData = cliCommonData.Data
	}

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

func (h *CampaignHandler) GameEndReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_PlayerCampaignDriveSettleReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	endData, cliCommonData, err, errCode := h.handleBattleEnd(in.UserId, &req)
	if err != nil {
		return nil, err, errCode
	}

	rsp := &cmd.LS2C_PlayerCampaignDriveSettleRes{
		EndData: endData,
	}
	if cliCommonData != nil {
		rsp.CommonData = cliCommonData.Data
	}

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

// BattleEndReq 游戏关卡结算请求
// 存在多种情况
// 1. 副本正常结束，成功获取奖励，扣除门票
// 2. 副本超时或失败结束，不扣除门票
// 每次游戏关卡结算通知副本更新状态，服务于门票扣除，
// 副本过程中跨日不扣除门票
func (h *CampaignHandler) handleBattleEnd(uid string, req proto.Message) (*cmd.CampaignEndData, *clidto.Comdata, error, int32) {
	var (
		err error

		reqSubCampaignId   int32               = 0
		reqBattleResult                        = cmd.BattleResult_BattleResult_None
		reqBattleScore     int32               = 0
		reqCostFoods                           = make([]*cmd.KeyValueItem, 0)
		reqBattleFrameData []*cmd.FrameCommand = nil
		reqVersionData     *cmd.CheckBattleVersionData

		tempRewards     = make(map[int32]int32)
		tempOnceRewards = make(map[int32]int32)

		reason common.ChangeReason
	)

	switch req.(type) {
	case *cmd.C2LS_CampaignBattleEndReq:
		battleEndReq := req.(*cmd.C2LS_CampaignBattleEndReq)

		reqSubCampaignId = battleEndReq.SubCampaignId
		reqBattleResult = battleEndReq.BattleResult
		reqBattleScore = battleEndReq.BattleScore
		reqCostFoods = battleEndReq.CostFoods
		reqBattleFrameData = battleEndReq.BattleFrameData
		reqVersionData = battleEndReq.VersionData

	case *cmd.C2LS_PlayerCampaignDriveSettleReq:
		gameEndReq := req.(*cmd.C2LS_PlayerCampaignDriveSettleReq)
		reqSubCampaignId = gameEndReq.SubCampaignId
		reqBattleScore = gameEndReq.Score

	default:
		h.Debugf("未支持的类型, %+v", req)
	}

	rsp := &cmd.CampaignEndData{
		DropChange:     &cmd.DropChange{},
		OnceDropChange: &cmd.DropChange{},
		BattleScore:    reqBattleScore, // 得分
		BattleResult:   reqBattleResult,
	}

	currCampaign := h.getCurrCampaign()
	if currCampaign == nil || currCampaign.SubCampaignId != reqSubCampaignId {
		return rsp, nil, fmt.Errorf("数据不存在, reqSubCampaignId=%d", reqSubCampaignId), int32(cmd.ErrorCode_CampaignNotExist)
	}

	campaignCfg := excel.GetCampaignMgr().GetById(currCampaign.SubCampaignId)
	if campaignCfg == nil {
		return rsp, nil, fmt.Errorf("没有对应的配置, reqSubCampaignId=%d", reqSubCampaignId), int32(cmd.ErrorCode_NotFoundConfig)
	}

	if reqBattleResult == cmd.BattleResult_BattleResult_Loser {
		// 战斗输了, do nothing...
		threading.RunSafe(func() {
			e := &taptap.CampaignBattleEnd{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				CampaignId:        currCampaign.CampaignId,
				SubCampaignId:     currCampaign.SubCampaignId,
				BattleScore:       rsp.BattleScore,
				BattleResult:      int64(rsp.BattleResult),
			}
			taptap.WriteDataLog(taptap.LogType_Campaign_end_battle, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})

		return rsp, nil, nil, int32(cmd.ErrorCode_Success)
	}

	campaignType := common.CAMPAIGN_TYPE(campaignCfg.CampaignType)
	// 战斗校验
	switch campaignType {
	case common.CAMPAIGN_TYPE_97, common.CAMPAIGN_TYPE_98, common.CAMPAIGN_TYPE_100:
		if reqBattleResult == cmd.BattleResult_BattleResult_Winer {
			// 战斗校验
			team := &cmd.GeneralCampaignTeam{}
			if len(currCampaign.Teams) > 0 {
				team = currCampaign.Teams[0]
			}
			selfBattleTeam := h.actor.BattleHandler.buildCampaignCards(team, campaignType)
			checkBattle, err, errCode := h.actor.BattleHandler.CheckBattle(
				currCampaign.BattleId, currCampaign.BattleRandomSeed, reqBattleResult,
				selfBattleTeam,
				campaignCfg.EventId,
				reqBattleFrameData, reqVersionData)
			if err != nil {
				return nil, nil, err, int32(errCode)
			}

			if checkBattle != nil && (checkBattle.CheckBattleResult == cmd.CheckBattleResult_CBR_fail || checkBattle.BattleResult != reqBattleResult) {
				return nil, nil, errors.New("校验失败"), int32(cmd.ErrorCode_CheckBattle_fail)
			}
		}

	default:
		h.Debugf("无需战斗校验, campaignType=%d", campaignType)
	}

	// 根据分数判断是否胜利
	//campaignType := common.CAMPAIGN_TYPE(campaignCfg.CampaignType)
	switch campaignType {
	case common.CAMPAIGN_TYPE_97, common.CAMPAIGN_TYPE_98:
		if campaignType == common.CAMPAIGN_TYPE_97 {
			reason = common.CR_Campaign_battle_97
		} else if campaignType == common.CAMPAIGN_TYPE_98 {
			reason = common.CR_Campaign_battle_98
		}

	case common.CAMPAIGN_TYPE_99, common.CAMPAIGN_TYPE_100: // 开车
		if pass, rewardByScoreMap := getRewardsByScore(currCampaign.CampaignId, reqSubCampaignId, reqBattleScore); pass {
			reqBattleResult = cmd.BattleResult_BattleResult_Winer
			myUtils.MergeItems(tempRewards, rewardByScoreMap) // 分数对应的奖励

		} else {
			reqBattleResult = cmd.BattleResult_BattleResult_Loser
		}

		if campaignType == common.CAMPAIGN_TYPE_99 {
			reason = common.CR_Campaign_battle_99
		} else if campaignType == common.CAMPAIGN_TYPE_100 {
			reason = common.CR_Campaign_battle_100
		}

	default:
		return rsp, nil, fmt.Errorf("不存在的关卡类型, %d", campaignCfg.CampaignType), int32(cmd.ErrorCode_NotFoundConfig)
	}

	costFoods := make(map[int32]int32)
	for _, food := range reqCostFoods {
		costFoods[food.Key] = food.Value
	}

	//myUtils.MergeItems(tempRewards, campaignCfg.RewardBase)                                               // 基础奖励
	//myUtils.MergeItems(tempRewards, map[int32]int32{common.ITEM_ID_ROLE_EXP_1001: campaignCfg.PlayerExp}) // 玩家经验
	//// 随机奖励
	//itemRewards := datahelper.GetRewardsByDropId(campaignCfg.RewardRandom)
	//for _, each := range itemRewards {
	//	myUtils.MergeItems(tempRewards, map[int32]int32{each.ItemId: each.Num})
	//}

	if reqBattleResult == cmd.BattleResult_BattleResult_Winer {
		// 消耗食物(胜利才会消耗)
		if !GetConsumeMgr(h.actor).CheckMapEnough(costFoods) {
			return rsp, nil, err, int32(cmd.ErrorCode_FoodNotEnough)

		}
		err = GetConsumeMgr(h.actor).ConsumeList(costFoods, h.actor.comData, reason)
		if err != nil {
			return rsp, nil, err, int32(cmd.ErrorCode_FoodNotEnough)
		}

		// 扣除体力
		costId := int32(common.ITEM_ID_STAMINA_1004)
		costNum := campaignCfg.StaminaCost
		if !GetConsumeMgr(h.actor).CheckEnough(costId, costNum) {
			return rsp, nil, fmt.Errorf("体力不足, 玩家id:%s, 当前%d, 需要%d", h.actor.ID(), h.actor.PlayerLevelHandler.GetPlayerStamina().Value, costNum), int32(cmd.ErrorCode_StaminaValueNotEnough)
		}
		err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{costId: costNum}, h.actor.comData, reason)
		if err != nil {
			return rsp, nil, err, int32(cmd.ErrorCode_StaminaValueNotEnough)
		}

		// 基础奖励
		_tempRewards := datahelper.GetCampaignBaseRewards(currCampaign.SubCampaignId)
		h.Debugf("特殊关卡基础奖励: CampaignType=%d, baseReward:%v", campaignCfg.CampaignType, _tempRewards)
		myUtils.MergeItems(tempRewards, _tempRewards)

		// 结算成功，发放奖励、更新数据
		switch campaignType {
		case common.CAMPAIGN_TYPE_99, common.CAMPAIGN_TYPE_100: // 开车	// 金币
			incr := getIncrCount(currCampaign.StartTimestamp)
			dailyCampaign := h.getDailyCampaign(currCampaign.CampaignId)
			dailyCampaign.CompleteCount += incr // 累计完成次数

			firstPass := h.IsDailyCampaignFirstPass(currCampaign.CampaignId, currCampaign.SubCampaignId)
			if firstPass {
				// 保存完成的关卡id
				dailyCampaign.CompletedStage = append(dailyCampaign.CompletedStage, currCampaign.SubCampaignId)

				// 首通
				myUtils.MergeItems(tempOnceRewards, campaignCfg.RewardOnce)
			}

		case common.CAMPAIGN_TYPE_97, common.CAMPAIGN_TYPE_98: // 日替
			/*incr := getIncrCount(currCampaign.StartTimestamp)
			dbCampaign := h.getCampaignInfo().ServerGeneralCampaign
			if err = incrResCompleteCount(dbCampaign, campaignCfg.CampaignType, incr); err != nil {
				return rsp, nil, err, int32(cmd.ErrorCode_InternalError)
			}*/

			firstPass := h.IsResCampaignFirstPass(currCampaign.CampaignId, currCampaign.SubCampaignId)
			if firstPass {
				// 保存完成的关卡id
				resCampaign := h.getResourceCampaign(currCampaign.CampaignId)
				resCampaign.CompletedStage = append(resCampaign.CompletedStage, currCampaign.SubCampaignId)

				// 首通奖励
				myUtils.MergeItems(tempOnceRewards, campaignCfg.RewardOnce)
			}
		default:
			h.Errorf("campaignType error BattleEndReq, %v", currCampaign)
		}

		// 上阵阵容(奖励卡牌经验)
		cardIds := make([]int32, 0)
		for _, team := range currCampaign.Teams {
			for _, cardId := range team.Cards {
				cardIds = append(cardIds, cardId)
			}
		}

		// 下发首通奖励
		h.Debugf("日替首通奖励:%v", tempOnceRewards)
		onceDropChange, err := GetDropMgr(h.actor).DropList2(tempOnceRewards, true, cardIds, h.actor.comData, reason)
		if err != nil {
			return rsp, nil, err, 0
		}
		rsp.OnceDropChange = onceDropChange

		// 下发常规奖励
		h.Debugf("日替奖励:%v", tempRewards)
		dropChange, err := GetDropMgr(h.actor).DropList2(tempRewards, true, cardIds, h.actor.comData, reason)
		if err != nil {
			return rsp, nil, err, 0
		}
		rsp.DropChange = dropChange

		// 更新历史最高分
		h.updateDriveMaxScore(currCampaign.CampaignId, currCampaign.SubCampaignId, reqBattleScore)
	}

	// 清空当前攻打副本数据
	h.getCampaignInfo().CurCampaign = nil
	err = h.SaveDB()
	if err != nil {
		h.Warnf("玩家%s, 日替 battleEnd 报错, %v", h.actor.ID(), err.Error())
	}

	// 任务事件发布
	if campaignCfg.CampaignType == int32(common.CAMPAIGN_TYPE_99) {
		errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_CAMPAIGN_CAR_SETTLE, []int32{TASK_TYPE_41}, nil))
		if errx != nil {
			h.Error(errx)
		}
	}

	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_CAMPAIGN_LEVEL, []int32{TASK_TYPE_507, TASK_TYPE_508}, map[string]interface{}{
		"type": campaignCfg.CampaignType,
	}))
	if errx != nil {
		h.Error(errx)
	}

	// 通知副本最新状态
	h.actor.comData.Data.Campaign = h.buildClientCampaignData()
	//rsp.CommonData = retCommonData.Data

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampaignBattleEnd{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Campaign_end_battle, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		CampaignId:     currCampaign.CampaignId,
	//		SubCampaignId:  currCampaign.SubCampaignId,
	//		BattleScore:    rsp.BattleScore,
	//		BattleResult:   int64(rsp.BattleResult),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampaignBattleEnd{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CampaignId:        currCampaign.CampaignId,
			SubCampaignId:     currCampaign.SubCampaignId,
			BattleScore:       rsp.BattleScore,
			BattleResult:      int64(reqBattleResult),
		}
		taptap.WriteDataLog(taptap.LogType_Campaign_end_battle, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, h.actor.comData, nil, int32(cmd.ErrorCode_Success)
}

func (h *CampaignHandler) checkTeam(subCampaignId int32, teams []*cmd.GeneralCampaignTeam) (error, cmd.ErrorCode) {
	var (
		team1 = make([]int32, 0) // 队伍1 - 参战
		team2 = make([]int32, 0) // 队伍2 - 观战
	)

	for _, each := range teams {
		switch each.TeamNumber {
		case cmd.GeneralCampaignTeamType_GeneralCampaignTeamType_Num1:
			for _, card := range each.Cards {
				if card <= 0 {
					continue // 无人在此(编号0的不需要判断是否重复)
				}
				team1 = append(team1, card)
			}

		case cmd.GeneralCampaignTeamType_GeneralCampaignTeamType_Num2:
			for _, card := range each.Cards {
				if card <= 0 {
					continue // 无人在此(编号0的不需要判断是否重复)
				}
				team2 = append(team2, card)
			}

		default:
			return fmt.Errorf("不存在的队伍类型, %d", each.TeamNumber), cmd.ErrorCode_InvalidParam
		}
	}

	if myUtils.WithinArray(team1) || myUtils.WithinArray(team2) || myUtils.WithinArray2(team1, team2) {
		return fmt.Errorf("有重复的卡牌, team1:%v, team2:%v", team1, team2), cmd.ErrorCode_CampaignTeamRoleDuplicate
	}

	campaignCfg := excel.GetCampaignMgr().GetById(subCampaignId)
	if campaignCfg == nil {
		return fmt.Errorf("配置未找到"), cmd.ErrorCode_NotFoundConfig
	}

	switch common.CAMPAIGN_TYPE(campaignCfg.CampaignType) {
	case common.CAMPAIGN_TYPE_97, common.CAMPAIGN_TYPE_98, common.CAMPAIGN_TYPE_100:
		if (len(team1) < 1 || len(team1) > 4) || len(team2) > 0 { // 日替、金币副本, 参战人数[1, 4]; 观战人数[0, 0]
			return fmt.Errorf("上阵不合法"), cmd.ErrorCode_CampaignTeamParamError
		}
	case common.CAMPAIGN_TYPE_99:
		if (len(team1) < 1 || len(team1) > 4) || (len(team2) < 0 || len(team2) > 2) { // 开车游戏, 参战人数[1, 4]; 观战人数[0, 2]
			return fmt.Errorf("上阵不合法"), cmd.ErrorCode_CampaignTeamParamError
		}
	}

	// 检测卡牌是否存在
	if team1Cards := h.actor.CardHandler.GetCards(team1); len(team1Cards) != len(team1) {
		return nil, cmd.ErrorCode_CardNotExist
	}
	if team2Cards := h.actor.CardHandler.GetCards(team2); len(team2Cards) != len(team2) {
		return nil, cmd.ErrorCode_CardNotExist
	}

	return nil, cmd.ErrorCode_Success
}

func getIncrCount(timestampSec int64) int32 {
	t := time.Unix(timestampSec, 0)
	now := time.Now()
	var incr int32
	if common.IsSameDayByOffset(t, now, common.GAME_DAILY_REFRESH_HOUR) {
		incr++
	}

	return incr
}

// 获取日替本开放列表
func getCampaignOpenedList(weekDay time.Weekday) ([]int32, []int32) {
	var (
		resCampaignIds = make([]int32, 0)
		subCampaignIds = make([]int32, 0)
	)

	excel.GetResourcecampaignMgr().Foreach(func(cfg *excel.ResourcecampaignCfg) bool {
		for _, v := range cfg.Unlockdays {
			if v == int32(weekDay) {
				resCampaignIds = append(resCampaignIds, cfg.Id)
				subCampaignIds = append(subCampaignIds, cfg.Includecampaign...)
			}
		}
		return true
	}, true)
	return resCampaignIds, subCampaignIds
}

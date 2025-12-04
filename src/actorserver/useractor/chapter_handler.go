package useractor

import (
	"context"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"

	"github.com/pkg/errors"

	"fmt"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/chapter"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"
	"google.golang.org/protobuf/proto"
)

type ChapterHandler struct {
	*UABaseHandler
}

func NewChapterHandler(actor *UserActor) *ChapterHandler {
	h := &ChapterHandler{UABaseHandler: NewUABaseHandler(actor, "ChapterHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_EnterLevelReq), h.EnterLevelReq)           // 进入大地图
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_LevelBattleSettlementReq), h.ExitLevelReq) // 退出大地图
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_StartBattleEventReq), h.BattleStartReq)    // 战斗开始
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_LevelBattleEventReq), h.BattleEndReq)      // 战斗结束事件
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_LevelEventReq), h.LevelEventReq)           // 地图事件(伐木、荆棘等的事件)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_SelectScenePathReq), h.SelectScenePathReq) // 选择路径
	//actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_UnlockedPointReq), h.UnlockedPointReq)                   // 节点信息
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_DiscoveryUnlockedPointReq), h.DiscoveryUnlockedPointReq) // 解锁节点
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_UpdateLevelTroopReq), h.UpdateLevelTroopReq)             // 关卡中更新队伍
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_BackToBCReq), h.BackToBCReq)                             // 返回大本营
	//actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_SaveWeatherReq), h.SaveWeatherReq)                       // 保存天气
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_DiscoverMonsterReq), h.DiscoverMonsterReq) // 发现怪物
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CardEatFoodReq), h.CardEatFoodReq)         // 使用食物回血
	return h
}

// Init 初始化模块数据
func (h *ChapterHandler) Init() error {
	// 初始化
	h.actor.Data.LevelsData = &cmd.LS2DB_LevelInfos{
		Createtime: time.Now().Unix(),
		LevelInfos: make(map[int32]*cmd.LS2DB_LevelInfo, 0),
		PLevelSummary: &cmd.PServerLevelSummary{
			MonsterTicketInfoMap: make(map[int32]*cmd.LevelMonsterTicketInfo, 0),
			LevelSummaryMap:      make(map[int32]*cmd.LevelSummary, 0),
		},
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debugf("init ChapterHandler data success. player: %s", h.actor.ID())
	return nil
}

func (h *ChapterHandler) EnterGame() error {
	levelsData := h.actor.GetLevelsData()

	// 初始化
	err := h.incrMonsterMaxTicketCount(levelsData.PLevelSummary.MonsterTicketInfoMap, h.actor.comData)
	if err != nil {
		h.Errorf(err.Error())
	}

	return nil
}

func (h *ChapterHandler) DailyRefresh() error {
	// 清除关卡中的战斗计数
	h.cleanBattleCount()

	h.dailyRefreshMonsterTicket()

	err := h.SaveDB()
	if err != nil {
		h.Errorf("DailyRefresh 报错, err:%+v", err)
	}

	return nil
}

func (h *ChapterHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.LS2DB_LevelInfos); ok {
		h.actor.Data.LevelsData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *ChapterHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserLevelInfo(h.actor.ID()), h.actor.Data.LevelsData
}

// EnterLevelReq 进入大地图战斗
func (h *ChapterHandler) EnterLevelReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err        error
		errCode    cmd.ErrorCode
		levelsData *cmd.LS2DB_LevelInfos
		levelData  *cmd.LS2DB_LevelInfo
		dropChange = &cmd.DropChange{}
	)

	var req cmd.C2LS_EnterLevelReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	levelCfg := data.GetLevelMgr().GetById(req.LevelId)
	if levelCfg == nil {
		return nil,
			errors.New(fmt.Sprintf("参数错误, 没有找到配置, levelId=%d", req.LevelId)),
			int32(cmd.ErrorCode_NotFoundConfig)
	}

	if levelCfg.LevelType == int32(common.CHAPTER_LEVEL_TYPE_MAIN) {
		// 主线关卡解锁条件
		err, errCode = h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1001)
		if err != nil {
			return nil, err, int32(errCode)
		}
	} else if levelCfg.LevelType == int32(common.CHAPTER_LEVEL_TYPE_SUB) {
		// fixme 在讨论 副本关卡解锁条件
		//err, errCode = h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1005)
		//if err != nil {
		//	return nil, err, int32(errCode)
		//}
	}

	// 输入静态值，为了同步后面函数getPlayerLevelData（） → GetCardIds（）
	errCode = h.actor.TroopHandler.CheckTroopTypAndId(int32(cmd.CardTroopType_CardTroopType_Normal), int32(req.TroopId))
	if errCode != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("err input param, troopId:%d", req.TroopId), int32(errCode)
	}

	// 地图信息
	levelsData = h.actor.GetLevelsData()
	if err != nil {
		h.Errorf("load user strength pool failed. err:", err)
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	dailyMaxBattleCount := data.GetConfigMgr().GetCfg().EXPLOREDAILYLIMIT
	if int32(levelsData.DailyTotalBattleCount) >= dailyMaxBattleCount {
		// 超过当日最大战斗次数
		h.Errorf("%s, 关卡 超过当日最大战斗次数.", h.actor.ID())
		return nil,
			fmt.Errorf("BattleStartReq 当日达到最大战斗次数, roleId:%d, dailyMaxBattleCount=%d, battleCount:%d",
				h.actor.GetUserData().Common.RoleId, dailyMaxBattleCount, levelsData.DailyTotalBattleCount),
			int32(cmd.ErrorCode_Chapter_event_over_battle_count)
	}

	// 判断自己关卡限制
	err = h.checkSelfLevelCondition(req.LevelId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_Chapter_level_is_once)
	}

	// 检查解锁条件
	if err, errCode = h.checkUnlockCondition(levelsData.CurrLevelId, levelCfg.UnlockCondition); err != nil {
		return nil, err, int32(errCode)
	}

	if req.EnterLevelType == cmd.EnterLevelType_EnterLevelType_Reenter { // 断线重连
		//if true {
		var ok bool
		if levelData, ok = h.GetCurrLevelData(); !ok {
			return nil, errors.New("副本已不存在"), int32(cmd.ErrorCode_Chapter_sub_level_not_exist)
		} else {
			if levelData.LevelId != req.LevelId {
				return nil, errors.New("请求对应的副本已不存在"), int32(cmd.ErrorCode_Chapter_sub_level_not_exist)
			}

			if levelCfg.LevelType != int32(common.CHAPTER_LEVEL_TYPE_SUB) {
				return nil, errors.New("副本才有断线重连"), int32(cmd.ErrorCode_Chapter_sub_level_not_exist)
			}

			// 断线重连-更新地图事件
			h.TryIncrMappointEvents(levelData.LevelId, levelData.CurrNiwaId)
		}
	} else { // 正常进入关卡

		// 在副本中, 再次进入只能走断线重连, (结算前, 关卡、副本都不可进)
		if h.IsInSubLevel() {
			return nil,
				fmt.Errorf("EnterLevelReq 上次副本还未结算, 不能进入其他关卡(副本), roleId:%d, 当前副本:%d, 请求进入的副本:%d",
					h.actor.GetUserData().Common.RoleId, levelsData.CurrLevelId, req.LevelId),
				int32(cmd.ErrorCode_Chapter_sub_level_not_finish)
		}

		// 清除子关卡信息
		h.cleanAllSubLevelInfo()

		if _, ok := h.GetLevelData(req.LevelId); !ok {
			// 没有数据, 扣除体力(预扣)
			if NeedWithholdStamina(common.LEVEL_TYPE(levelCfg.LevelType)) {
				if !h.actor.PlayerLevelHandler.CheckStaminaEnough(levelCfg.StaminaCost) {
					err = errors.New(fmt.Sprintf("进入副本体力不满足要求, 至少需要体力:%d", levelCfg.StaminaCost))
					return nil, err, int32(cmd.ErrorCode_StaminaValueNotEnough)
				}
			}
		}

		// 进入关卡
		if battleLevel, _itemRewards, err, errCode := h.enterLevel(req.LevelId, req.TroopId); err != nil {
			return nil, err, int32(errCode)
		} else {
			levelData = battleLevel.FormatStage2DB()
			mergeDropChange(dropChange, _itemRewards) // 默认开启的点位奖励, 先不给客户端
		}

		// 进入地图
		_, err, errCode = h.enterNiwa(req.LevelId, req.NiwaId)
		if err != nil {
			return nil, err, int32(errCode)
		}

		// 血量重置为满血
		h.UpdateCardHpFull(levelData)
	}

	playerLevelCard := h.getOrInitBattleCards(levelData, int32(req.TroopId))
	if nil == playerLevelCard {
		return nil, fmt.Errorf("队伍中没有活着的卡牌, troopId:%d", req.TroopId), int32(cmd.ErrorCode_Chapter_no_live_in_troop)
	}
	levelData.PlayerLevelData = playerLevelCard

	//  持久化
	err = h.SaveDB()
	if err != nil {
		h.Errorf("EnterLevelReq SaveChapterData2DB 报错, err:%+v", err)
	}

	// 处理剧情任务刷新
	if err = h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_LEVEL_ENTER, []int32{}, nil)); err != nil {
		logger.Error(err)
	}

	// 地图信息
	var (
		niwaData *cmd.BattleMapInfo
		ok       bool
	)

	// 获取关卡当前所在的箱庭信息
	niwaData, ok = h.GetCurrNiwaData()
	if !ok {
		return nil, fmt.Errorf("没有地图数据, troopId:%d", req.TroopId), int32(cmd.ErrorCode_InternalError)
	}

	h.actor.comData.Data.LevelSummary = h.Dto2PClientLevelSummary(levelData.LevelId, 0)

	// 下发货币数据
	h.actor.comData.Data.Currency = h.actor.CurrencyHandler.buildCurrencyList()

	rsp := &cmd.LS2C_EnterLevelRes{
		LevelId:         levelData.LevelId,
		PlayerLevelData: levelData.PlayerLevelData,
		MapInfo:         niwaData,
		DropChange:      dropChange,
		CommonData:      h.actor.comData.FixDownComData(),
		BigLevelData:    levelData.BigLevelData,
	}
	// 下发数据
	h.Debugf("enter level rsp<<<===, %+v", rsp)

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelEnter{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Level_enter, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		LevelId:        req.LevelId, // 卡牌id
	//		TroopId:        req.TroopId, // 升级前等级
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelEnter{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			LevelId:           req.LevelId, // 卡牌id
			TroopId:           req.TroopId, // 升级前等级
		}
		taptap.WriteDataLog(taptap.LogType_Level_enter, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
	h.Debugf("进去地图返回数据commonData:", rsp.GetCommonData())
	return rsp, nil, 0
}

// ExitLevelReq 退出大地图战斗
func (h *ChapterHandler) ExitLevelReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err     error
		errCode cmd.ErrorCode
		uid     string
		//
		//levelsData *cmd.LS2DB_LevelInfos
		//levelData  *cmd.LS2DB_LevelInfo
		//rsp = &cmd.LS2C_LevelBattleSettlementRes{}
		//nowSec     = time.Now().Unix()
		//
		//onceRewards    *cmd.LS2C_ChangeItemNtf      // 首次通关奖励 - 最终量
		//onceAddRewards = make([]*cmd.ItemReward, 0) // 首次通关奖励 - 增量
		//
		//baseRewards    *cmd.LS2C_ChangeItemNtf      // 通关基础奖励 - 最终量
		//baseAddRewards = make([]*cmd.ItemReward, 0) // 通关基础奖励 - 增量
		rsp = &cmd.LS2C_LevelBattleSettlementRes{
			OnceDropChange: &cmd.DropChange{},
			DropChange:     &cmd.DropChange{},
		}
	)

	var req cmd.C2LS_LevelBattleSettlementReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 地图信息
	levelsData := h.actor.GetLevelsData()
	if err != nil {
		h.Errorf("load user strength pool failed. err:", err)
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	if err, errCode = h.doExitLevel(uid, levelsData.CurrLevelId, req.EndId, req.BattleResult, rsp, false); err != nil {
		return nil, err, int32(errCode)
	}

	return rsp, nil, 0
}

//// 检查大关卡完成条件
//func (h *ChapterHandler) checkFinishMainLevelCondition(levelsData *cmd.LS2DB_LevelInfos, levelData *cmd.LS2DB_LevelInfo) (error, cmd.ErrorCode) {
//	if h.IsInSubLevel() {
//		// 只有大关卡才会判断指定副本是否完成
//		return nil, cmd.ErrorCode_Success
//	}
//
//	if summary, ok := levelsData.PLevelSummary.LevelSummaryMap[levelData.LevelId]; ok {
//		if summary.LevelSimpleInfo.HistoryHadPassed == cmd.HistoryHadPassed_PLevelStatus_Passed {
//			// 历史完成过就可以
//			return nil, cmd.ErrorCode_Success
//		} else {
//			// 历史中还未完成过, 验证指定任务是否完成
//			levelCfg := data.GetLevelMgr().GetById(levelData.LevelId)
//			if levelCfg.PassCondition > 0 {
//				//if passCondSimpleLeveData, ok := levelsData.LevelSimpleInfo[levelCfg.PassCondition]; ok &&
//				//	passCondSimpleLeveData.HistoryHadPassed == cmd.HistoryHadPassed_PLevelStatus_Passed { // 指定关卡历史中有完成过
//				//	// 历史完成过指定副本
//				//	return nil, cmd.ErrorCode_Success
//				//} else {
//				//	return fmt.Errorf("关卡%d, 指定副本未完成, levelCfg.PassCondition=%d", levelData.LevelId, levelCfg.PassCondition), cmd.ErrorCode_GotSelfDefinedError
//				//}
//				if !h.actor.QuestHandler.checkQuestFinish(levelCfg.PassCondition) {
//					return fmt.Errorf("关卡%d, 指定任务未完成, levelCfg.PassCondition=%d", levelData.LevelId, levelCfg.PassCondition), cmd.ErrorCode_GotSelfDefinedError
//				}
//			}
//		}
//	}
//
//	return nil, cmd.ErrorCode_Success
//}

// BattleStartReq 战斗开始
func (h *ChapterHandler) BattleStartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err        error
		errCode    cmd.ErrorCode
		levelsData *cmd.LS2DB_LevelInfos
		levelData  *cmd.LS2DB_LevelInfo
		cliComData *clidto.Comdata
	)

	var req cmd.C2LS_StartBattleEventReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 地图信息
	levelsData = h.actor.GetLevelsData()

	if each, ok := h.GetCurrLevelData(); ok {
		levelData = each
	} else {
		return nil, fmt.Errorf("还未开始战斗, 当前所在的关卡id:%d", levelsData.CurrLevelId), int32(cmd.ErrorCode_InvalidParam)
	}

	eventCfg := data.GetMappointEventMgr().GetById(req.EventId)
	if eventCfg == nil {
		return nil, fmt.Errorf("无效的事件id, roleId=%d, eventId=%d",
			h.actor.GetUserData().Common.RoleId, req.EventId), int32(cmd.ErrorCode_ParamError)
	}
	battleEventCfg := data.GetBattleEventMgr().GetById(eventCfg.GetBattlestageId())
	if battleEventCfg == nil {
		return nil, fmt.Errorf("无效的事件id, roleId=%d, eventId=%d",
			h.actor.GetUserData().Common.RoleId, req.EventId), int32(cmd.ErrorCode_ParamError)
	}
	//if myUtils.GetInt32AtBit(battleEventCfg.IsVerify, common.CheckBattleBitPos_DO_hpFull) == 1 {
	//	h.UpdateCardHpFull(levelData)
	//	h.Debugf("玩家%v, 事件id=%d, 进入战斗前恢复满血", h.actor.ID(), req.EventId)
	//}

	// 发现怪物
	cliComData, err, errCode = h.doDiscoverMonster(req.EventId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	battleId := uint64(utils.GenIntUUID())
	rseed := utils.GenIntUUID()

	// 保存随机种子
	levelsData.BattleId = battleId
	levelsData.BattleRandomSeed = rseed

	levelData.BattleLostPunish = 0 // 取消战斗失败惩罚标记

	// 持久化
	err = h.SaveDB()
	if err != nil {
		h.Errorf("BattleStartReq SaveChapterData2DB 报错, err:%+v", err)
	}

	rsp := &cmd.LS2C_StartBattleEventRes{
		BattleId:         battleId,
		BattleRandomSeed: rseed,
	}
	if cliComData != nil {
		rsp.CommonData = cliComData.Data
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelBattleStart{
	//		CustomHeadInfo:   lilith.BuildCustomHeadInfo(lilith.LogType_Level_start_battle, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		BattleId:         rsp.BattleId,
	//		BattleRandomSeed: rsp.BattleRandomSeed,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelBattleStart{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			BattleId:          rsp.BattleId,
			BattleRandomSeed:  rsp.BattleRandomSeed,
		}
		taptap.WriteDataLog(taptap.LogType_Level_start_battle, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

//// 检查核心事件是否全部完成
//func checkAllMainEventFinish(levelData *cmd.LS2DB_LevelInfo) (error, cmd.ErrorCode) {
//	for _, mapInfo := range levelData.GetMapInfos() {
//		err, errorCode := checkNiwaMainEventFinish(mapInfo)
//		if err != nil {
//			return err, errorCode
//		}
//	}
//	return nil, cmd.ErrorCode_Success
//}

//func checkNiwaMainEventFinish(mapInfo *cmd.BattleMapInfo) (error, cmd.ErrorCode) {
//
//	for _, event := range mapInfo.MappointEvents {
//		eventCfg := data.GetMappointEventMgr().GetById(event.EventId)
//		if eventCfg.IsmainEvent == 1 { // 是核心事件
//			hadFound := false
//			for _, finishedEventId := range mapInfo.FinishedEventIds {
//				if finishedEventId == event.EventId {
//					hadFound = true
//					break
//				}
//			}
//
//			if !hadFound {
//				return errors.New(fmt.Sprintf("还有核心事件未完成, niwaId:%d, 找到未完成的eventId=%d", mapInfo.NiwaId, event.EventId)),
//					cmd.ErrorCode_Chapter_main_event_undone
//			}
//		}
//	}
//	return nil, cmd.ErrorCode_Success
//}

// BattleEndReq 战斗事件
func (h *ChapterHandler) BattleEndReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		errCode   cmd.ErrorCode
		uid       = in.UserId
		levelData *cmd.LS2DB_LevelInfo
	)

	var req cmd.C2LS_LevelBattleEventReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 地图信息
	levelsData := h.actor.GetLevelsData()

	if each, ok := h.GetCurrLevelData(); ok {
		levelData = each
	} else {
		return nil, fmt.Errorf("没有对应的关卡数据, 当前所在的关卡id:%d", levelsData.CurrLevelId), int32(cmd.ErrorCode_InvalidParam)
	}

	eventData, err, errCode := h.handleEvent(uid, &req, h.actor.comData)
	if err != nil || errCode != cmd.ErrorCode_Success {
		h.Debugf("ChapterHandler.handleBattleEvent got err: %+v", err)
		return nil, err, int32(errCode)
	}

	h.actor.comData.Data.LevelSummary = h.Dto2PClientLevelSummary(levelData.LevelId, req.EventId)

	// 下发货币数据
	h.actor.comData.Data.Currency = h.actor.CurrencyHandler.buildCurrencyList()

	rsp := &cmd.LS2C_LevelBattleEventRes{
		EventData:  eventData,
		CommonData: h.actor.comData.FixDownComData(),
	}

	if eventData.EventResult == cmd.EventResult_EventResult_Success {
		rsp.BattleResult = cmd.BattleResult_BattleResult_Winer

		// 发布事件-战斗胜利才算
		eventCfg := data.GetMappointEventMgr().GetById(req.EventId)
		monster_ids := datahelper.GetMonsterIdsByBattleEventId(eventCfg.BattlestageId)
		errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_MONSTER_BATTLE, []int32{TASK_TYPE_1, TASK_TYPE_2, TASK_TYPE_3}, map[string]interface{}{
			"monster_ids": monster_ids,      // 怪物id列表
			"type":        h.IsInSubLevel(), // true=副本，false=大地图
			"card_id":     h.GetCardIds(h.actor.GetLevelsData().TroopId),
		}))
		if errx != nil {
			h.Error(errx)
		}
	} else {
		rsp.BattleResult = cmd.BattleResult_BattleResult_Loser
	}

	monster := make([]uint32, 0)
	for _, value := range req.Monster {
		monster = append(monster, value.Common.MonsterId)
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelBattleEnd{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Level_end_battle, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		NiwaId:         req.NiwaId,
	//		EventId:        req.EventId,
	//		QuestObjectId:  req.QuestObjectId,
	//		Monster:        lilith.ConvertList2Str(monster),
	//		BattleResult:   int64(req.BattleResult),
	//		BattleCards:    lilith.ConvertStruct2Str(rsp.EventData.PlayerLevelData.BattleCards), // 卡牌信息
	//		Foods:          lilith.ConvertList2Str(rsp.EventData.PlayerLevelData.Foods),         // 食物itemId列表
	//		CostFoods:      lilith.ConvertListStruct2Str(req.CostFoods),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelBattleEnd{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			NiwaId:            req.NiwaId,
			EventId:           req.EventId,
			QuestObjectId:     req.QuestObjectId,
			Monster:           taptap.ConvertList2Str(monster),
			BattleResult:      int64(req.BattleResult),
			BattleCards:       taptap.ConvertStruct2Str(rsp.EventData.PlayerLevelData.BattleCards), // 卡牌信息
			Foods:             taptap.ConvertList2Str(rsp.EventData.PlayerLevelData.Foods),         // 食物itemId列表
			CostFoods:         taptap.ConvertListStruct2Str(req.CostFoods),
		}
		taptap.WriteDataLog(taptap.LogType_Level_end_battle, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

// LevelEventReq 地图事件
func (h *ChapterHandler) LevelEventReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err     error
		errCode cmd.ErrorCode
		uid     = in.UserId
	)

	var req cmd.C2LS_LevelEventReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	eventData, err, errCode := h.handleEvent(uid, &req, h.actor.comData)
	if err != nil {
		h.Debugf("ChapterHandler.handleBattleEvent got err: %+v", err, req)
		return nil, err, int32(errCode)
	}

	rsp := &cmd.LS2C_LevelEventRes{
		EventData:  eventData,
		CommonData: h.actor.comData.FixDownComData(),
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelMapEvent{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Level_map_event, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		NiwaId:         req.NiwaId,
	//		EventId:        req.EventId,
	//		QuestObjectId:  req.QuestObjectId,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelMapEvent{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			NiwaId:            req.NiwaId,
			EventId:           req.EventId,
			QuestObjectId:     req.QuestObjectId,
		}
		taptap.WriteDataLog(taptap.LogType_Level_map_event, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, 0
}

// 选择路径
func (h *ChapterHandler) SelectScenePathReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		ok        bool
		levelData *cmd.LS2DB_LevelInfo
		rsp       = &cmd.LS2C_SelectScenePathRes{}
	)

	var req cmd.C2LS_SelectScenePathReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	levelsData := h.actor.GetLevelsData()
	levelData, ok = h.GetCurrLevelData()
	if !ok {
		return nil, fmt.Errorf("未找到当前关卡信息, 当前所在的关卡id:%d", levelsData.CurrLevelId), int32(cmd.ErrorCode_InvalidParam)
	}

	if err, errCode := checkNiwaPathValid(levelData.CurrNiwaId, req.PathId); err != nil {
		return nil, err, int32(errCode)
	}

	pathCfg := data.GetPathMgr().GetById(int32(req.PathId))
	if pathCfg == nil {
		return nil, fmt.Errorf("配置为空"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	if len(pathCfg.GetNiwagroup()) <= 0 {
		return rsp, nil, 0
	} else {
		randIdx, err := myUtils.RandomInt(0, len(pathCfg.GetNiwagroup()))
		if err != nil {
			return rsp, fmt.Errorf("随机路径报错, %+v", err), int32(cmd.ErrorCode_InternalError)
		}

		randNiwaId := pathCfg.GetNiwagroup()[randIdx]
		levelData.PathNiwaIds = append(levelData.PathNiwaIds, randNiwaId)

		rsp.NiwaId = randNiwaId
	}

	niwaData, err, errCode := h.enterNiwa(levelData.LevelId, rsp.NiwaId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	err = h.SaveDB()
	if err != nil {
		h.Errorf("EnterLevelReq SaveChapterData2DB 报错, err:%+v", err)
	}

	rsp.NiwaId = levelData.CurrNiwaId
	rsp.MapInfo = niwaData

	h.Infof("ChapterHandler.SelectScenePathReq 跳转地图下发数据: req.PathId=%d, resp=%+v, mapInfo=%+v",
		req.PathId, rsp, rsp.MapInfo)

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelChooseNiwaPath{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Level_choose_niwa_path, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		PathId:         req.PathId,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelChooseNiwaPath{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			PathId:            req.PathId,
		}
		taptap.WriteDataLog(taptap.LogType_Level_choose_niwa_path, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, 0
}

// CardEatFoodReq 食物使用
func (h *ChapterHandler) CardEatFoodReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.C2LS_CardEatFoodReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	var levelData *cmd.LS2DB_LevelInfo
	var ok bool
	if levelData, ok = h.GetCurrLevelData(); !ok {
		return nil, errors.New("副本已不存在"), int32(cmd.ErrorCode_Chapter_sub_level_not_exist)
	}

	consumeMgr := GetConsumeMgr(h.actor)
	costs := make(map[int32]int32, len(req.GetFoods()))
	foodIds := make([]int32, 0, len(req.GetFoods()))
	for _, item := range req.GetFoods() {
		// 判定道具是不是食物
		if !consumeMgr.CheckItemType(item.GetKey(), int32(cmd.ItemType_Food), 0) {
			return nil, fmt.Errorf("param err"), int32(cmd.ErrorCode_ParamError)
		}
		// 判断食物是否再编队中
		if !h.FoodIsExist(levelData, item.GetKey()) {
			h.Debugf("食物不在编队中:", item.GetKey())
			return nil, fmt.Errorf("param err"), int32(cmd.ErrorCode_ParamError)
		}
		costs[item.GetKey()] = item.GetValue()
		foodIds = append(foodIds, item.GetKey())
	}
	// 判断卡片是否再编队中
	if !h.CheckCardExist(levelData, req.GetCardId()) {
		h.Debugf("卡片不在编队中:", req.GetCardId())
		return nil, fmt.Errorf("param err"), int32(cmd.ErrorCode_ParamError)
	}
	// 判断卡片血量是否满 true hp<
	if !h.CheckCardFullHp(levelData, req.GetCardId()) {
		h.Debugf("卡片已经是最大血量", req.GetCardId())
		return nil, fmt.Errorf("param err"), int32(cmd.ErrorCode_ParamError)
	}

	// 材料检查
	if !consumeMgr.CheckMapEnough(costs) {
		return nil, fmt.Errorf("item check failed"), int32(cmd.ErrorCode_NotEnoughItem)
	}

	// 更新使用过的食物
	if !h.UpdateUseFoods(levelData, req.GetFoods()) {
		h.Debugf("使用食物超过最大限制：", req.GetFoods())
		return nil, err, int32(cmd.ErrorCode_Chapter_use_food_limit)
	}

	err = consumeMgr.ConsumeList(costs, h.actor.comData, common.CR_Card_Eat_Food)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	if code := h.HandleEatFood(levelData, uint32(req.GetCardId()), req.GetFoods()); code != cmd.ErrorCode_Success {
		return nil, nil, int32(code)
	}

	playerLevelCard := h.getOrInitBattleCards(levelData, req.GetTroopId())
	if nil == playerLevelCard {
		return nil, fmt.Errorf("队伍中没有活着的卡牌, troopId:%d", req.TroopId), int32(cmd.ErrorCode_Chapter_no_live_in_troop)
	}

	if h.SaveDB() != nil {
		h.Errorf("handleBattleEvent SaveChapterData2DB 报错, err:%+v", err)
	}

	// 消息返回
	rsp := &cmd.LS2C_CardEatFoodRes{
		CommonData:      h.actor.comData.FixDownComData(),
		Items:           h.actor.BagHandler.GetMulItemNum(foodIds),
		PlayerLevelData: playerLevelCard,
	}

	// 发布事件
	e := event.NewBasicEvent(TASK_EVENT_USE_FOOD, []int32{TASK_TYPE_111}, map[string]interface{}{})
	h.actor.eventManager.SyncPublish(e)

	h.Debugf("CardEatFoodReq handle:", in.UserId)
	return rsp, nil, 0
}

// 检查地图跳转路径是否合法
func checkNiwaPathValid(niwaId int32, reqPathId uint32) (error, cmd.ErrorCode) {
	// 验证客户端选择pathId的有效性
	hakoniwaCfg := data.GetHakoniwaMgr().GetById(niwaId)
	if hakoniwaCfg == nil {
		return fmt.Errorf("配置为空, niwaId=%d, reqPathId=%d", niwaId, reqPathId), cmd.ErrorCode_NotFoundConfig
	}

	choosePaths := hakoniwaCfg.GetPath()
	paramFound := false
	for _, each := range choosePaths {
		if each == int32(reqPathId) {
			paramFound = true
			break
		}
	}
	if !paramFound {
		return fmt.Errorf("无效的路径, niwaId=%d, reqPathId=%d", niwaId, reqPathId), cmd.ErrorCode_InvalidParam
	}

	return nil, cmd.ErrorCode_Success
}

func (h *ChapterHandler) DiscoveryUnlockedPointReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err             error
		ok              bool
		levelsData      *cmd.LS2DB_LevelInfos
		levelData       *cmd.LS2DB_LevelInfo
		simpleLevelData *cmd.LevelSummary

		rsp = &cmd.LS2C_DiscoveryUnlockedPointRes{
			DropChange: &cmd.DropChange{},
		}
	)

	var req cmd.C2LS_DiscoveryUnlockedPointReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	levelsData = h.actor.GetLevelsData()

	levelData, ok = h.GetCurrLevelData()
	if !ok {
		return nil,
			fmt.Errorf("没有当前关卡信息, levelId=%d, UnlockedPointId=%d", levelsData.CurrLevelId, req.UnlockedPointId),
			int32(cmd.ErrorCode_ParamError)
	}

	simpleLevelData, ok = levelsData.PLevelSummary.LevelSummaryMap[levelData.LevelId]
	if !ok {
		return nil,
			fmt.Errorf("关卡的摘要信息还未初始化, levelId=%d, UnlockedPointId=%d", levelsData.CurrLevelId, req.UnlockedPointId),
			int32(cmd.ErrorCode_ParamError)
	}

	mapunlockpointCfg := data.GetMapunlockpointMgr().GetById(req.UnlockedPointId)
	if itemRewards, err, errCode := h.doUnlockUnlockedPoint(
		levelsData.CurrLevelId, simpleLevelData.LevelSimpleInfo, []*data.MapunlockpointCfg{mapunlockpointCfg}, true); err != nil {

		return nil, err, int32(errCode)
	} else {
		rsp.DropChange.Items = append(rsp.DropChange.Items, itemRewards.Items...)
	}

	rsp.CommonData = h.actor.comData.FixDownComData()
	rsp.CommonData.LevelSummary = h.Dto2PClientLevelSummary(levelData.LevelId, 0)
	return rsp, nil, 0
}

// UpdateLevelTroopReq 关卡中更新队伍信息
func (h *ChapterHandler) UpdateLevelTroopReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var (
		err       error
		levelData *cmd.LS2DB_LevelInfo
		rsp       = &cmd.LS2C_UpdateLevelTroopRes{}
	)

	if h.IsInSubLevel() {
		return nil, err, int32(cmd.ErrorCode_Chapter_cannot_recover_in_sub)
	}

	var req cmd.C2LS_UpdateLevelTroopReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	levelsData := h.actor.GetLevelsData()
	if each, ok := h.GetCurrLevelData(); ok {
		levelData = each
	} else {
		return nil, fmt.Errorf("还未开始战斗, 当前所在的关卡id:%d", levelsData.CurrLevelId), int32(cmd.ErrorCode_InvalidParam)
	}

	// 恢复到满血
	h.UpdateCardHpFull(levelData)

	playerLevelData := h.getOrInitBattleCards(levelData, levelsData.TroopId)
	if playerLevelData == nil {
		return nil, errors.New("构建getPlayerLevelData错误"), int32(cmd.ErrorCode_InternalError)
	}

	levelData.PlayerLevelData = playerLevelData

	err = h.SaveDB()
	if err != nil {
		h.Errorf("UpdateLevelTroopReq SaveChapterData2DB 报错, err:%+v", err)
	}

	rsp = &cmd.LS2C_UpdateLevelTroopRes{
		PlayerLevelData: playerLevelData,
	}
	return rsp, nil, 0
}

// BackToBCReq 返回大本营
func (h *ChapterHandler) BackToBCReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		//err        error
		levelsData *cmd.LS2DB_LevelInfos
		levelData  *cmd.LS2DB_LevelInfo // 当前关卡数据
	)

	if h.IsInSubLevel() {
		return nil, fmt.Errorf("副本中不支持该回血"), int32(cmd.ErrorCode_Chapter_cannot_recover_in_sub)
	}

	levelsData = h.actor.GetLevelsData()

	if currLevel, ok := h.GetCurrLevelData(); ok {
		levelData = currLevel
	}

	//全都恢复满血
	h.UpdateCardHpFull(levelData)

	rsp := &cmd.LS2C_BackToBCRes{
		PlayerLevelData: h.getOrInitBattleCards(levelData, levelsData.GetTroopId()),
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelBackToBC{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Level_backtobc, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelBackToBC{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
		}
		taptap.WriteDataLog(taptap.LogType_Level_backtobc, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

// DiscoverMonsterReq 发现怪物
func (h *ChapterHandler) DiscoverMonsterReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
		//levelsData = h.actor.GetLevelsData()
		//levelData  *cmd.LS2DB_LevelInfo
		//niwaData   *cmd.BattleMapInfo
		cliComData *clidto.Comdata
		//rsp        = &cmd.LS2C_DiscoverMonsterRes{}
	)

	var req cmd.C2LS_DiscoverMonsterReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	cliComData, err, errCode := h.doDiscoverMonster(req.MonsterEventId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	rsp := &cmd.LS2C_DiscoverMonsterRes{}
	if cliComData != nil {
		rsp.CommonData = cliComData.Data
	}

	return rsp, nil, int32(cmd.ErrorCode_Success)
}

func (h *ChapterHandler) doDiscoverMonster(monsterEventId int32) (*clidto.Comdata, error, cmd.ErrorCode) {
	var (
		//err        error
		levelsData = h.actor.GetLevelsData()
		levelData  *cmd.LS2DB_LevelInfo
		niwaData   *cmd.BattleMapInfo
		//rsp        = &cmd.LS2C_DiscoverMonsterRes{}
	)

	dungeonentranceCfg := data.GetDungeonentranceMgr().GetById(monsterEventId)
	if dungeonentranceCfg == nil {
		h.Infof("配表中没有对应的数据, 不做处理:%d", monsterEventId)
		return nil, nil, cmd.ErrorCode_Success
	}

	if each, ok := h.GetCurrLevelData(); ok {
		levelData = each
	} else {
		return nil, fmt.Errorf("没有对应的关卡数据, levelId:%d", levelsData.CurrLevelId), cmd.ErrorCode_InvalidParam
	}

	if currNiwa, ok := h.GetCurrNiwaData(); ok {
		niwaData = currNiwa
	} else {
		return nil, fmt.Errorf("没有对应的地图数据, levelId:%d, niwaId:%d", levelsData.CurrLevelId, levelData.CurrNiwaId), cmd.ErrorCode_InvalidParam
	}

	// 地图上是否存在该事件
	if !checkEventIdInNiwa(niwaData, monsterEventId) {
		return nil, fmt.Errorf("无效的地图事件, uid=%s, roleId=%d, currLevel:%+v, currNiwa:%+v",
			h.actor.ID(), h.actor.GetUserData().Common.RoleId, levelData, niwaData), cmd.ErrorCode_InvalidParam
	}

	if simpleInfo, ok := levelsData.PLevelSummary.LevelSummaryMap[levelsData.CurrLevelId]; !ok {
		return nil, fmt.Errorf("没有对应的关卡数据, currLevelId=%d", levelsData.CurrLevelId), cmd.ErrorCode_InvalidParam
	} else {
		for _, each := range simpleInfo.MonsterList {
			if each.EventId == monsterEventId {
				h.Debugf("已经保存过该事件了:%d, monsterEventId=%d", levelsData.CurrLevelId, monsterEventId)
				return nil, nil, cmd.ErrorCode_Success
			}
		}

		// 事件组信息
		eventGroupInfo, err := h.getEventGroupInfo(levelsData.CurrLevelId, levelData.CurrNiwaId, monsterEventId)
		if err != nil {
			return nil, err, cmd.ErrorCode_InternalError
		}

		monsterEventInfo := &cmd.MonsterEventInfo{
			NiwaId:        levelData.CurrNiwaId,
			EventId:       monsterEventId,
			GroupId:       eventGroupInfo.GroupId,
			NextUpdateSec: eventGroupInfo.NextUpdateSec,
		}

		// 添加新的怪物数据
		simpleInfo.MonsterList = append(simpleInfo.MonsterList, monsterEventInfo)

		// 接口返回给前端
		h.actor.comData.GetLevelSummaryData().LevelSummaryList = append(h.actor.comData.GetLevelSummaryData().LevelSummaryList,
			&cmd.LevelSummary{
				LevelId:         levelData.LevelId,
				LevelSimpleInfo: simpleInfo.LevelSimpleInfo,
				MonsterList:     []*cmd.MonsterEventInfo{monsterEventInfo},
			})
	}

	err := h.SaveDB()
	if err != nil {
		return nil, err, cmd.ErrorCode_InternalError
	}

	return h.actor.comData, nil, cmd.ErrorCode_Success
}

//// SaveWeatherReq 保存天气
//func (h *ChapterHandler) SaveWeatherReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
//	var (
//		err     error
//		errCode cmd.ErrorCode
//	)
//
//	var req cmd.C2LS_SaveWeatherReq
//	_, _, err, errCode = in.UnmarshalData(&req)
//	if err != nil {
//		return nil, err, int32(errCode)
//	}
//
//	levelsData := h.actor.GetLevelsData()
//	levelsData.WeatherIdx = req.WeatherIdx
//	err = h.SaveDB()
//	if err != nil {
//		h.Errorf("SaveWeatherReq SaveDB 报错, err:%+v", err)
//	}
//
//	rsp := &cmd.LS2C_SaveWeatherRes{
//		WeatherIdx: levelsData.WeatherIdx,
//	}
//
//	return rsp, nil, int32(cmd.ErrorCode_Success)
//}

// 更新关卡事件组
func (h *ChapterHandler) updateLevelEventGroup(levelData *cmd.LS2DB_LevelInfo) {
	for _, mapInfo := range levelData.MapInfos {
		h.updateNiwaEventGroup(mapInfo)
	}

	err := h.SaveDB()
	if err != nil {
		h.Errorf("updateLevelEventGroup SaveDB 报错, err:%+v", err)
	}
}

// 更新地图事件组
func (h *ChapterHandler) updateNiwaEventGroup(mapInfo *cmd.BattleMapInfo) {
	var (
		now = time.Now().Unix()
	)
	h.Debugf("updateNiwaEventGroup, %+v", mapInfo.GetMappointEventGroupInfo())

	for _, eventGroupInfo := range mapInfo.GetMappointEventGroupInfo() {
		if eventGroupInfo.NextUpdateSec == common.NIWA_EVENT_GROUP_CD {
			// 无需刷新
			h.Debugf("updateNiwaEventGroup 无需刷新, %+v", eventGroupInfo)
			continue
		}

		if eventGroupInfo.NextUpdateSec > now {
			// 还未过期, 无需刷新
			h.Debugf("updateNiwaEventGroup 还未过期, %+v", eventGroupInfo)
			continue
		}

		eventGroupCfg := data.GetEventGroupMgr().GetById(eventGroupInfo.GetGroupId())
		if nil == eventGroupCfg {
			h.Warnf("updateNiwaEventGroup eventGroupCfg==nil, %+v", eventGroupInfo)
			continue
		}

		//// 更新事件组下次刷新时间戳
		//eventGroupInfo.NextUpdateSec = now + int64(eventGroupCfg.GetUpdateSec())

		h.Infof("updateNiwaEventGroup 刷新事件组的cd时间, mapId=%d, groupId=%d, eventGroup=%d, cd=%d",
			mapInfo.NiwaId, eventGroupInfo.GroupId, eventGroupCfg.GetId(), eventGroupInfo.NextUpdateSec)

		// 保留的事件id列表
		keepEventIds := make([]int32, 0)
		for _, eachEventId := range mapInfo.GetFinishedEventIds() {
			eventCfg := data.GetMappointEventMgr().GetById(eachEventId)
			if eventCfg == nil {
				continue
			}

			if eventCfg.GetGroupId() == eventGroupInfo.GetGroupId() {
				// 同一组的事件, 丢弃
				continue
			}
			keepEventIds = append(keepEventIds, eachEventId)
		}
		// 刷新事件
		mapInfo.FinishedEventIds = keepEventIds
		// 更新下次需要刷新事件的时间戳
		eventGroupInfo.NextUpdateSec = now + int64(eventGroupCfg.GetUpdateSec())

		h.Debugf("updateNiwaEventGroup 刷新后完成的事件, %+v", mapInfo.FinishedEventIds)
	}

}

// 启动地图事件组刷新倒计时
func (h *ChapterHandler) startEventGroupCD(levelData *cmd.LS2DB_LevelInfo, eventId int32) {
	var (
		hadChange = false
		now       = time.Now().Unix()
	)

	eventCfg := data.GetMappointEventMgr().GetById(eventId)
	eventGroupCfg := data.GetEventGroupMgr().GetById(eventCfg.GetGroupId())
	if eventGroupCfg == nil {
		return
	}

	if -1 == eventGroupCfg.GetUpdateSec() {
		// 配置为-1, 表示不刷新
		return
	}

	//for _, mapInfo := range levelData.MapInfos {
	//	for _, eventGroupInfo := range mapInfo.GetMappointEventGroupInfo() {
	//		if eventGroupCfg.GetId() != eventGroupInfo.GetGroupId() {
	//			// 不是该组事件
	//			h.Debugf("updateNiwaEventGroup 不是该组事件, %+v", eventGroupInfo)
	//			continue
	//		}
	//
	//		if eventGroupInfo.NextUpdateSec != common.NIWA_EVENT_GROUP_CD /*&& eventGroupInfo.NextUpdateSec >= now*/ { // 不是初始状态, 已经启动过了
	//			// 倒计时已启动
	//			h.Debugf("updateNiwaEventGroup 倒计时已启动, %+v", eventGroupInfo)
	//			continue
	//		}
	//
	//		eventGroupInfo.NextUpdateSec = now + int64(eventGroupCfg.GetUpdateSec())
	//		h.Infof("启动刷新事件组的cd时间, levelId=%d, eventId=%d, eventGroup=%d, cd=%d",
	//			levelData.LevelId, eventId, eventGroupCfg.GetId(), eventGroupInfo.NextUpdateSec)
	//		hadChange = true
	//		break
	//	}
	//}

	mapInfo, ok := h.GetCurrNiwaData()
	if !ok {
		h.Warnf("未找到当前地图数据, levelId=%d niwaId=%d", levelData.LevelId, levelData.CurrNiwaId)
		return
	}

	for _, eventGroupInfo := range mapInfo.GetMappointEventGroupInfo() {
		if eventGroupCfg.GetId() != eventGroupInfo.GetGroupId() {
			// 不是该组事件
			h.Debugf("updateNiwaEventGroup 不是该组事件, %+v", eventGroupInfo)
			continue
		}

		if eventGroupInfo.NextUpdateSec != common.NIWA_EVENT_GROUP_CD /*&& eventGroupInfo.NextUpdateSec >= now*/ { // 不是初始状态, 已经启动过了
			// 倒计时已启动
			h.Debugf("updateNiwaEventGroup 倒计时已启动, %+v", eventGroupInfo)
			continue
		}

		eventGroupInfo.NextUpdateSec = now + int64(eventGroupCfg.GetUpdateSec())
		h.Infof("启动刷新事件组的cd时间, levelId=%d, eventId=%d, eventGroup=%d, cd=%d",
			levelData.LevelId, eventId, eventGroupCfg.GetId(), eventGroupInfo.NextUpdateSec)
		hadChange = true
		break
	}

	if hadChange {
		err := h.SaveDB()
		if err != nil {
			h.Errorf("handleBattleEvent SaveChapterData2DB 报错, err:%+v", err)
		}
	}
	//
	//if hadChange {
	//
	//}
}

//func (h *ChapterHandler) pushEnterNewChapterSectionNtf(uid string) error {
//	ntf := &cmd.LS2C_EnterNewChapterSectionNtf{
//		ChapterId: 1,
//		SectionId: 101,
//	}
//
//	err := h.actor.PushMsg2Gate(ntf)
//	if err != nil {
//		h.Error("ChapterHandler invoke to gate got err ", err)
//		return err
//	}
//
//	h.Debug("pushEnterNewChapterSectionNtf :", h.actor.ID(), uid)
//	return nil
//}

// 事件处理
// @param uid
// @param [req] -->> [*cmd.C2LS_LevelEventReq, *cmd.C2LS_LevelBattleEventReq]
func (h *ChapterHandler) handleEvent(uid string, req proto.Message, commonData *clidto.Comdata) (*cmd.EventResData, error, cmd.ErrorCode) {
	var (
		err     error
		errCode cmd.ErrorCode

		niwaId        int32
		eventId       int32
		questObjectId int32
		//costFoods     = make(map[int32]int32)

		levelsData *cmd.LS2DB_LevelInfos
		levelData  *cmd.LS2DB_LevelInfo // 当前关卡数据
		niwaData   *cmd.BattleMapInfo   // 当前地图数据

		//rsp        = &cmd.EventResData{}
		//commonData = clidto.BuildComData()

		retOpenQuest          = make([]*cmd.PCommonQuestInfo, 0)
		retOldQuestId         int32
		retIncrMappointEvents = make([]*cmd.MappointEvent, 0)
	)
	h.Infof("===>>>uid=%s, handleEvent:%+v", uid, req)

	switch req.(type) {
	case *cmd.C2LS_LevelEventReq:
		levelEventReq := req.(*cmd.C2LS_LevelEventReq)
		niwaId = levelEventReq.NiwaId
		eventId = levelEventReq.EventId
		questObjectId = int32(levelEventReq.QuestObjectId)

	case *cmd.C2LS_LevelBattleEventReq:
		battleEventReq := req.(*cmd.C2LS_LevelBattleEventReq)
		niwaId = battleEventReq.NiwaId
		eventId = battleEventReq.EventId
		questObjectId = int32(battleEventReq.QuestObjectId)
		//for _, food := range battleEventReq.CostFoods {
		//	costFoods[food.Key] = food.Value
		//}

	default:
		h.Debugf("未支持的类型, %+v", req)
	}

	levelsData = h.actor.GetLevelsData()

	if currLevel, ok := h.GetCurrLevelData(); ok {
		levelData = currLevel
	}

	if currNiwa, ok := h.GetCurrNiwaData(); ok {
		niwaData = currNiwa
	}

	if niwaData.NiwaId != niwaId {
		return nil, fmt.Errorf("uid=%s, roleId=%d, 当前所在地图id:%d, req.NiwaId=%d, 不匹配",
			uid, h.actor.GetUserData().Common.RoleId, levelData.CurrNiwaId, niwaId), cmd.ErrorCode_Chapter_not_current_niwa
	}

	if questObjectId > 0 {
		questObjectCfg := data.GetQuestObjectMgr().GetById(questObjectId)
		if questObjectCfg != nil && questObjectCfg.GetEventId() != eventId {
			return nil, fmt.Errorf("物件id与事件id不符, uid=%s, roleId=%d, eventId=%d, questObjectCfg.EventId:%d, questObjectId:%d",
				uid, h.actor.GetUserData().Common.RoleId, eventId, questObjectCfg.GetEventId(), questObjectId), cmd.ErrorCode_ParamError
		}
	}

	eventCfg := data.GetMappointEventMgr().GetById(eventId)
	if eventCfg == nil {
		return nil, fmt.Errorf("无效的事件id, uid=%s, roleId=%d, eventId=%d",
			uid, h.actor.GetUserData().Common.RoleId, eventId), cmd.ErrorCode_ParamError
	}

	// 检查消耗品
	if err, errCode = h.tryEventCostItem(eventCfg); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("完成任务时, 消耗道具失败, costItem=%v", eventCfg.ItemSubmitGroupNum))
		return nil, err, errCode
	}

	if questObjectId > 0 {
		// 有物件id, 做物件相关条件检查
		if err, errCode = h.actor.QuestHandler.CheckQuestCondition(questObjectId); err != nil {
			return nil, err, errCode
		}
	} else if eventCfg.FromClient == 1 {
		// 客户端随出的事件, 不做事件检查
		if err, errCode = datahelper.CheckEventIdCanExistInArea(levelData.GetCurrNiwaId(), eventId); err != nil {
			return nil, err, errCode
		}
	} else {
		// 检查需要完成相关的事件
		if err, errCode = h.CheckFinishEventCondition(niwaData, eventId); err != nil {
			return nil, err, errCode
		}

		// 地图上是否存在该事件
		if !checkEventIdInNiwa(niwaData, eventId) {
			return nil, fmt.Errorf("无效的地图事件, uid=%s, roleId=%d, currLevel:%+v, currNiwa:%+v",
				uid, h.actor.GetUserData().Common.RoleId, levelData, niwaData), cmd.ErrorCode_Chapter_event_invaild
		}

		// 该事件是否是一次性事件并且完成过
		if finishedOnceEvent, ok := levelsData.FinishedOnceEvents[eventId]; ok {
			return nil, fmt.Errorf("事件已经完成过了, uid=%s, roleId=%d, currLevel:%+v, currNiwa:%+v, finishedOnceEvent:%+v",
				uid, h.actor.GetUserData().Common.RoleId, levelData, niwaData, finishedOnceEvent), cmd.ErrorCode_Chapter_event_had_done
		}
	}

	h.Infof("ChapterHandler.handleBattleEvent ===>>>, %d, %d, %v",
		eventId, eventCfg.GetEventType(), cmd.SceneEventType(eventCfg.GetEventType()))

	var (
		finalAddItemRewards = &cmd.DropChange{}
		tobeRewards         = make([]*data.ItemReward, 0)
		cardIds             = h.GetCardIds(levelsData.TroopId) // 上阵的阵容
		changeReason        = common.CR_EXEC_EVENT
	)

	// 战斗事件且战斗失败
	isBattleAndLost := false

	switch cmd.SceneEventType(eventCfg.GetEventType()) {
	case cmd.SceneEventType_SceneEventType_Battle, // 普通战斗
		cmd.SceneEventType_SceneEventType_Battle_boss,         // boss战斗
		cmd.SceneEventType_SceneEventType_Battle_infernalmobs: // 精英怪
		var (
			battleEventReq = req.(*cmd.C2LS_LevelBattleEventReq)
		)

		if battleEventReq.BattleResult == cmd.BattleResult_BattleResult_Winer {

			// 配置了事件id, 则需要检查是否消耗门票
			dungeonentranceCfg := data.GetDungeonentranceMgr().GetById(eventId)
			if dungeonentranceCfg != nil {
				_, currencyType, err := h.getMonsterTicket(cmd.LevelMonsterType(dungeonentranceCfg.EntranceType))
				if err != nil {
					h.Errorf(err.Error())
					return nil, err, cmd.ErrorCode_InvalidParam
				}

				if !h.actor.CurrencyHandler.CheckEnough(currencyType, 1) {
					return nil, fmt.Errorf("门票不足"), cmd.ErrorCode_Chapter_monster_ticket_not_enough
				} else {
					err := h.actor.CurrencyHandler.SubValue(currencyType, 1, commonData, common.CR_LEVEL_BATTLE_COST)
					if err != nil {
						return nil, err, cmd.ErrorCode_InternalError
					}
				}
			}

			battleEventCfg := data.GetBattleEventMgr().GetById(eventCfg.GetBattlestageId())

			var checkBattle *cmd.CheckBattleRes
			var subTypeId int32
			if myUtils.GetInt32AtBit(battleEventCfg.IsVerify, common.CheckBattleBitPos_doNot_checkBattle) == 1 ||
				myUtils.GetInt32AtBit(battleEventCfg.IsVerify, common.CheckBattleBitPos_DO_hpFull) == 1 {
				h.Debugf("玩家%v, 事件id=%d, 配置值:%d, 不做战斗校验", h.actor.ID(), eventId, battleEventCfg.IsVerify)
			} else {
				// 战斗校验
				mappointEventCfg := data.GetMappointEventMgr().GetById(eventId)
				if mappointEventCfg == nil {
					return nil, fmt.Errorf("没有对应的mappointEventCfg信息"), cmd.ErrorCode_ParamError
				}

				playerLevelData := h.getOrInitBattleCards(levelData, levelsData.TroopId)
				selfBattleTeam := h.actor.BattleHandler.buildSelfBattleCards(cmd.CardTroopType_CardTroopType_Normal, levelsData.TroopId, playerLevelData)
				checkBattle, err, errCode = h.actor.BattleHandler.CheckBattle(
					levelsData.BattleId, levelsData.BattleRandomSeed, battleEventReq.BattleResult,
					selfBattleTeam,
					mappointEventCfg.BattlestageId,
					battleEventReq.BattleFrameData, battleEventReq.VersionData)
				if err != nil {
					return nil, err, errCode
				}

				if checkBattle != nil && (checkBattle.CheckBattleResult == cmd.CheckBattleResult_CBR_fail || checkBattle.BattleResult != battleEventReq.BattleResult) {
					return nil, errors.New("校验失败"), cmd.ErrorCode_CheckBattle_fail
				}
				subTypeId = mappointEventCfg.GetSubtype()
			}
			if checkBattle == nil {
				// 校验关闭或其他情况没有返回值, 使用前端结果
				checkBattle = &cmd.CheckBattleRes{
					CheckBattleResult: cmd.CheckBattleResult_CBR_success,
					BattleResult:      battleEventReq.BattleResult,
					SelfCards:         battleEventReq.Card,
					OppoCards:         nil,
					CostFoods:         battleEventReq.CostFoods,
				}
			}

			// 消耗食物(胜利才会消耗) - 在战斗校验后处理，防止校验端拿到道具数量是扣除后的
			if myUtils.GetInt32AtBit(battleEventCfg.IsVerify, common.CheckBattleBitPos_doNot_checkFood) == 1 {
				h.Warnf("玩家%v, 事件id=%d, 不做校验食物", h.actor.ID(), eventId)
			} else {
				if !GetConsumeMgr(h.actor).CheckKeyValItemEnough(checkBattle.CostFoods) {
					return nil, err, cmd.ErrorCode_FoodNotEnough
				}
				err = GetConsumeMgr(h.actor).ConsumeKeyValItemList(checkBattle.CostFoods, commonData, changeReason)
				if err != nil {
					return nil, err, cmd.ErrorCode_FoodNotEnough
				}
				// 记录吃掉的食物
				h.UpdateUseFoods(levelData, checkBattle.CostFoods)
			}

			// 战斗计数
			levelsData.DailyTotalBattleCount = levelsData.DailyTotalBattleCount + 1

			// 同步血量
			if myUtils.GetInt32AtBit(battleEventCfg.IsVerify, common.CheckBattleBitPos_doNot_saveHp) == 1 {
				h.Warnf("玩家%v, 事件id=%d, 不做血量同步", h.actor.ID(), eventId)
			} else {
				h.UpdateCardHpSet(levelData, checkBattle.SelfCards, false)
			}

			// 怪物掉落
			for _, monster := range battleEventReq.Monster {
				monsterCfg := data.GetMonsterMgr().GetById(int32(monster.Common.MonsterId))
				rewards := datahelper.GetRewardsByDropId(monsterCfg.GetDropId())

				tobeRewards = append(tobeRewards, rewards...)
			}

			// 事件发布
			h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BATTLE_WIN, []int32{TASK_TYPE_92}, map[string]interface{}{
				"condition": int32(cmd.AchieveConditionType_Battle_5),
				"card_id":   cardIds,
				"type":      int(subTypeId),
				"level_id":  levelData.LevelId,
			}))
			//增加羁绊值
			h.actor.UserRelationHandler.AddRelation(cardIds, commonData, common.Realtion_type_win)
			h.Debugf("成就类型%d: 埋点level_id: %d", int32(cmd.AchieveConditionType_Battle_5), levelData.LevelId)
		} else if battleEventReq.BattleResult == cmd.BattleResult_BattleResult_Loser {
			isBattleAndLost = true // 标记战斗事件且失败

		} else {
			return nil, fmt.Errorf("不存在的战斗结果, %d", battleEventReq.BattleResult), cmd.ErrorCode_InvalidParam
		}

	case cmd.SceneEventType_SceneEventType_Collect: // 收集类事件
		/*// 扣除心能
		resourceCfg := data.GetResourceMgr().GetById(eventCfg.GetResource())
		if !GetConsumeMgr(h.actor).CheckEnough(common.ITEM_ID_COLLECT_COST_2008, resourceCfg.Cost) {
			return nil, nil, fmt.Errorf("心能不足, uid=%s, roleId=%d, resourceCfg.Cost:%d",
				uid, h.actor.GetUserData().Common.RoleId, resourceCfg.Cost), cmd.ErrorCode_CurrencyNotEnough
		}
		err = GetConsumeMgr(h.actor).ConsumeOne(common.ITEM_ID_COLLECT_COST_2008, resourceCfg.Cost, commonData, common.CR_Battle_collect)
		if err != nil {
			return nil, nil, err, cmd.ErrorCode_InternalError
		}*/

		// 发放奖励
		_rewards := datahelper.GetRewardsByResourceId(eventCfg.GetResource())
		tobeRewards = append(tobeRewards, _rewards...)

		// 事件发布
		var num int32
		temp := make(map[int32]int32)
		for _, v := range _rewards {
			temp[v.ItemId] += v.Num
			num += v.Num
		}
		h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_LEVEL_COLLECT, []int32{TASK_TYPE_11, TASK_TYPE_12, TASK_TYPE_525, TASK_TYPE_527}, map[string]interface{}{
			"condition":   int32(cmd.AchieveConditionType_Collect_4),
			"resource_id": int(eventCfg.Subtype),
			"num":         num,
			"card_id":     cardIds,
			"level_id":    levelData.LevelId,
			"reward":      temp,
		}))
		h.Debugf("成就类型%d: 埋点level_id: %d", int32(cmd.AchieveConditionType_Collect_4), levelData.LevelId)

	case cmd.SceneEventType_SceneEventType_SceneInsidePlot: // 剧情事件
		// do nothing...
	case cmd.SceneEventType_SceneEventType_TreasureChest, // 宝箱
		cmd.SceneEventType_SceneEventType_Ener, // 能量
		cmd.SceneEventType_SceneEventType_trap: // 荆棘（陷阱）

		if _rewards, err, errCode := h.handleDeviceLogic(uid, levelData, eventCfg.GetDeviceId(), commonData); err != nil {
			return nil, err, errCode
		} else {
			tobeRewards = append(tobeRewards, _rewards...)
			changeReason = common.CR_Battle_collect
		}

		// 任务
		if eventCfg.EventType == int32(cmd.SceneEventType_SceneEventType_TreasureChest) {
			h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_LEVEL_BOX, []int32{TASK_TYPE_31}, map[string]interface{}{
				"condition": int32(cmd.AchieveConditionType_Box_3),
				"card_id":   cardIds,
				"level_id":  levelData.LevelId,
			}))
			h.Debugf("成就类型%d: 埋点level_id: %d", int32(cmd.AchieveConditionType_Box_3), levelData.LevelId)
		}

	case cmd.SceneEventType_SceneEventType_tiny: // 微交互
		// 奖励统一走dropId下发
		// do nothing...
	default:
		h.Debugf("未支持的事件类型, req.EventId:%d, eventType:%d", eventId, eventCfg.GetEventType())
	}

	// 只有在 战斗事件并且失败, 才不走dropId逻辑
	if !isBattleAndLost {
		// 消耗事件体力
		if err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{common.ITEM_ID_STAMINA_1004: eventCfg.StaminaCost}, commonData, common.CR_EXEC_EVENT); err != nil {
			return nil, err, cmd.ErrorCode_StaminaValueNotEnough
		}
		//// 记录预扣的体力
		//levelData.PreCostStamina += eventCfg.StaminaPrecost

		// 记录完成的事件id
		if questObjectId > 0 || // 有物件id
			eventCfg.FromClient == 1 { // 客户端随出的事件
			// do nothing...不记录
		} else {
			// 记录关卡的完成事件id
			niwaData.FinishedEventIds = append(niwaData.FinishedEventIds, eventId)
			// 永久记录一次事件的id
			err, eachEvents5 := h.saveOnceEventFinished(levelData.LevelId, levelData.CurrNiwaId, eventId)
			if err != nil {
				return nil, err, cmd.ErrorCode_Chapter_event_had_done
			}
			retIncrMappointEvents = append(retIncrMappointEvents, eachEvents5...)
		}

		// dropId针对所有事件都有的奖励
		_rewards := datahelper.GetRewardsByDropId(eventCfg.GetDropId())
		tobeRewards = append(tobeRewards, _rewards...)

		// 掉落奖励
		if dropChange, err := GetDropMgr(h.actor).DropListByItems(tobeRewards, true, cardIds, commonData, changeReason); err != nil {
			return nil, err, cmd.ErrorCode_InternalError
		} else {
			mergeDropChange(finalAddItemRewards, dropChange)
		}
	}

	// 只有在 战斗事件并且失败, 才不走物件逻辑
	if !isBattleAndLost {
		// 启动事件组刷新倒计时
		h.startEventGroupCD(levelData, eventId)

		_dropChange := &cmd.DropChange{}
		_dropChange, _openQuest, _oldQuestId, eachEvents2, err, errCode := h.actor.QuestHandler.tryCompleteQuest(questObjectId, commonData)
		if err != nil && errCode != cmd.ErrorCode_Success {
			h.Errorf("tryCompleteQuest err: %+v", err)
			return nil, err, errCode
		}
		retIncrMappointEvents = append(retIncrMappointEvents, eachEvents2...)
		retOpenQuest = append(retOpenQuest, _openQuest...)
		retOldQuestId = _oldQuestId

		// 新任务走commonData下发
		commonData.GetQuestData().OpenQuests = append(commonData.GetQuestData().OpenQuests, _openQuest...)
		commonData.GetQuestData().CompleteQuests = append(commonData.GetQuestData().CompleteQuests, _oldQuestId)
		mergeDropChange(finalAddItemRewards, _dropChange)

		// 剧情任务刷新
		_dropChange, _openQuests, _oldQuests, _events, err, errCode := h.actor.QuestHandler.TryRefreshProgress(eventCfg.EventGroupId, commonData)
		if err != nil && errCode != cmd.ErrorCode_Success {
			h.Errorf("TryRefreshProgress err: %+v", err)
			return nil, err, errCode
		}
		commonData.GetQuestData().OpenQuests = append(commonData.GetQuestData().OpenQuests, _openQuests...)
		commonData.GetQuestData().CompleteQuests = append(commonData.GetQuestData().CompleteQuests, _oldQuests...)
		mergeDropChange(finalAddItemRewards, _dropChange)

		retIncrMappointEvents = append(retIncrMappointEvents, _events...)

	}

	err = h.SaveDB()
	if err != nil {
		h.Errorf("handleBattleEvent SaveChapterData2DB 报错, err:%+v", err)
	}

	rsp := &cmd.EventResData{
		NiwaId:          niwaId,
		EventId:         eventId,
		EventResult:     cmd.EventResult_EventResult_Success,
		DropChange:      finalAddItemRewards,
		PlayerLevelData: h.getOrInitBattleCards(levelData, levelsData.TroopId),
		QuestId:         retOldQuestId,
		BigLevelData:    levelData.BigLevelData,
	}
	if isBattleAndLost {
		rsp.EventResult = cmd.EventResult_EventResult_Failed
	}

	if len(retIncrMappointEvents) > 0 {
		// 完成任务，新增地图事件
		rsp.IncrEventInfo = &cmd.IncrMappointEventInfo{
			NiwaId:         niwaData.NiwaId,
			MappointEvents: retIncrMappointEvents,
		}
	}

	return rsp, nil, cmd.ErrorCode_Success
}

// 地图上是否存在该事件
func checkEventIdInNiwa(niwaData *cmd.BattleMapInfo, eventId int32) bool {
	if niwaData == nil {
		return false
	}

	hadFound := false
	for _, each := range niwaData.MappointEvents {
		if each.EventId == eventId {
			hadFound = true
			break
		}
	}

	return hadFound
}

// GetCurrBigLevelData 大地图数据(关卡:自身数据; 副本:副本嫁接的关卡数据)
func (h *ChapterHandler) GetCurrBigLevelData() (*cmd.LS2DB_LevelInfo, bool) {
	levelsData := h.actor.GetLevelsData()

	if ret, ok := levelsData.LevelInfos[levelsData.CurrBigLevelId]; ok {
		return ret, ok
	}
	return nil, false
}

// GetLevelData 获取关卡信息
func (h *ChapterHandler) GetLevelData(levelId int32) (*cmd.LS2DB_LevelInfo, bool) {
	levelsData := h.actor.GetLevelsData()

	if ret, ok := levelsData.LevelInfos[levelId]; ok {
		return ret, ok
	}
	return nil, false
}

// GetCurrLevelData 获取当前关卡信息
func (h *ChapterHandler) GetCurrLevelData() (*cmd.LS2DB_LevelInfo, bool) {
	//levelsData := h.actor.GetLevelsData()
	currLevelId := h.GetCurrLevelId()
	return h.GetLevelData(currLevelId)
}

// GetNiwaData 获取当前关卡的箱庭信息
func (h *ChapterHandler) GetNiwaData(levelId, niwaId int32) (*cmd.BattleMapInfo, bool) {
	levelData, levelOk := h.GetLevelData(levelId)
	if !levelOk {
		return nil, false
	}

	if levelData.MapInfos == nil {
		levelData.MapInfos = make(map[int32]*cmd.BattleMapInfo)
	}

	if niwa, ok := levelData.MapInfos[niwaId]; ok {
		return niwa, ok
	}
	return nil, false
}

// GetCurrNiwaData 获取当前的箱庭信息
func (h *ChapterHandler) GetCurrNiwaData() (*cmd.BattleMapInfo, bool) {
	levelData, levelOk := h.GetCurrLevelData()
	if !levelOk {
		return nil, false
	}

	return h.GetNiwaData(levelData.LevelId, levelData.CurrNiwaId)
}

// 获取当前箱庭信息
func (h *ChapterHandler) GetCurrNiwaId() int32 {
	if currNiwaData, ok := h.GetCurrNiwaData(); ok {
		return currNiwaData.NiwaId
	}

	return 0
}

// 检查自己关卡条件
func (h *ChapterHandler) checkSelfLevelCondition(levelId int32) error {
	levelsData := h.actor.GetLevelsData()

	// 当前关卡已经通关过了
	if simpleInfo, ok := levelsData.PLevelSummary.LevelSummaryMap[levelId]; ok {
		if simpleInfo.LevelSimpleInfo.HistoryHadPassed == cmd.HistoryHadPassed_PLevelStatus_Passed {
			levelCfg := data.GetLevelMgr().GetById(levelId)
			if levelCfg.Isonce == 1 {
				return fmt.Errorf("当前关卡已经完成, 且为一次性关卡, levelId=%d, 前置关卡:%d", levelId, levelCfg.GetPreLevelId())
			}
		}
	}

	return nil
}

// 检查关卡是否通关过
func (h *ChapterHandler) CheckMainLevelHadPassed(levelId int32) bool {
	levelsData := h.actor.GetLevelsData()

	// 当前关卡已经通关过了
	if simpleInfo, ok := levelsData.PLevelSummary.LevelSummaryMap[levelId]; ok {
		return simpleInfo.LevelSimpleInfo.HistoryHadPassed == cmd.HistoryHadPassed_PLevelStatus_Passed
	}

	return false
}

// 检查关卡可进入的前置条件
func (h *ChapterHandler) checkPreLevelCondition(levelId int32) error {
	levelCfg := data.GetLevelMgr().GetById(levelId)
	if levelCfg == nil {
		return fmt.Errorf("无效的关卡id, %d", levelId)
	}

	levelsData := h.actor.GetLevelsData()

	preLevelId := levelCfg.GetPreLevelId()
	if preLevelId > 0 {
		if _, ok := levelsData.LevelInfos[preLevelId]; ok {
			// 当前关卡已经通关过了
			if preSimpleInfo, ok := levelsData.PLevelSummary.LevelSummaryMap[preLevelId]; ok {
				if preSimpleInfo.LevelSimpleInfo.HistoryHadPassed != cmd.HistoryHadPassed_PLevelStatus_Passed {
					return fmt.Errorf("尚未完成前置关卡, %d, 前置关卡:%d", levelId, levelCfg.GetPreLevelId())
				}
			}
		}
	}

	return nil
}

// 清除所有子关卡信息
func (h *ChapterHandler) cleanAllSubLevelInfo() {
	levelsData := h.actor.GetLevelsData()

	for key, each := range levelsData.LevelInfos {
		levelCfg := data.GetLevelMgr().GetById(each.LevelId)
		if levelCfg == nil {
			logger.Debugf(fmt.Sprintf("副本数据不存在, key存在, key=%d", key))
			continue
		}
		if levelCfg.GetLevelType() == int32(common.CHAPTER_LEVEL_TYPE_SUB) { // 子关卡
			delete(levelsData.LevelInfos, key)
		}
	}

	// 取消进入副本的标记
	h.MarkInSubLevel(false)
	levelsData.CurrBigLevelId = 0 // 清除当前大地图关卡id(进副本时保存的是依赖的大地图关卡id)
	levelsData.CurrLevelId = 0    // 清除当前关卡id

	err := h.SaveDB()
	if err != nil {
		h.Errorf("cleanAllSubLevelInfo SaveChapterData2DB 报错, err:%+v", err)
	}
}

// 首次通关

func (h *ChapterHandler) firstPass(
	levelSimpleInfo *cmd.LevelSimpleInfo,
	monsterTicketInfoMap map[int32]*cmd.LevelMonsterTicketInfo,
	levelCfg *data.LevelCfg,
	commonData *clidto.Comdata) *cmd.DropChange {

	var (
		err            error
		onceAddRewards = &cmd.DropChange{} // 通关基础奖励 - 增量
		nowSec         = time.Now().Unix()
	)

	// 首次通关
	onceAddRewards, err = GetDropMgr(h.actor).DropListByItems(levelCfg.GetRewardOnce(), true, nil, commonData, common.CR_PASS_LEVEL_ONCE)
	if err != nil {
		h.Debugf("首通奖励报错, 关卡id:%d, err:%+v", levelCfg.GetId(), err)
	}

	//if levelSummary, ok := levelsData.LevelSimpleInfo[levelData.LevelId]; ok {
	levelSimpleInfo.HistoryHadPassed = cmd.HistoryHadPassed_PLevelStatus_Passed
	levelSimpleInfo.FirstPassedTimeSec = nowSec
	//}

	// 更新怪物门票最大值
	err = h.incrMonsterMaxTicketCount(monsterTicketInfoMap, commonData)
	if err != nil {
		h.Errorf(err.Error())
	}

	return onceAddRewards
}

// 处理奖励
func (h *ChapterHandler) handleItemReward(uid string, troopId int32, itemRewards []*data.ItemReward, commonData *clidto.Comdata) (*cmd.DropChange, error) {

	var (
		err            error
		addItemRewards *cmd.DropChange
		//finalItems     *cmd.LS2C_ChangeItemNtf
	)

	cardIds := h.GetCardIds(troopId)
	addItemRewards, err = GetDropMgr(h.actor).DropListByItems(itemRewards, true, cardIds, commonData, common.CR_Battle_collect)
	if err != nil {
		return addItemRewards, err
	}
	//if err = h.actor.BagHandler.pushChangeItemNtf(uid, finalItems); err != nil {
	//	return addItemRewards, err
	//}

	//// 卡牌增加经验
	//changeCards := h.actor.ChapterHandler.AddExpByTroop(cmd.CardTroopType_CardTroopType_Normal, troopId, addExp)
	//h.Debugf("handleItemReward ===>>> %v", changeCards)

	return addItemRewards, nil
}

// 进入关卡
func (h *ChapterHandler) enterLevel(levelId int32, troopId uint32) (*chapter.BattleLevel, *cmd.DropChange, error, cmd.ErrorCode) {
	var (
		//err         error
		levelsData  = h.actor.GetLevelsData() // 关卡列表
		battleLevel *chapter.BattleLevel
		simpleInfo  *cmd.LevelSummary

		itemRewards *cmd.DropChange
	)

	levelCfg := data.GetLevelMgr().GetById(levelId)

	// 保存当前攻打的关卡id
	levelsData.CurrLevelId = levelId

	if levelCfg.GetLevelType() == int32(common.CHAPTER_LEVEL_TYPE_SUB) {
		// 标记是否在副本中
		h.MarkInSubLevel(levelCfg.GetLevelType() == int32(common.CHAPTER_LEVEL_TYPE_SUB))
	} else {
		// 保存当前大地图关卡id(进入副本时保留依赖的大地图关卡id)
		levelsData.CurrBigLevelId = levelId
	}

	// 保存当前阵容id
	levelsData.TroopId = int32(troopId)

	// 关卡数据
	if each, ok := h.GetLevelData(levelId); ok {
		battleLevel = chapter.ReloadBattleLevel(each)
	} else {
		// 创建新的关卡数据
		battleLevel, simpleInfo = chapter.NewBattleLevel(
			levelId,
			//niwaId,
			levelsData.PLevelSummary.LevelSummaryMap, /*,
			levelsData.FinishedOnceEvents,
			h.actor.QuestHandler.GetCompleteQuestIds()*/)

		levelsData.LevelInfos[levelId] = battleLevel.FormatStage2DB()
		levelsData.PLevelSummary.LevelSummaryMap[levelId] = simpleInfo

		// 默认天气编号
		if err := h.updateBigLevelDataWeatherIdx(true, levelId, data.GetConfigMgr().GetCfg().DEFAULT_WEATHER_IDX); err != nil {
			h.Warnf("天气数据报错, %v", err.Error())
			//return nil, nil, err, cmd.ErrorCode_Chapter_update_weather_got_err
		}

		// 默认解锁的Unlocked-point
		mapunlockpointCfgs := datahelper.GetDefaultUnlockCfgs(levelsData.CurrLevelId)
		if _itemRewards, err, _ := h.doUnlockUnlockedPoint(
			levelId, simpleInfo.LevelSimpleInfo, mapunlockpointCfgs, false); err != nil {
			h.Warnf("默认解锁的Unlocked-point, %v", err.Error())
			//return nil, itemRewards, err, errCode
		} else {
			itemRewards = _itemRewards
		}
	}

	return battleLevel, itemRewards, nil, cmd.ErrorCode_Success
}

// 获取推图阵容
func (h *ChapterHandler) GetCardIds(troopId int32) []int32 {
	var (
		troopType = cmd.CardTroopType_CardTroopType_Normal
	)

	cardIds := h.actor.TroopHandler.GetTroopCardIds(int32(troopType), troopId)
	h.Debugf("获取组队数据: troopType:%d, troopId:%d, cards个数:%d", troopType, troopId, len(cardIds))

	return cardIds
}

// 构建 PlayerLevelData
func (h *ChapterHandler) getOrInitBattleCards(levelData *cmd.LS2DB_LevelInfo, troopId int32) *cmd.PlayerLevelData {
	var (
		troopType = cmd.CardTroopType_CardTroopType_Normal
	)

	cardIds := h.GetCardIds(troopId)
	if len(cardIds) <= 0 {
		return nil
	}

	if levelData.PlayerLevelData == nil {
		levelData.PlayerLevelData = &cmd.PlayerLevelData{
			BattleCards: make([]*cmd.PPlayerBattleCard, 0),
			Foods:       make([]int32, 0),
		}
	}

	battleCards := make([]*cmd.PPlayerBattleCard, 0)

	for _, eachId := range cardIds {
		if eachId <= 0 {
			continue
		}

		var battleCard *cmd.PPlayerBattleCard
		for _, each := range levelData.PlayerLevelData.BattleCards {
			if int32(each.CardId) == eachId {
				battleCard = each
				break
			}
		}

		card, err := h.actor.CardHandler.GetCard(uint32(eachId))
		if err != nil {
			h.Errorf(err.Error())
			continue
		}

		if battleCard != nil { // 已经有记录

			// 防止超过最大血量
			if battleCard.CardHp > card.OldMaxHp {
				battleCard.CardHp = card.OldMaxHp
			}

			battleCards = append(battleCards, battleCard)
		} else {
			// 新的英雄, 创建一份数据
			battleCards = append(battleCards, &cmd.PPlayerBattleCard{
				CardId:   uint32(eachId),
				CardHp:   card.OldMaxHp,
				CardEner: 0,
			})
		}
	}

	// 食物
	foods := h.actor.TroopHandler.GetTroopFoodLog(int32(troopType))

	playerLevelData := &cmd.PlayerLevelData{
		BattleCards: battleCards,
		Foods:       foods,
		UseFoods:    levelData.PlayerLevelData.UseFoods,
	}

	return playerLevelData
}

// UpdateCardHpSet 同步血量
func (h *ChapterHandler) UpdateCardHpSet(levelData *cmd.LS2DB_LevelInfo, changeCards []*cmd.PPlayerBattleCard, canDie bool) {
	if levelData.PlayerLevelData == nil {
		return
	}

	for _, card := range changeCards {
		if !h.actor.CardHandler.IsExistCard(card.CardId) {
			//卡牌不存在
			continue
		}

		var battleCard *cmd.PPlayerBattleCard = nil
		for _, _battleCard := range levelData.PlayerLevelData.BattleCards {
			if card.CardId != _battleCard.CardId {
				continue
			}

			battleCard = _battleCard
		}

		if battleCard != nil {
			h.doSyncCardHp(battleCard, int32(card.CardHp), canDie) // 累加\累减 血量
		} else {
			// 未找到历史记录, 直接加入列表
			levelData.PlayerLevelData.BattleCards = append(levelData.PlayerLevelData.BattleCards, &cmd.PPlayerBattleCard{
				CardId:   card.CardId,
				CardHp:   card.CardHp,
				CardEner: 0,
			})
		}
	}

	err := h.SaveDB()
	if err != nil {
		h.Errorf("UpdateCardHpSet 报错, err:%+v", err)
	}
}

// UpdateCardHpFull 恢复满血
func (h *ChapterHandler) UpdateCardHpFull(levelData *cmd.LS2DB_LevelInfo) {
	if levelData.PlayerLevelData == nil {
		return
	}

	for _, battleCard := range levelData.PlayerLevelData.BattleCards {
		dbCard, err := h.actor.CardHandler.GetCard(battleCard.CardId)
		if err != nil {
			h.Errorf("玩家:%s, 卡牌id=%d, 不存在", h.actor.Data.GetBase().Common.RoleId, battleCard.CardId)
			continue
		}

		h.doSyncCardHp(battleCard, int32(dbCard.OldMaxHp), false) // 同步到最大血量
	}

	err := h.SaveDB()
	if err != nil {
		h.Errorf("UpdateCardHpFull 报错, err:%+v", err)
	}
}

// UpdateCardHpAdd 加减血量
// @param addHp 小于0时,扣血; 大于0时,加血
func (h *ChapterHandler) UpdateCardHpAdd(levelData *cmd.LS2DB_LevelInfo, addHp int32, canDie bool) {
	if levelData.PlayerLevelData == nil {
		return
	}

	for _, battleCard := range levelData.PlayerLevelData.BattleCards {
		dbCard, err := h.actor.CardHandler.GetCard(battleCard.CardId)
		if err != nil {
			h.Errorf("玩家:%s, 卡牌id=%d, 不存在", h.actor.Data.GetBase().Common.RoleId, battleCard.CardId)
			continue
		}

		h.doSyncCardHp(battleCard, int32(dbCard.OldMaxHp)+addHp, canDie) // 累加\累减 血量
	}

	err := h.SaveDB()
	if err != nil {
		h.Errorf("UpdateCardHpAdd 报错, err:%+v", err)
	}
}

func (h *ChapterHandler) doSyncCardHp(battleCard *cmd.PPlayerBattleCard, syncHp int32, canDie bool) {
	var (
		err error
	)

	minHp := getCardMinHp(canDie)
	tempHp := myUtils.Max(minHp, syncHp)

	dbCard, _ := h.actor.CardHandler.GetCard(battleCard.CardId)
	if dbCard == nil {
		return
	}

	maxHp := dbCard.OldMaxHp
	tempHp = myUtils.Min(int32(maxHp), tempHp)

	if tempHp < 0 {
		err = fmt.Errorf("cardId=%d, 血量为负数, minHp=%d, maxHp=%d", battleCard.CardId, minHp, maxHp)
		h.Errorf(err.Error())
	}
	battleCard.CardHp = uint32(tempHp)
}

// MarkInSubLevel 标记是否在副本中
func (h *ChapterHandler) MarkInSubLevel(inSubLevel bool) {
	levelsData := h.actor.GetLevelsData()
	if inSubLevel {
		levelsData.InSubLevel = cmd.InSubLevelType_yes
	} else {
		levelsData.InSubLevel = cmd.InSubLevelType_no
	}
	err := h.SaveDB()
	if err != nil {
		h.Errorf("EnterLevelReq SaveChapterData2DB 报错, err:%+v", err)
	}
}

// IsInSubLevel 是否在副本中
func (h *ChapterHandler) IsInSubLevel() bool {
	levelsData := h.actor.GetLevelsData()
	return levelsData.InSubLevel == cmd.InSubLevelType_yes
}

// 清除战斗计数
func (h *ChapterHandler) cleanBattleCount() {
	// 清除总的战斗次数
	levelsData := h.actor.GetLevelsData()
	levelsData.DailyTotalBattleCount = 0

	// 清除每个通关副本次数
	for _, each := range levelsData.PLevelSummary.LevelSummaryMap {
		each.LevelSimpleInfo.DailyPassedCount = 0
	}

	err := h.SaveDB()
	if err != nil {
		h.Errorf("cleanBattleCount SaveChapterData2DB 报错, err:%+v", err)
	}
}

// 检查UnlockedPoint解锁条件
func (h *ChapterHandler) checkUnlockUnlockedPointCondition(currLevelId int32, UnlockedPointId int32) (error, cmd.ErrorCode) {
	mapUnlockPointCfg := data.GetMapunlockpointMgr().GetById(UnlockedPointId)
	if mapUnlockPointCfg == nil || mapUnlockPointCfg.StageId != currLevelId {
		return fmt.Errorf("该关卡没有对应的节点id, levelId=%d, UnlockedPointId=%d", currLevelId, UnlockedPointId),
			cmd.ErrorCode_ParamError
	}

	if err, errCode := h.checkUnlockCondition(currLevelId, mapUnlockPointCfg.UnlockCondition); err != nil {
		return err, errCode
	}

	if pass := h.checkFinishStoryFlagContains(mapUnlockPointCfg.OnFlagOr, mapUnlockPointCfg.OnFlag); !pass {
		return fmt.Errorf("flags 不满足条件, currLevelId=%d, UnlockedPointId=%d, mapUnlockPointCfg.OnFlagOr=%d, mapUnlockPointCfg.OnFlag=%v",
			currLevelId, UnlockedPointId, mapUnlockPointCfg.OnFlagOr, mapUnlockPointCfg.OnFlag), cmd.ErrorCode_Chapter_UnlockedPoint_cond_not_finish
	}

	return nil, cmd.ErrorCode_Success
}

// 检查完成的story-flag是否包含targetFlags列表
// @param andOr 逻辑运算标识(0: targetFlags列表全部需要完成; 1:targetFlags列表至少完成一个)
func (h *ChapterHandler) checkFinishStoryFlagContains(andOr int32, targetFlags []string) bool {
	storyFlagData := h.actor.GetStoryFlagData()

	if len(targetFlags) <= 0 {
		// 没有可检查的数据, 直接通过
		return true
	}

	switch andOr {
	case 0: // 且
		for _, targetFlag := range targetFlags {
			if _, ok := storyFlagData.Flags[targetFlag]; !ok {
				// 没有找到，直接结束
				return false
			}
		}
		// 全都找到了
		return true
	case 1: // 或
		for _, targetFlag := range targetFlags {
			if _, ok := storyFlagData.Flags[targetFlag]; ok {
				// 找一个了, 直接结束
				return true
			}
		}
		// 一个都没找到
		return false
	default:
		h.Warnf("on_flag_or, 尚未支持的类型, andOr=%d", andOr)
	}

	// 不支持的逻辑规则
	return false
}

func (h *ChapterHandler) handleDeviceLogic(uid string, levelData *cmd.LS2DB_LevelInfo, deviceId int32, commonData *clidto.Comdata) ([]*data.ItemReward, error, cmd.ErrorCode) {
	var (
		//err            error
		//addItemRewards = &cmd.DropChange{}
		//levelsData      *cmd.LS2DB_LevelInfos
		waitDropRewards = make([]*data.ItemReward, 0)
	)

	//levelsData = h.actor.GetLevelsData()

	mapDeviceCfg := data.GetMapDeviceMgr().GetById(deviceId)

	switch cmd.MapDeviceType(mapDeviceCfg.GetType()) {
	//case cmd.MapDeviceType_Change_card_ener: // 能量改变
	//	if err = h.updateCardEner(levelData, mapDeviceCfg.GetEffectPara()); err != nil {
	//		return addItemRewards, err, cmd.ErrorCode_InternalError
	//	}

	case cmd.MapDeviceType_Change_card_hp: // 血量改变
		h.UpdateCardHpAdd(levelData, mapDeviceCfg.GetEffectPara(), false) // 荆棘不可致死
		//if err != nil {
		//	return addItemRewards, err, cmd.ErrorCode_InternalError
		//}

	case cmd.MapDeviceType_Change_box_reward: // 宝箱
		rewards := datahelper.GetRewardsByDropId(mapDeviceCfg.GetDropId())
		waitDropRewards = append(waitDropRewards, rewards...)
		//if itemRewards, err := h.handleItemReward(uid, levelsData.TroopId, rewards, commonData); err != nil {
		//	return addItemRewards, err, cmd.ErrorCode_InternalError
		//} else {
		//	mergeDropChange(addItemRewards, itemRewards)
		//}

	default:
		h.Debugf("map_device表, 未支持的类型, id:%d, type:%d", mapDeviceCfg.GetId(), mapDeviceCfg.GetType())
	}

	return waitDropRewards, nil, cmd.ErrorCode_Success
}

func (h *ChapterHandler) doUnlockUnlockedPoint(currLevelId int32, simpleLevelData *cmd.LevelSimpleInfo,
	cfgs []*data.MapunlockpointCfg, needCheck bool) (*cmd.DropChange, error, cmd.ErrorCode) {

	var (
		_unlockPointIds = make([]int32, 0)
		cfgRewards      = make(map[int32]int32, 0)
	)

	for _, cfg := range cfgs {
		hadFound := false
		for _, each := range simpleLevelData.UnlockedPointInfos {
			if each.PointId == cfg.Id {
				hadFound = true
				break
			}
		}
		if hadFound {
			// 已经存过Unlocked-point
			continue
		}

		// 检查解锁条件
		if needCheck {
			if err, errCode := h.checkUnlockUnlockedPointCondition(currLevelId, cfg.Id); err != nil {
				return &cmd.DropChange{}, err, errCode
			}
		} else {
			h.Warnf("不检查点位解锁条件, 默认解锁, uid=%s, currLevelId=%d, cfgs=%v", h.actor.uid, currLevelId, cfgs)
		}

		UnlockedPoint := &cmd.UnlockedPointInfo{PointId: cfg.Id}
		simpleLevelData.UnlockedPointInfos = append(simpleLevelData.UnlockedPointInfos, UnlockedPoint)
		err := h.SaveDB()
		if err != nil {
			h.Errorf("_doUnlockUnlockedPoint SaveDB 报错, err:%+v", err)
		}

		mapUnlockPointCfg := data.GetMapunlockpointMgr().GetById(cfg.Id)
		myUtils.MergeItems(cfgRewards, mapUnlockPointCfg.GetUnlockReward())
		_unlockPointIds = append(_unlockPointIds, cfg.Id)
	}

	// 下发奖励
	addRewards, err := GetDropMgr(h.actor).DropList2(cfgRewards, true, nil, h.actor.comData, common.CR_DISCOVERY_UNLOCK_POINT)
	if err != nil {
		return nil, fmt.Errorf("DropList2 出错:%d", currLevelId), cmd.ErrorCode_InternalError
	}
	//err = h.actor.BagHandler.pushChangeItemNtf(uid, rewards)
	//if err != nil {
	//	return nil, fmt.Errorf("pushChangeItemNtf 出错:%d", currLevelId), cmd.ErrorCode_InternalError
	//}

	if len(_unlockPointIds) > 0 {
		// 事件发布
		h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_UNLOCK_POINT, []int32{}, map[string]interface{}{
			"condition": int32(cmd.AchieveConditionType_Level_2),
			"point_id":  _unlockPointIds,
			"level_id":  currLevelId,
		}))
		h.Debugf("成就类型%d: 埋点level_id: %d", int32(cmd.AchieveConditionType_Level_2), currLevelId)
		// 埋点
		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.LevelUnlockPoint{
		//		CustomHeadInfo:  lilith.BuildCustomHeadInfo(lilith.LogType_Level_unlockpoint, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		UnlockedPointId: lilith.ConvertList2Str(_unlockPointIds),
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.LevelUnlockPoint{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				UnlockedPointId:   taptap.ConvertList2Str(_unlockPointIds),
			}
			taptap.WriteDataLog(taptap.LogType_Level_unlockpoint, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	return addRewards, nil, cmd.ErrorCode_Success
}

// 解析LevelCfg.UnlockCondition mapunlockpoint.UnlockCondition的字段
func (h *ChapterHandler) checkUnlockCondition(currLevelId int32, unlockCondition map[int32]int32) (error, cmd.ErrorCode) {
	for condType, condVal := range unlockCondition {
		switch condType {
		case 1: // 任务是否完成
			if !h.actor.QuestHandler.checkQuestFinish(condVal) {
				return fmt.Errorf("指定任务尚未达到, currLevelId=%d, unlockCondition=%v, condType=%d, questId=%d",
					currLevelId, unlockCondition, condType, condVal), cmd.ErrorCode_Chapter_UnlockedPoint_cond_not_finish
			}
		case 2: // 玩家达到指定等级
			if int32(h.actor.GetUserData().Common.RoleLevel) < condVal {
				return fmt.Errorf("指定玩家等级尚未达到, unlockCondition=%v, UnlockedPointId=%d, condType=%d, targetLv=%d",
					currLevelId, unlockCondition, condType, condVal), cmd.ErrorCode_Chapter_UnlockedPoint_cond_not_finish
			}
		case 3: // 完成指定关卡
			if simpleInfo, ok := h.actor.GetLevelsData().PLevelSummary.LevelSummaryMap[condVal]; !ok || simpleInfo.LevelSimpleInfo.HistoryHadPassed != cmd.HistoryHadPassed_PLevelStatus_Passed {
				// do nothing... 指定关卡尚未完成
				return fmt.Errorf("指定关卡尚未完成, unlockCondition=%v, UnlockedPointId=%d, condType=%d, targetLevelId=%d",
					currLevelId, unlockCondition, condType, condVal), cmd.ErrorCode_Chapter_UnlockedPoint_cond_not_finish
			}
		case 4: // 完成旅途关卡
			if !h.actor.TravelLevelHandler.CheckTravelLevelHadPassed(condVal) {
				return fmt.Errorf("指定旅途关卡尚未完成, unlockCondition=%v, UnlockedPointId=%d, condType=%d, targetLevelId=%d",
					currLevelId, unlockCondition, condType, condVal), cmd.ErrorCode_Chapter_UnlockedPoint_cond_not_finish
			}

		default:
			h.Warnf("unlock_condition, 尚未支持解锁类型, unlockCondition=%v, UnlockedPointId=%d, condType=%d", currLevelId, unlockCondition, condType)
		}
	}

	return nil, cmd.ErrorCode_Success
}

func (h *ChapterHandler) GmExitLevel(uid string, levelId, endId int, battleResult cmd.BattleResult) (error, cmd.ErrorCode) {
	var (
		err     error
		errCode cmd.ErrorCode
		rsp     = &cmd.LS2C_LevelBattleSettlementRes{}
	)

	if err, errCode = h.doExitLevel(uid, int32(levelId), int32(endId), battleResult, rsp, true); err != nil {
		return err, errCode
	}
	return nil, cmd.ErrorCode_Success
}

func (h *ChapterHandler) doExitLevel(uid string, levelId, endId int32, battleResult cmd.BattleResult, rsp *cmd.LS2C_LevelBattleSettlementRes, isGM bool) (error, cmd.ErrorCode) {
	var (
		err     error
		errCode cmd.ErrorCode

		levelsData *cmd.LS2DB_LevelInfos
		levelData  *cmd.LS2DB_LevelInfo
		nowSec     = time.Now().Unix()

		//onceAddRewards *DropChange // 首次通关奖励 - 增量
		//dropChage *cmd.DropChange // 通关基础奖励 - 增量
	)
	h.Infof("doExitLevel, uid=%s, levelId=%d, battleResult=%d, isGM=%v", uid, levelId, battleResult, isGM)

	// 地图信息
	levelsData = h.actor.GetLevelsData()
	if err != nil {
		h.Errorf("load user strength pool failed. err:", err)
		return err, cmd.ErrorCode_InternalError
	}

	if each, ok := h.GetCurrLevelData(); ok {
		levelData = each
	} else {
		return fmt.Errorf("还未有关卡数据, 请求的关卡id:%d", levelId), cmd.ErrorCode_InvalidParam
	}

	levelCfg := data.GetLevelMgr().GetById(levelData.LevelId)
	if levelCfg == nil {
		return fmt.Errorf("没有对应的levelCfg信息"), cmd.ErrorCode_ParamError
	}

	// 胜利
	if battleResult == cmd.BattleResult_BattleResult_Winer {
		// 检查关卡通关条件
		if isGM {
			h.Debugf("GM命令调用, 不做检查, uid=%s, levelId=%d", uid, levelId)
		} else {
			if err, errCode = h.checkLevelFinishCondition(endId, levelData); err != nil {
				return err, errCode
			}
		}

		// 扣除副本需要消耗的体力
		if NeedWithholdStamina(common.LEVEL_TYPE(levelCfg.LevelType)) {
			err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{common.ITEM_ID_STAMINA_1004: levelCfg.StaminaCost}, h.actor.comData, common.CR_PASS_LEVEL_BASE)
			if err != nil {
				return err, cmd.ErrorCode_StaminaValueNotEnough
			}
		}

		if simpleInfo, ok := levelsData.PLevelSummary.LevelSummaryMap[levelData.LevelId]; ok {
			// 首次通关
			if simpleInfo.LevelSimpleInfo.HistoryHadPassed == cmd.HistoryHadPassed_PLevelStatus_None {
				h.Warnf("玩家 %s, 首次通关关卡, levelId=%d", uid, levelData.LevelId)

				_dropChange := h.firstPass(simpleInfo.LevelSimpleInfo, levelsData.PLevelSummary.MonsterTicketInfoMap, levelCfg, h.actor.comData)
				mergeDropChange(rsp.OnceDropChange, _dropChange, true) // 首通奖励

				// 结算界面的显示数据，需要玩家上次的等级，和玩家这次的经验成长
				rsp.Role = &cmd.PRoleBattleSettlement{
					RoleLevel: h.actor.GetUserData().Common.RoleLevel,
					RoleExp:   uint32(levelCfg.GetPlayerExp()),
				}

				// 角色经验 - 首通才有
				dropChange, err := GetDropMgr(h.actor).DropList2(map[int32]int32{common.ITEM_ID_ROLE_EXP_1001: levelCfg.GetPlayerExp()},
					false, nil, h.actor.comData, common.CR_PASS_LEVEL_ONCE)
				if err != nil {
					return fmt.Errorf("ExitLevelReq AddRoleExp 报错, roleId:%d, 关卡id:%d, err:%+v",
							h.actor.GetUserData().Common.RoleId, levelId, err),
						cmd.ErrorCode_InternalError
				}
				mergeDropChange(rsp.OnceDropChange, dropChange, true) // 首通奖励

				//_, err = h.actor.PlayerLevelHandler.AddRoleExp(uint64(levelCfg.GetPlayerExp()))
				//if err != nil {
				//	return fmt.Errorf("ExitLevelReq AddRoleExp 报错, roleId:%d, 关卡id:%d, err:%+v",
				//			h.actor.GetUserData().Common.RoleId, levelId, err),
				//		cmd.ErrorCode_InternalError
				//}

				//h.actor.AddRoleExp(uint64(levelCfg.GetPlayerExp()))
				//err = h.actor.LoginHandler.SaveDB()
				//if err != nil {
				//	return fmt.Errorf("ExitLevelReq SaveUserData2DB 报错, roleId:%d, 关卡id:%d, err:%+v",
				//			h.actor.GetUserData().Common.RoleId, levelId, err),
				//		cmd.ErrorCode_InternalError
				//}
			}

			simpleInfo.LevelSimpleInfo.LastPassedTimeSec = nowSec
			//  统计次数
			simpleInfo.LevelSimpleInfo.DailyPassedCount += 1
		}

		// 结算界面的显示数据，需要玩家上次的等级，和玩家这次的经验成长
		if rsp.Role == nil {
			rsp.Role = &cmd.PRoleBattleSettlement{
				RoleLevel: h.actor.GetUserData().Common.RoleLevel,
				RoleExp:   0,
			}
		}

		// 持久化
		err = h.SaveDB()
		if err != nil {
			return fmt.Errorf("ExitLevelReq SaveChapterData2DB 报错, roleId:%d, 关卡id:%d, err:%+v",
					h.actor.GetUserData().Common.RoleId, levelId, err),
				cmd.ErrorCode_InternalError
		}

		rewards := datahelper.GetRewardsByDropId(levelCfg.RewardRandom)
		rewards = append(rewards, levelCfg.GetRewardBase()...)
		_dropChange, err := GetDropMgr(h.actor).DropListByItems(rewards, true, nil, h.actor.comData, common.CR_PASS_LEVEL_BASE)
		if err != nil {
			h.Debugf("基础奖励报错, 关卡id:%d, err:%+v", levelId, err)
		}
		mergeDropChange(rsp.DropChange, _dropChange, true)

		// 玩家数据
		h.actor.comData.Data.Base = h.actor.LoginHandler.buildRoleBaseInfo()

		// 事件发布
		h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_LEVEL_WIN, []int32{TASK_TYPE_91}, map[string]interface{}{
			"condition": int32(cmd.AchieveConditionType_Level_6),
			"level_id":  levelData.LevelId,
			"card_id":   h.GetCardIds(levelsData.TroopId),
		}))
		h.Debugf("成就类型%d: 埋点level_id: %d levelId:%d", int32(cmd.AchieveConditionType_Level_6), levelData.LevelId)
	} else {
		//// 失败
		//if levelData.BattleLostPunish == 0 {
		//	err = h.loseBattleSyncCardHp(levelData, true, commonData)
		//	if err != nil {
		//		return err, cmd.ErrorCode_InternalError
		//	}
		//}

		//// 对进入时预扣的体力尝试返还
		//if NeedWithholdStamina(common.LEVEL_TYPE(levelCfg.LevelType)) {
		//	//levelCfg := data.GetLevelMgr().GetById(levelId)
		//	//if levelCfg == nil {
		//	//	return fmt.Errorf("没有对应的levelCfg信息"), cmd.ErrorCode_ParamError
		//	//}
		//
		//	h.Warnf("结算中, 返回预扣的体力, levelCfg.StaminaCost=%d, levelData.PreCostStamina=%d",
		//		levelCfg.StaminaCost, levelData.PreCostStamina)
		//
		//	giveBackStamina := myUtils.Min(levelCfg.StaminaCost, levelCfg.StaminaCost-levelData.PreCostStamina)
		//	giveBackStamina = myUtils.Max(0, giveBackStamina)
		//	err, _dropChange := h.AddStamina(giveBackStamina, h.actor.comData)
		//	if err != nil {
		//		return err, cmd.ErrorCode_InternalError
		//	}
		//	mergeDropChange(rsp.DropChange, _dropChange, true)
		//}
	}

	// 取消标记进入副本
	h.MarkInSubLevel(false)
	levelsData.CurrBigLevelId = 0 // 清除当前大地图关卡id(进副本时保存的是依赖的大地图关卡id)
	levelsData.CurrLevelId = 0    // 清除当前关卡id

	//h.updateBigLevelDataExitLevel(levelId)
	//// 取消进入副本的标记
	//h.MarkInSubLevel(false)
	//// 清除当前关卡id
	//levelsData.CurrLevelId = 0

	// 持久化
	err = h.SaveDB()
	if err != nil {
		h.Errorf("ExitLevelReq saveDB 报错, err:%+v", err)
	}

	//err = h.actor.BagHandler.pushChangeItemNtf(uid, onceRewards)
	//if err != nil {
	//	_ = fmt.Errorf("推送数据get err:%+v", err)
	//}
	//err = h.actor.BagHandler.pushChangeItemNtf(uid, baseRewards)
	//if err != nil {
	//	_ = fmt.Errorf("推送数据get err:%+v", err)
	//}

	//if err = h.actor.LoginHandler.Dto2PClientLevelSummary(uid, levelData.LevelId); err != nil {
	//	return fmt.Errorf("Dto2PClientLevelSummary, 出错, levelId:%d", levelData.LevelId), cmd.ErrorCode_InternalError
	//}
	playerLevelCard := h.getOrInitBattleCards(levelData, levelsData.TroopId)
	if nil == playerLevelCard {
		return fmt.Errorf("队伍中没有活着的卡牌, troopId:%d", levelsData.TroopId), cmd.ErrorCode_Chapter_no_live_in_troop
	}

	h.actor.comData.Data.LevelSummary = h.Dto2PClientLevelSummary(levelData.LevelId, 0)
	rsp.LevelId = levelData.LevelId
	rsp.BattleResult = battleResult
	rsp.Exploration = 100
	rsp.Collection = 0
	rsp.CommonData = h.actor.comData.FixDownComData()
	rsp.PlayerLevelData = playerLevelCard

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.LevelExit{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_Level_exit, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		BattleResult:   int64(rsp.BattleResult),
	//		LevelId:        levelData.LevelId,                                             // 关卡id
	//		BattleCards:    lilith.ConvertListStruct2Str(rsp.PlayerLevelData.BattleCards), // 卡牌信息 []*cmd.PPlayerBattleCard
	//		Foods:          lilith.ConvertList2Str(rsp.PlayerLevelData.Foods),             // 食物itemId列表
	//		Collection:     rsp.Collection,                                                // 消耗采集点数
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.LevelExit{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			BattleResult:      int64(rsp.BattleResult),
			LevelId:           levelData.LevelId,                                             // 关卡id
			BattleCards:       taptap.ConvertListStruct2Str(rsp.PlayerLevelData.BattleCards), // 卡牌信息 []*cmd.PPlayerBattleCard
			Foods:             taptap.ConvertList2Str(rsp.PlayerLevelData.Foods),             // 食物itemId列表
			Collection:        rsp.Collection,                                                // 消耗采集点数
		}
		taptap.WriteDataLog(taptap.LogType_Level_exit, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return nil, cmd.ErrorCode_Success
}

// 检查关卡通关条件
func (h *ChapterHandler) checkLevelFinishCondition(endId int32, levelData *cmd.LS2DB_LevelInfo) (error, cmd.ErrorCode) {
	var (
		err     error
		errCode cmd.ErrorCode
	)

	// 检查flag是否全部完成
	if err, errCode = h.checkFinishFlags(levelData.LevelId, endId); err != nil {
		return err, errCode
	}

	// 检查事件是否全部完成
	if err, errCode = h.checkFinishEventIds(levelData.LevelId, endId); err != nil {
		return err, errCode
	}

	// 检查物件是否全部完成
	if err, errCode = h.checkFinishQuestIds(levelData.LevelId, endId); err != nil {
		return err, errCode
	}

	//// 检查指定任务是否完成
	//err, errCode = h.checkFinishMainLevelCondition(levelsData, levelData)
	//if err != nil {
	//	return err, errCode
	//}

	return nil, cmd.ErrorCode_Success
}

// 检查物件是否已经完成
func (h *ChapterHandler) checkFinishQuestIds(levelId, endId int32) (error, cmd.ErrorCode) {
	levelEndCfg := data.GetLevelEndMgr().GetById(endId)
	if levelEndCfg == nil {
		return errors.New(fmt.Sprintf("配置未读到, endId=%d", endId)), cmd.ErrorCode_Chapter_finish_questId_undone
	}

	if levelId != levelEndCfg.LevelId {
		return errors.New(fmt.Sprintf("参数错误, levelId=%d, endId=%d", levelId, endId)), cmd.ErrorCode_ParamError
	}

	if !h.actor.QuestHandler.checkQuestFinish(levelEndCfg.AppearConditionQuestid...) {
		return errors.New("flag还未完成"), cmd.ErrorCode_Chapter_finish_flag_undone
	}

	return nil, cmd.ErrorCode_Success
}

// 检查flags是否已经完成
func (h *ChapterHandler) checkFinishFlags(levelId, endId int32) (error, cmd.ErrorCode) {
	levelEndCfg := data.GetLevelEndMgr().GetById(endId)
	if levelEndCfg == nil {
		return errors.New(fmt.Sprintf("配置未读到, endId=%d", endId)), cmd.ErrorCode_Chapter_finish_flag_undone
	}

	if levelId != levelEndCfg.LevelId {
		return errors.New(fmt.Sprintf("参数错误, levelId=%d, endId=%d", levelId, endId)), cmd.ErrorCode_ParamError
	}

	if !h.actor.StoryFlagHandler.checkExistFlags(levelEndCfg.AppearConditionFlag...) {
		return errors.New("flag还未完成"), cmd.ErrorCode_Chapter_finish_flag_undone
	}

	return nil, cmd.ErrorCode_Success
}

// 检查flags是否已经完成
func (h *ChapterHandler) checkFinishEventIds(levelId, endId int32) (error, cmd.ErrorCode) {
	levelEndCfg := data.GetLevelEndMgr().GetById(endId)
	if levelEndCfg == nil {
		return errors.New(fmt.Sprintf("配置未读到, endId=%d", endId)), cmd.ErrorCode_Chapter_finish_eventId_undone
	}

	if levelId != levelEndCfg.LevelId {
		return errors.New(fmt.Sprintf("参数错误, levelId=%d, endId=%d", levelId, endId)), cmd.ErrorCode_ParamError
	}

	finishEventIdMap := make(map[int32]bool, 0)
	niwaData, ok := h.GetNiwaData(levelId, levelEndCfg.NiwaId)
	if !ok {
		return errors.New(fmt.Sprintf("没有获取到地图数据, levelId:%d, niwaId:%d, endId:%d", levelId, levelEndCfg.NiwaId, endId)),
			cmd.ErrorCode_Chapter_finish_eventId_undone
	}
	for _, evenId := range niwaData.FinishedEventIds {
		finishEventIdMap[evenId] = true
	}

	for _, eventId := range levelEndCfg.AppearConditionEventid {
		if _, ok := finishEventIdMap[eventId]; !ok {
			return errors.New(fmt.Sprintf("事件未完成, eventId:%d", eventId)), cmd.ErrorCode_Chapter_finish_eventId_undone
		}
	}

	return nil, cmd.ErrorCode_Success
}

// 记录一次性事件完成数据
func (h *ChapterHandler) saveOnceEventFinished(levelId, niwaId, eventId int32) (error, []*cmd.MappointEvent) {
	eventCfg := data.GetMappointEventMgr().GetById(eventId)
	if eventCfg == nil {
		return fmt.Errorf("eventCfg not found %d", eventId), nil
	}

	groupCfg := data.GetEventGroupMgr().GetById(eventCfg.GetGroupId())
	if groupCfg == nil {
		return fmt.Errorf("groupCfg not found %d", eventId), nil
	}

	if -1 != groupCfg.GetUpdateSec() {
		// 非一次性事件
		return nil, nil
	}

	levelsData := h.actor.GetLevelsData()
	if finishedOnceEvent, ok := levelsData.FinishedOnceEvents[eventId]; ok {
		return errors.New(fmt.Sprintf("一次性事件已经有记录, 不可重复记录, eventId=%d, finishedOnceEvent=%v", eventId, finishedOnceEvent)), nil
	} else {
		h.Warnf(fmt.Sprintf("记录一次性事件, eventId=%d, finishedOnceEvent=%v", eventId, finishedOnceEvent))
		levelsData.FinishedOnceEvents[eventId] = &cmd.FinishedOnceEvent{
			LevelId:         levelId,
			EventId:         eventId,
			FinishedTimeSec: time.Now().Unix(),
		}
	}

	// 完成指定的一次性事件，新增事件
	eachEvents5, _ := h.TryIncrMappointEventByType(
		levelId, niwaId,
		common.MAPPOINT_EVENT_UPDATE_TYPE_5, myUtils.Convert2StrList([]int32{eventId}))
	//incrMappointEvents = append(incrMappointEvents, eachEvents5...)
	//incrEventGroups = append(incrEventGroups, eachEventGroups5...)

	return nil, eachEvents5
}

//func (h *ChapterHandler) AddStamina(addNum int32, commonData *clidto.Comdata) (error, *cmd.DropChange) {
//	if addNum < 0 {
//		return fmt.Errorf("AddStamina param addNum=%d, is less than ZERO", addNum), nil
//	}
//
//	_dropChange, err := GetDropMgr(h.actor).DropList2(map[int32]int32{common.ITEM_ID_STAMINA_1004: addNum}, true, nil, commonData, common.CR_REBACK_StAMINA)
//	if err != nil {
//		return err, nil
//	}
//	return nil, _dropChange
//
//}

// 扣除体力
func (h *ChapterHandler) CostStamina(costNum int32, commonData *clidto.Comdata, changeReason common.ChangeReason) (error, cmd.ErrorCode) {
	if costNum < 0 {
		h.Warnf("CostStamina param costNum=%d, is less than ZERO", costNum)
		return nil, cmd.ErrorCode_Success
	}

	// 扣除体力
	err := GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{common.ITEM_ID_STAMINA_1004: costNum}, commonData, changeReason)
	if err != nil {
		return err, cmd.ErrorCode_StaminaValueNotEnough
	}
	return nil, cmd.ErrorCode_Success
}

//func (h *ChapterHandler) updateBigLevelDataEnterLevel(levelId int32) {
//	levelCfg := data.GetLevelMgr().GetById(levelId)
//	levelsData := h.actor.GetLevelsData()
//
//	if levelCfg.GetLevelType() == int32(common.CHAPTER_LEVEL_TYPE_SUB) {
//		// 标记进入副本
//		h.MarkInSubLevel(true)
//	} else {
//		levelsData.CurrBigLevelId = levelId // 保存当前大地图关卡id
//		levelsData.CurrLevelId = levelId
//	}
//}

//func (h *ChapterHandler) updateBigLevelDataExitLevel(levelId int32) {
//	levelsData := h.actor.GetLevelsData()
//
//}

func (h *ChapterHandler) updateBigLevelDataWeatherIdx(isFromCreate bool, levelId, weatherIdx int32) error {
	if weatherIdx <= 0 {
		return nil
	}

	levelData, ok := h.GetLevelData(levelId)
	if !ok {
		return fmt.Errorf("没有当前关卡数据")
	}
	levelCfg := data.GetLevelMgr().GetById(levelData.LevelId)

	if levelCfg.GetLevelType() == int32(common.CHAPTER_LEVEL_TYPE_SUB) {

		bigChapterCfg := data.GetChapterMgr().GetById(levelCfg.GetChapterId())
		bigLevelCfg := data.GetLevelMgr().GetById(bigChapterCfg.Bigmaplevel) // 副本所属关卡的配置

		// 副本所属关卡的数据
		bigLevelData, ok := h.GetLevelData(bigLevelCfg.Id)
		if !ok {
			return errors.New(fmt.Sprintf("没有对应的所属关卡数据: 副本id=%d, 所属关卡id=%d", levelData.LevelId, bigLevelCfg.Id))
		}
		if bigLevelData.BigLevelData == nil {
			return errors.New(fmt.Sprintf("没有对应的所属关卡天气: 副本id=%d, 所属关卡id=%d", levelData.LevelId, bigLevelCfg.Id))
		}

		if isFromCreate {
			// 创建副本时，继承所属关卡的天气数据
			levelData.BigLevelData = &cmd.BigLevelData{
				BigLevelId: levelData.LevelId,
				WeatherIdx: bigLevelData.BigLevelData.WeatherIdx,
			}
		} else {
			// 副本中改变天气，需同时更新所属关卡的天气数据
			levelData.BigLevelData.WeatherIdx = weatherIdx
			bigLevelData.BigLevelData.WeatherIdx = weatherIdx
		}
	} else {
		// 大地图时, 直接更新当前关卡数据
		levelData.BigLevelData = &cmd.BigLevelData{
			BigLevelId: levelData.LevelId,
			WeatherIdx: weatherIdx,
		}
	}

	err := h.SaveDB()
	if err != nil {
		return err
	}

	return nil
}

func (h *ChapterHandler) TryIncrMappointEvents(levelId, niwaId int32) []*cmd.MappointEvent {
	var (
		incrEvents      = make([]*cmd.MappointEvent, 0)
		incrEventGroups = make([]*cmd.MappointEventGroupInfo, 0)
	)

	// 无条件, 直接生成
	eachEvents0, eachEventGroups0 := h.TryIncrMappointEventByType(
		levelId, niwaId,
		common.MAPPOINT_EVENT_UPDATE_TYPE_0, make([]string, 0))
	incrEvents = append(incrEvents, eachEvents0...)
	incrEventGroups = append(incrEventGroups, eachEventGroups0...)

	// 完成指定的任务, 则可生成
	eachEvents2, eachEventGroups2 := h.TryIncrMappointEventByType(
		levelId, niwaId,
		common.MAPPOINT_EVENT_UPDATE_TYPE_2, myUtils.Convert2StrList(h.actor.QuestHandler.GetCompleteQuestIds()))
	incrEvents = append(incrEvents, eachEvents2...)
	incrEventGroups = append(incrEventGroups, eachEventGroups2...)

	// 完成指定的一次性事件, 则可生成
	eachEvents5, eachEventGroups5 := h.TryIncrMappointEventByType(
		levelId, niwaId,
		common.MAPPOINT_EVENT_UPDATE_TYPE_5, myUtils.Convert2StrList(h.getFinishOnceEventIds()))
	incrEvents = append(incrEvents, eachEvents5...)
	incrEventGroups = append(incrEventGroups, eachEventGroups5...)

	return incrEvents
}

func (h *ChapterHandler) TryIncrMappointEventByType(
	levelId, niwaId int32,
	incrEventType common.MAPPOINT_EVENT_UPDATE_TYPE,
	params []string) ([]*cmd.MappointEvent, []*cmd.MappointEventGroupInfo) {

	var (
		err        error
		ok         bool
		levelsData *cmd.LS2DB_LevelInfos
		niwaData   *cmd.BattleMapInfo // 当前地图数据

		incrEvents      = make([]*cmd.MappointEvent, 0)
		incrEventGroups = make([]*cmd.MappointEventGroupInfo, 0)
	)

	levelsData = h.actor.GetLevelsData()
	if levelsData == nil {
		return incrEvents, incrEventGroups
	}

	if niwaData, ok = h.GetNiwaData(levelId, niwaId); !ok {
		return incrEvents, incrEventGroups
	}

	incrEvents, incrEventGroups = chapter.GetMappointEvents(niwaData.NiwaId, niwaData.MappointEvents, h.getFinishOnceEventIds(), incrEventType, params)
	niwaData.MappointEvents = append(niwaData.MappointEvents, incrEvents...)
	niwaData.MappointEventGroupInfo = append(niwaData.MappointEventGroupInfo, incrEventGroups...)

	err = h.SaveDB()
	if err != nil {
		return incrEvents, incrEventGroups
	}

	return incrEvents, incrEventGroups
}

func (h *ChapterHandler) IncrCurrNiwaMappointEvents(incrEventType common.MAPPOINT_EVENT_UPDATE_TYPE, params []string) []*cmd.MappointEvent {
	var (
		//err       error
		ok        bool
		levelData *cmd.LS2DB_LevelInfo
		//niwaData  *cmd.BattleMapInfo // 当前地图数据

		incrEvents = make([]*cmd.MappointEvent, 0)
		//incrEventGroups = make([]*cmd.MappointEventGroupInfo, 0)
	)

	if levelData, ok = h.GetCurrLevelData(); !ok {
		return incrEvents
	}

	// 完成指定任务, 则可生成
	eachEvents2, _ := h.TryIncrMappointEventByType(
		levelData.LevelId, levelData.CurrNiwaId,
		incrEventType, params)
	incrEvents = append(incrEvents, eachEvents2...)
	//incrEventGroups = append(incrEventGroups, eachEventGroups2...)

	return incrEvents
}

// CheckFinishEventCondition 检查完成事件的前置条件
func (h *ChapterHandler) CheckFinishEventCondition(niwaData *cmd.BattleMapInfo, eventId int32) (error, cmd.ErrorCode) {
	eventCfg := data.GetMappointEventMgr().GetById(eventId)
	if eventCfg == nil {
		return fmt.Errorf("配表错误, eventId=%d 配置不存在", eventId), cmd.ErrorCode_NotFoundConfig
	}

	// 该事件是否完成过
	if myUtils.ArrayContain(niwaData.FinishedEventIds, eventId) {
		return fmt.Errorf("事件已经完成过了, uid=%s, roleId=%d, currNiwa:%+v, finishEventId:%+v",
			h.actor.uid, h.actor.GetUserData().Common.RoleId, niwaData, eventId), cmd.ErrorCode_Chapter_event_had_done
	}

	// 前置事件是否完成
	for _, condEventId := range eventCfg.TriggerCondEventids {
		if !myUtils.ArrayContain(niwaData.FinishedEventIds, condEventId) {
			return fmt.Errorf("前置事件还未完成, uid=%s, roleId=%d, currNiwa:%+v, eventId=%d, condEventId:%+v",
				h.actor.uid, h.actor.GetUserData().Common.RoleId, niwaData, eventId, condEventId), cmd.ErrorCode_Chapter_event_cond_event_not_finish
		}
	}

	// 前置flag是否完成
	for _, condFlag := range eventCfg.TriggerCondFlags {
		if !h.actor.StoryFlagHandler.checkExistFlags(condFlag) {
			return fmt.Errorf("前置flag还未完成, uid=%s, roleId=%d, currNiwa:%+v, eventId=%d, condFlag:%+v",
				h.actor.uid, h.actor.GetUserData().Common.RoleId, niwaData, eventId, condFlag), cmd.ErrorCode_Chapter_event_cond_flag_not_finish
		}
	}

	return nil, cmd.ErrorCode_Success
}

// 获取card的最小血量
func getCardMinHp(canDie bool) int32 {
	if canDie {
		return 0 // 致死的最小血量
	}
	return 1 // 非致死的最小血量
}

//// 获取db中保存的卡牌能量
//func getOldCardEner(levelData *cmd.LS2DB_LevelInfo, cardId uint32) uint32 {
//	if levelData.PlayerLevelData == nil {
//		return 0
//	}
//
//	for _, card := range levelData.PlayerLevelData.BattleCards {
//		if card.CardId == cardId {
//			return card.CardEner
//		}
//	}
//
//	return 0
//}

// NeedWithholdStamina 是否需要预扣体力
func NeedWithholdStamina(levelType common.LEVEL_TYPE) bool {
	return levelType == common.CHAPTER_LEVEL_TYPE_SUB // 只针对副本
}

// Dto2PClientLevelSummary [PServerLevelSummary 转换成 PClientLevelSummary]
// @param levelId 传值表示获取指定的关卡数据
// @param eventId 传值表示获取指定关卡的指定事件(传值时, levelId应该赋值)
func (h *ChapterHandler) Dto2PClientLevelSummary(levelId int32, eventId int32) *cmd.PClientLevelSummary {
	var (
		levelsData       *cmd.LS2DB_LevelInfos
		levelSummaryList = make([]*cmd.LevelSummary, 0)
	)
	levelsData = h.actor.GetLevelsData()
	serverSummary := levelsData.PLevelSummary

	if levelId > 0 {
		if simpleInfo, ok := serverSummary.LevelSummaryMap[levelId]; ok {
			for _, monsterEventInfo := range simpleInfo.MonsterList {
				if eventId > 0 && monsterEventInfo.EventId != eventId {
					continue
				}

				// 事件组信息
				eventGroupInfo, err := h.getEventGroupInfo(levelId, monsterEventInfo.NiwaId, monsterEventInfo.EventId)
				if err != nil {
					h.Errorf(err.Error())
				}

				// 获取地图中的倒计时时间
				monsterEventInfo.NextUpdateSec = eventGroupInfo.NextUpdateSec
			}

			levelSummaryList = append(levelSummaryList, simpleInfo)
		} else {
			h.Warnf("Dto2PClientLevelSummary 不存在摘要数据, levelId:", levelId)
		}

	} else {
		// 获取所有的关卡摘要信息
		levelsData = h.actor.GetLevelsData()
		for _, each := range serverSummary.LevelSummaryMap {
			levelSummaryList = append(levelSummaryList, each)
		}
	}

	clientSummary := &cmd.PClientLevelSummary{
		TickInfos:        make([]*cmd.LevelMonsterTicketInfo, 0),
		LevelSummaryList: levelSummaryList,
	}
	//门票信息
	for _, ticketInfo := range serverSummary.MonsterTicketInfoMap {
		clientSummary.TickInfos = append(clientSummary.TickInfos, ticketInfo)
	}

	return clientSummary
}

// 每日刷新精英怪/boss门票
func (h *ChapterHandler) dailyRefreshMonsterTicket() {
	var (
	//currencyType cmd.CurrencyType
	)

	// 精英怪/boss门票下次刷新时间戳
	levelsData := h.actor.GetLevelsData()

	for _, ticketInfo := range levelsData.PLevelSummary.MonsterTicketInfoMap {
		// 更新下次刷新时间
		ticketInfo.RefreshTicketSec = common.GetNextDailyRefreshTime()

		// 门票恢复到最大数量
		ticketCount, currencyType, err := h.getMonsterTicket(ticketInfo.MonsterType)
		if err != nil {
			h.Errorf(err.Error())
		}
		addCount := h.getCfgMaxTicketCount(ticketInfo.MonsterType) - ticketCount
		h.Debugf("怪物类型, monsterType=%v, 恢复数量:%v", ticketInfo.MonsterType, addCount)

		if addCount > 0 {
			// 恢复到最大值
			err = h.actor.CurrencyHandler.AddValue(currencyType, int64(addCount), h.actor.comData, common.CR_LEVEL_DAILY_RECOVER)
			if err != nil {
				h.Errorf(err.Error())
			}
		}
	}

	err := h.SaveDB()
	if err != nil {
		h.Errorf("dailyRefreshMonsterTicket saveDB 报错, err:%+v", err)
	}
}

func (h *ChapterHandler) getEventGroupInfo(levelId, niwaId int32, eventId int32) (*cmd.MappointEventGroupInfo, error) {
	var (
		//err        error
		levelsData = h.actor.GetLevelsData()
		//levelData  *cmd.LS2DB_LevelInfo
		niwaData *cmd.BattleMapInfo
		ok       bool
		//rsp        = &cmd.LS2C_DiscoverMonsterRes{}
	)

	//if each, ok := h.GetLevelData(levelId); ok {
	//	levelData = each
	//} else {
	//	return nil, fmt.Errorf("没有对应的关卡数据:%d", levelsData.CurrLevelId)
	//}

	if niwaData, ok = h.GetNiwaData(levelId, niwaId); !ok {
		return nil, fmt.Errorf("没有对应的地图数据:%d", levelsData.CurrLevelId)
	}

	cfg := data.GetMappointEventMgr().GetById(eventId)
	for _, groupInfo := range niwaData.MappointEventGroupInfo {
		if groupInfo.GroupId == cfg.GroupId {
			return groupInfo, nil
		}
	}

	return nil, fmt.Errorf("niwaData=%v, 事件eventId=%d, 没有对应的事件组信息", niwaData, eventId)
}

// 根据当前通关关卡, 获取配表中最大门票数量
func (h *ChapterHandler) getCfgMaxTicketCount(monsterType cmd.LevelMonsterType) int32 {
	var (
		maxCount    int32 = 0
		cfgLimitMap       = make(map[int32]int32) // 配表限制门票最大数量
	)

	switch monsterType {
	case cmd.LevelMonsterType_MonsterType_Boss:
		maxCount = data.GetConfigMgr().GetCfg().LEVEL_TICKET_BOSS_DEFAULT
		cfgLimitMap = data.GetConfigMgr().GetCfg().LEVEL_TICKET_BOSS_LIMIT
	case cmd.LevelMonsterType_MonsterType_Elite:
		maxCount = data.GetConfigMgr().GetCfg().LEVEL_TICKET_MONSTER_DEFAULT
		cfgLimitMap = data.GetConfigMgr().GetCfg().LEVEL_TICKET_MONSTER_LIMIT
	default:
		h.Infof("尚未支持的类型, monsterType=%v", monsterType)
	}

	levelsData := h.actor.GetLevelsData()
	for _, summary := range levelsData.PLevelSummary.LevelSummaryMap {
		if summary.LevelSimpleInfo.HistoryHadPassed != cmd.HistoryHadPassed_PLevelStatus_Passed {
			// 未通关
			continue
		}

		if limitCount, ok := cfgLimitMap[summary.LevelId]; ok && maxCount < limitCount {
			maxCount = limitCount // 有配置, 并且当前最大次数小于配表限制次数
		}
	}

	return maxCount
}

// 根据怪物类型, 获取门票数量
func (h *ChapterHandler) getMonsterTicket(monsterType cmd.LevelMonsterType) (int32, int32, error) {
	var (
		err          error
		currencyType int32
	)

	switch monsterType {
	case cmd.LevelMonsterType_MonsterType_Elite:
		currencyType = common.CURRENCY_ITEM_ID_2010

	case cmd.LevelMonsterType_MonsterType_Boss:
		currencyType = common.CURRENCY_ITEM_ID_2011

	default:
		return 0, 0, fmt.Errorf("未支持的类型, monsterType=%v", monsterType)
	}

	currTicket, err := h.actor.CurrencyHandler.GetValue(currencyType)
	if err != nil {
		h.Errorf(err.Error())
	}

	h.Debugf("怪物类型, monsterType=%v, 当前门票数量:%v", monsterType, currTicket.Value)

	return int32(currTicket.Value), currencyType, nil
}

func (h *ChapterHandler) incrMonsterMaxTicketCount(monsterTicketInfoMap map[int32]*cmd.LevelMonsterTicketInfo, commonData *clidto.Comdata) error {
	var (
		monsterTypes = []cmd.LevelMonsterType{
			cmd.LevelMonsterType_MonsterType_Elite,
			cmd.LevelMonsterType_MonsterType_Boss,
		}
	)

	for _, monsterType := range monsterTypes {
		cfgMaxTicketCount := h.getCfgMaxTicketCount(monsterType)
		if ticketInfo, ok := monsterTicketInfoMap[int32(monsterType)]; !ok {
			//h.Errorf("没有对应的门票数据, monsterTicketInfoMap=%v, monsterType=%d",
			//	monsterTicketInfoMap, cmd.LevelMonsterType_MonsterType_Elite)
			monsterTicketInfoMap[int32(monsterType)] = &cmd.LevelMonsterTicketInfo{
				MonsterType:      monsterType,
				RefreshTicketSec: common.GetNextDailyRefreshTime(),
				MaxCount:         cfgMaxTicketCount,
			}
		} else {
			incrCount := cfgMaxTicketCount - ticketInfo.MaxCount
			if incrCount > 0 {
				// 更新最大值
				ticketInfo.MaxCount += incrCount

				// 累增当前拥有门票数量
				_, currencyType, err := h.getMonsterTicket(monsterType)
				if err != nil {
					return err
				}
				err = h.actor.CurrencyHandler.AddValue(currencyType, int64(incrCount), commonData, common.CR_LEVEL_INCR_MAX)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetCurrLevelId 获取当前攻打的关卡id
func (h *ChapterHandler) GetCurrLevelId() int32 {
	levelsData := h.actor.GetLevelsData()
	if levelsData == nil {
		return 0
	}

	return levelsData.CurrLevelId
}

func (h *ChapterHandler) getFinishOnceEventIds() []int32 {
	var (
		eventIds = make([]int32, 0)
	)
	for eventId, _ := range h.actor.GetLevelsData().FinishedOnceEvents {
		eventIds = append(eventIds, eventId)
	}

	return eventIds
}

func (h *ChapterHandler) enterNiwa(levelId, niwaId int32) (*cmd.BattleMapInfo, error, cmd.ErrorCode) {
	// 地图所在的关卡数据
	levelData, ok := h.GetLevelData(levelId)
	if !ok {
		return nil, errors.New(fmt.Sprintf("还未创建关卡数据, levelId=%d, niwaId=%d", levelId, niwaId)), cmd.ErrorCode_Chapter_create_niwa_fail
	}

	if niwaId == 0 {
		niwaId = levelData.CurrNiwaId // 没有指定地图id, 默认最后次进的地图
		h.Debugf("没有指定地图id, 默认进入最后次进的地图, levelId=%d, niwaId=%d", levelId, niwaId)
	}

	// 获取地图数据
	niwaData, ok := h.GetNiwaData(levelData.LevelId, niwaId)
	if !ok {
		// 生成地图信息
		battleNiwa, err := chapter.NewBattleNiwa(levelId, niwaId)
		if err != nil {
			return nil, err, cmd.ErrorCode_Chapter_create_niwa_fail
		} else {
			niwaData = battleNiwa.FormatNiWa2Proto()
		}
		// 保存地图数据
		levelData.MapInfos[niwaData.NiwaId] = niwaData
	}

	// 当前的地图id
	levelData.CurrNiwaId = niwaData.NiwaId

	// 更新地图事件
	h.TryIncrMappointEvents(levelData.LevelId, levelData.CurrNiwaId)
	// 刷新地图事件
	h.updateNiwaEventGroup(niwaData)

	err := h.SaveDB(false)
	if err != nil {
		return nil, err, cmd.ErrorCode_SaveDBError
	}

	return niwaData, nil, cmd.ErrorCode_Success
}

func (h *ChapterHandler) UpdateUseFoods(levelData *cmd.LS2DB_LevelInfo, useFoods []*cmd.KeyValueItem) bool {
	if len(useFoods) == 0 {
		return false
	}

	if !h.IsInSubLevel() { // 在大地图中不做限制
		return true
	}
	if levelData.PlayerLevelData == nil {
		levelData.PlayerLevelData = &cmd.PlayerLevelData{
			BattleCards: make([]*cmd.PPlayerBattleCard, 0),
			Foods:       make([]int32, 0),
			UseFoods:    make([]*cmd.KeyValueItem, 0),
		}
	}

	if len(levelData.PlayerLevelData.UseFoods) == 0 {
		levelData.PlayerLevelData.UseFoods = useFoods
		return true
	}

	tmp := make(map[int32]int32, len(useFoods))
	for _, item := range levelData.PlayerLevelData.UseFoods {
		tmp[item.GetKey()] = item.GetValue()
	}
	maxFoodNum := excel.GetConfigMgr().GetCfg().FOOD_BATTLEUSE_LIMIT

	//判断原来是否有，有就累加
	for _, item := range useFoods {
		if value, ok := tmp[item.GetKey()]; ok { // 有就累加
			value += item.GetValue()
			//判断是否超过最大次数
			if value > maxFoodNum {
				return false
			}
			tmp[item.GetKey()] = value
		} else {
			tmp[item.GetKey()] = item.GetValue()
		}
	}

	// 重新复制

	newUseFoods := make([]*cmd.KeyValueItem, 0)
	for key, value := range tmp {
		newUseFoods = append(newUseFoods, &cmd.KeyValueItem{
			Key:   key,
			Value: value,
		})
	}

	levelData.PlayerLevelData.UseFoods = newUseFoods

	if err := h.SaveDB(); err != nil {
		h.Errorf("handleBattleEvent SaveChapterData2DB 报错, err:%+v", err)
	}
	return true
}

func (h *ChapterHandler) HandleEatFood(levelData *cmd.LS2DB_LevelInfo, cardId uint32, foods []*cmd.KeyValueItem) cmd.ErrorCode {
	if levelData.PlayerLevelData == nil {
		levelData.PlayerLevelData = &cmd.PlayerLevelData{
			BattleCards: make([]*cmd.PPlayerBattleCard, 0),
			Foods:       make([]int32, 0),
			UseFoods:    make([]*cmd.KeyValueItem, 0),
		}
	}

	// 判断这个卡片是否阵亡
	//for _, batterCard := range levelData.PlayerLevelData.BattleCards {
	//	if batterCard.CardId != cardId {
	//		continue
	//	}
	//	if batterCard.CardHp <= 0 { // 如果阵亡，要过滤掉没有复活特效的食物
	//		foods = h.FilterFood(foods)
	//	}
	//}

	if len(foods) == 0 {
		return cmd.ErrorCode_InvalidParam
	}

	add := uint32(0)
	for _, item := range foods {
		add += uint32(h.CalFoodRecoveryHPRate(int32(cardId), item.GetKey()) * item.GetValue())
	}

	for _, batterCard := range levelData.PlayerLevelData.BattleCards {
		if batterCard.CardId != cardId {
			continue
		}
		card, err := h.actor.CardHandler.GetCard(batterCard.CardId)
		if err != nil {
			h.Errorf(err.Error())
			continue
		}
		batterCard.CardHp = myUtils.Min(batterCard.CardHp+add, card.OldMaxHp)
	}
	return cmd.ErrorCode_Success
}

// CalFoodRecoveryHPRate 食物使用效率的计算
func (h *ChapterHandler) CalFoodRecoveryHPRate(cardId, foodId int32) int32 {
	// 卡牌配置表
	cfg := excel.GetBeastarMgr().GetById(cardId)
	if cfg == nil {
		return 0
	}
	card, err := h.actor.CardHandler.GetCard(uint32(cardId))
	if err != nil {
		return 0
	}

	addValue := int32(0)
	addPercent := int32(0)
	// 突破固定值
	evolutionCfg := excel.GetEvolutionMgr().GetById(cardId*100 + int32(card.BreakthroughLevel))
	if evolutionCfg != nil {
		addValue += evolutionCfg.UltraAppetite
	}
	h.Debugf("使用食物回复血量,突破百分比:", addPercent, ",突破固定值:", addValue)
	// 基础值 cfg.GetAppetite()

	// 潜力
	cardAwakenId := cfg.Potential*100 + int32(card.AwakenLevel)
	awakenCfg := excel.GetPotentialMgr().GetById(cardAwakenId)
	if awakenCfg != nil {
		if upValue, ok := awakenCfg.GetUpAtt()[int32(7)]; ok {
			switch awakenCfg.AbiType {
			case 1: // 绝对值
				addValue += upValue
			case 2: // 百分比
				addPercent += upValue
			}
		}
	}
	h.Debugf("使用食物回复血量,潜力百分比:", addPercent, ",潜力固定值:", addValue)
	// 性格
	id := cardId*100 + int32(card.CharacterLevel)
	if characterCfg := excel.GetCharacterMgr().GetById(id); characterCfg != nil {
		tempAbi := getCharacterAbi(card, characterCfg)
		if tempAbi > 0 {
			abiCfg := excel.GetCharacterAbiMgr().GetById(tempAbi)
			if abiCfg != nil {
				if upValue, ok := abiCfg.GetUpAtt()[int32(7)]; ok {
					switch abiCfg.AbiType {
					case 1: // 绝对值
						addValue += upValue
					case 2: // 百分比
						addPercent += upValue
					}
				}
			}
		}
	}
	h.Debugf("使用食物回复血量,性格百分比:", addPercent, ",性格固定值:", addValue)
	foodCfg := excel.GetFoodMgr().GetById(foodId)
	if foodCfg == nil {
		return 0
	}
	// 计算食量
	percent := float32(1) + float32(addPercent)/float32(100)
	appetite := float32(cfg.Appetite)*percent + float32(addValue)
	h.Debugf("使用食物回复血量,计算食量:", appetite)

	// 道具恢复量*（1+食量/1000）
	add := float32(foodCfg.GetRestoreval()) * (float32(1) + appetite/float32(1000))
	h.Debugf("使用食物回复血量,道具恢复量:", float32(foodCfg.GetRestoreval()), ",(1+食量/1000):", float32(1)+appetite/float32(1000), ",结果:", add)
	return int32(add)
}

func (h *ChapterHandler) FilterFood(oldFood []*cmd.KeyValueItem) []*cmd.KeyValueItem {
	effFoods := make([]*cmd.KeyValueItem, 0)
	for _, food := range oldFood {
		foodCfg := excel.GetFoodMgr().GetById(food.GetKey())
		if foodCfg == nil {
			continue
		}
		if foodCfg.Effecttype == 1 {
			effFoods = append(effFoods, &cmd.KeyValueItem{
				Key:   food.GetKey(),
				Value: food.GetValue(),
			})
		}
	}
	return effFoods
}

func (h *ChapterHandler) FoodIsExist(levelData *cmd.LS2DB_LevelInfo, foodId int32) bool {
	if !h.IsInSubLevel() { // 不在副本中，就不需要判断
		return true
	}
	// 在副本中
	for _, fId := range levelData.PlayerLevelData.Foods {
		if fId == foodId {
			return true
		}
	}
	return false
}

func (h *ChapterHandler) CheckCardExist(levelData *cmd.LS2DB_LevelInfo, cardId int32) bool {
	// 卡片是否再编队中
	for _, card := range levelData.PlayerLevelData.BattleCards {
		if int32(card.CardId) == cardId {
			return true
		}
	}
	// 判断卡片是否是满血
	return false
}

func (h *ChapterHandler) CheckCardFullHp(levelData *cmd.LS2DB_LevelInfo, cardId int32) bool {
	for _, bCard := range levelData.PlayerLevelData.BattleCards {
		if int32(bCard.CardId) == cardId {
			card, err := h.actor.CardHandler.GetCard(bCard.CardId)
			if err != nil {
				h.Errorf(err.Error())
				continue
			}
			if bCard.CardHp < card.OldMaxHp {
				return true
			}
		}
	}
	return false
}

// 完成任务时消耗材料
func (h *ChapterHandler) tryEventCostItem(eventCfg *excel.MappointEventCfg) (error, cmd.ErrorCode) {
	if eventCfg == nil {
		return errors.New("没有对应配置"), cmd.ErrorCode_NotFoundConfig
	}

	if cmd.SceneEventType(eventCfg.EventType) != cmd.SceneEventType_SceneEventType_costItem {
		h.Debugf("tryEventCostItem 事件类型:%d, 不做消耗处理", eventCfg.EventType)
		return nil, cmd.ErrorCode_Success
	}

	costItemMap := eventCfg.ItemSubmitGroupNum

	if len(costItemMap) == 0 {
		return nil, cmd.ErrorCode_Success // 不需要消耗
	}

	// 扣除提交的材料
	if !GetConsumeMgr(h.actor).CheckMapEnough(costItemMap) {
		return fmt.Errorf("item not enough"), cmd.ErrorCode_NotEnoughItem
	}
	err := GetConsumeMgr(h.actor).ConsumeList(costItemMap, h.actor.comData, common.CR_Finish_Event_Submit)
	if err != nil {
		return err, cmd.ErrorCode_InternalError
	}

	return nil, cmd.ErrorCode_Success
}

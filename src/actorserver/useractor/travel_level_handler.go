package useractor

import (
	"context"
	"fmt"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"

	"gitlab.musadisca-games.com/wangxw/musae/framework/base"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type TravelLevelHandler struct {
	*UABaseHandler
}

func NewTravelLevelHandler(actor *UserActor) *TravelLevelHandler {
	h := &TravelLevelHandler{UABaseHandler: NewUABaseHandler(actor, "TravelLevelHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TravelEnterLevelReq), h.TravelEnterLevelReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TravelBattleStartReq), h.TravelBattleStartReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TravelBattleEndReq), h.TravelBattleEndReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TravelStoryEndReq), h.FinishTravelStoryReq) // 完成剧情关卡

	return h
}

// Init 初始化模块数据
func (h *TravelLevelHandler) Init() error {
	// 初始化
	h.actor.Data.TravelLevelData = &cmd.PUserTravelLevelData{
		Createtime:             time.Now().Unix(),
		PassedTravelLevelDatas: make([]*cmd.PassedTravelLevelData, 0),
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init TravelLevel data success.")
	return nil
}

func (h *TravelLevelHandler) EnterGame() error {
	return nil
}

func (h *TravelLevelHandler) DailyRefresh() error {
	return nil
}

func (h *TravelLevelHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUserTravelLevelData); ok {
		h.actor.Data.TravelLevelData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *TravelLevelHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserTravelLevel(h.actor.ID()), h.actor.Data.TravelLevelData
}

func (h *TravelLevelHandler) TravelEnterLevelReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_TravelEnterLevelReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	// 检查是否为旅途关卡
	if err, errCode := checkIsTravelLevel(req.LevelId); err != nil {
		return nil, err, int32(errCode)
	}

	// 检查前置关卡是否通关
	if err, errCode := h.checkTravelLevelPreCondition(req.LevelId); err != nil {
		return nil, err, int32(errCode)
	}

	levelCfg := data.GetTravelEventMgr().GetById(req.LevelId)

	res := &cmd.LS2C_TravelEnterLevelRes{
		EventIds: []int32{levelCfg.EventId},
	}

	return res, nil, int32(cmd.ErrorCode_Success)
}

func (h *TravelLevelHandler) TravelBattleStartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_TravelBattleStartReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	battleId := uint64(utils.GenIntUUID())
	rseed := utils.GenIntUUID()

	travelLevelData := h.getTravelLevelData()
	travelLevelData.BattleId = battleId
	travelLevelData.BattleRandomSeed = rseed

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	res := &cmd.LS2C_TravelBattleStartRes{
		BattleId:         battleId,
		BattleRandomSeed: rseed,
		CommonData:       h.actor.comData.FixDownComData(),
	}

	return res, nil, int32(cmd.ErrorCode_Success)
}

func (h *TravelLevelHandler) TravelBattleEndReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_TravelBattleEndReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	// 检查是否为旅途关卡
	if err, errCode := checkIsTravelLevel(req.LevelId); err != nil {
		return nil, err, int32(errCode)
	}

	// 检查前置关卡是否通关
	if err, errCode := h.checkTravelLevelPreCondition(req.LevelId); err != nil {
		return nil, err, int32(errCode)
	}

	// 游戏端结果为战斗失败
	if req.BattleResult != cmd.BattleResult_BattleResult_Winer {
		travelLevelData := h.getTravelLevelData()
		travelLevelData.BattleId = 0
		travelLevelData.BattleRandomSeed = 0

		res := &cmd.LS2C_TravelBattleEndRes{
			CommonData:   h.actor.comData.FixDownComData(),
			BattleResult: req.BattleResult,
		}
		return res, nil, int32(cmd.ErrorCode_Success)
	}

	travelEventCfg := data.GetTravelEventMgr().GetById(req.LevelId)

	if h.CheckTravelLevelHadPassed(req.LevelId) && //已经通关了
		travelEventCfg.Isonce == 1 { // 一次性关卡
		return nil, errors.New(fmt.Sprintf("旅途关卡已经打过了, levelId=%d", req.LevelId)), int32(cmd.ErrorCode_Travel_level_had_passed)
	}

	if !h.actor.PlayerLevelHandler.CheckStaminaEnough(travelEventCfg.StaminaCost) {
		return nil, errors.New("体力不足"), int32(cmd.ErrorCode_StaminaValueNotEnough)
	}

	travelLevelData := h.getTravelLevelData()

	// 战斗校验
	levelsData := h.actor.Data.LevelsData
	selfBattleTeam := h.actor.BattleHandler.buildSelfBattleCards(cmd.CardTroopType_CardTroopType_Normal, levelsData.TroopId, nil)
	checkBattle, err, errCode := h.actor.BattleHandler.CheckBattle(travelLevelData.BattleId, travelLevelData.BattleRandomSeed, req.BattleResult,
		selfBattleTeam,
		travelEventCfg.EventId,
		req.BattleFrameData, req.VersionData)
	if err != nil {
		return nil, err, int32(errCode)
	}
	if checkBattle != nil && (checkBattle.CheckBattleResult == cmd.CheckBattleResult_CBR_fail || checkBattle.BattleResult != req.BattleResult) {
		return nil, errors.New("校验失败"), int32(cmd.ErrorCode_CheckBattle_fail)
	}

	if checkBattle == nil {
		// 校验关闭或其他情况没有返回值, 使用前端结果
		checkBattle = &cmd.CheckBattleRes{
			CheckBattleResult: cmd.CheckBattleResult_CBR_success,
			BattleResult:      req.BattleResult,
			//SelfCards:         battleEventReq.Card,
			OppoCards: nil,
			CostFoods: req.CostFoods,
		}
	}

	// 消耗食物
	if len(checkBattle.CostFoods) > 0 {
		if !GetConsumeMgr(h.actor).CheckKeyValItemEnough(checkBattle.CostFoods) {
			return nil, err, int32(cmd.ErrorCode_FoodNotEnough)
		}
		err = GetConsumeMgr(h.actor).ConsumeKeyValItemList(checkBattle.CostFoods, h.actor.comData, common.CR_COST_BY_BATTLE_TRAVEL_LEVEL)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_FoodNotEnough)
		}
	}

	// 扣除副本需要消耗的体力
	if travelEventCfg.StaminaCost > 0 {
		err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{common.ITEM_ID_STAMINA_1004: travelEventCfg.StaminaCost}, h.actor.comData, common.CR_COST_BY_BATTLE_TRAVEL_LEVEL)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_StaminaValueNotEnough)
		}
	}

	onceDropChange, dropChange := h.doExitLevel(req.LevelId, h.actor.comData, selfBattleTeam.CardList)

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	res := &cmd.LS2C_TravelBattleEndRes{
		BattleResult:   checkBattle.BattleResult,
		OnceDropChange: onceDropChange,
		DropChange:     dropChange,
		CommonData:     h.actor.comData.FixDownComData(),
	}

	return res, nil, int32(cmd.ErrorCode_Success)
}

func (h *TravelLevelHandler) FinishTravelStoryReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_TravelStoryEndReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	// 检查是否为旅途关卡
	if err, errCode := checkIsTravelLevel(req.LevelId); err != nil {
		return nil, err, int32(errCode)
	}

	// 检查是否为剧情关卡
	travelEventCfg := data.GetTravelEventMgr().GetById(req.LevelId)
	if travelEventCfg.EventId != 0 {
		return nil, errors.New(fmt.Sprintf("不是剧情关卡, levelId=%d", req.LevelId)), int32(cmd.ErrorCode_Travel_level_not_travel_story_level)
	}

	// 检查前置关卡是否通关
	if err, errCode := h.checkTravelLevelPreCondition(req.LevelId); err != nil {
		return nil, err, int32(errCode)
	}

	onceDropChange, dropChange := h.doExitLevel(req.LevelId, nil, nil)

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	res := &cmd.LS2C_TravelStoryEndRes{
		OnceDropChange: onceDropChange,
		DropChange:     dropChange,
		CommonData:     h.actor.comData.FixDownComData(),
	}

	return res, nil, int32(cmd.ErrorCode_Success)
}

// GetPassedTravelLevel 获取通关的旅途关卡数据
func (h *TravelLevelHandler) GetPassedTravelLevel(travelLevelId int32) *cmd.PassedTravelLevelData {
	for _, travelLevel := range h.getTravelLevelData().PassedTravelLevelDatas {
		if travelLevel.LevelId == travelLevelId {
			return travelLevel
		}
	}

	return nil
}

// CheckTravelLevelHadPassed 检查旅途关卡是否通关
func (h *TravelLevelHandler) CheckTravelLevelHadPassed(travelLevelId int32) bool {
	travelLevel := h.GetPassedTravelLevel(travelLevelId)
	return travelLevel != nil && travelLevel.PassedCount > 0
}

func (h *TravelLevelHandler) checkTravelLevelPreCondition(levelId int32) (error, cmd.ErrorCode) {
	travelEventCfg := data.GetTravelEventMgr().GetById(levelId)

	// 大地图关卡是否完成
	if travelEventCfg.PreMainLevel > 0 {
		isMainLevelPassed := h.actor.ChapterHandler.CheckMainLevelHadPassed(travelEventCfg.PreMainLevel)
		if !isMainLevelPassed {
			return errors.New(fmt.Sprintf("大地图前置关卡没有通关, mainLevelId=%d", travelEventCfg.PreMainLevel)),
				cmd.ErrorCode_Travel_level_pre_main_level_not_passed
		}
	}

	// 旅途关卡是否完成
	if travelEventCfg.TravelCondition > 0 {
		isTravelLevelPassed := h.CheckTravelLevelHadPassed(travelEventCfg.TravelCondition)
		if !isTravelLevelPassed {
			return errors.New(fmt.Sprintf("旅途前置关卡没有通关, mainLevelId=%d", travelEventCfg.PreMainLevel)),
				cmd.ErrorCode_Travel_level_pre_travel_level_not_passed
		}
	}

	return nil, cmd.ErrorCode_Success
}

func (h *TravelLevelHandler) firstPass(levelId int32) *cmd.DropChange {
	var (
		err            error
		onceAddRewards = &cmd.DropChange{} // 通关基础奖励 - 增量
		nowSec         = time.Now().Unix()
	)
	campaignCfg := data.GetTravelEventMgr().GetById(levelId)

	// 首次通关
	onceAddRewards, err = GetDropMgr(h.actor).DropList2(campaignCfg.RewardOnce, true, nil, h.actor.comData, common.CR_PASS_TRAVEL_LEVEL_ONCE)
	if err != nil {
		h.Debugf("首通奖励报错, 旅途关卡id:%d, err:%+v", campaignCfg.GetId(), err)
	}

	travelLevelData := &cmd.PassedTravelLevelData{
		LevelId:      levelId,
		FirstPassSec: nowSec,
		PassedCount:  1,
	}
	h.actor.Data.TravelLevelData.PassedTravelLevelDatas = append(h.actor.Data.TravelLevelData.PassedTravelLevelDatas, travelLevelData)

	// 下发给客户端数据
	h.actor.comData.AddTravelLevelData(travelLevelData)

	return onceAddRewards
}

func (h *TravelLevelHandler) doExitLevel(levelId int32, commonData *clidto.Comdata, cards []*cmd.BattleCard) (*cmd.DropChange, *cmd.DropChange) {
	var (
		onceDropChange = &cmd.DropChange{}
		dropChange     = &cmd.DropChange{}
	)

	if !h.CheckTravelLevelHadPassed(levelId) {
		// 首次通关
		_dropChange := h.firstPass(levelId)
		mergeDropChange(onceDropChange, _dropChange, true) // 首次通关奖励
	} else {
		// 记录通关信息
		travelLevel := h.GetPassedTravelLevel(levelId)
		travelLevel.PassedCount += 1
	}

	// 基础奖励
	baseRewards := datahelper.GetTravelLevelBaseRewards(levelId)
	_dropChange, err := GetDropMgr(h.actor).DropList2(baseRewards, true, nil, h.actor.comData, common.CR_PASS_TRAVEL_LEVEL_BASE)
	travelCfg := data.GetTravelEventMgr().GetById(levelId)
	if err != nil {
		h.Debugf("奖励报错, 旅途关卡id:%d, err:%+v", travelCfg.GetId(), err)
	}
	mergeDropChange(dropChange, _dropChange, true) // 首通奖励
	h.Debugf("旅途关卡通关奖励:%v", dropChange)

	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_TRAVEL_LEVEL_WIN, []int32{TASK_TYPE_507}, map[string]interface{}{
		"type": travelCfg.TravelEventType,
	}))
	if errx != nil {
		h.Error(errx)
	}
	//增加羁绊值
	cardIds := GeBattleCardCardId(cards)
	if len(cards) > 0 {
		h.actor.UserRelationHandler.AddRelation(cardIds, commonData, common.Realtion_type_win)
		h.Debugf("旅途关卡增加羁绊值:%v", dropChange)
	}

	return onceDropChange, dropChange
}

func (h *TravelLevelHandler) getTravelLevelData() *cmd.PUserTravelLevelData {
	return h.actor.Data.TravelLevelData
}

func (h *TravelLevelHandler) GmExitTravelLevel(travelLevelId int32) {
	h.doExitLevel(travelLevelId, nil, nil)
	h.Debugf("gm 通关旅途关卡, travelLevelId=%d", travelLevelId)
}

func checkIsTravelLevel(travelLevelId int32) (error, cmd.ErrorCode) {
	levelCfg := data.GetTravelEventMgr().GetById(travelLevelId)
	if levelCfg == nil {
		return errors.New(fmt.Sprintf("配置数据不存在, levelId=%d", travelLevelId)), cmd.ErrorCode_NotFoundConfig
	}

	if common.LEVEL_TYPE(levelCfg.TravelEventType) != common.CHAPTER_LEVEL_TYPE_TRAVEL {
		return errors.New(fmt.Sprintf("不是旅途关卡, levelId=%d", travelLevelId)), cmd.ErrorCode_Travel_level_is_not
	}

	return nil, cmd.ErrorCode_Success
}

func GeBattleCardCardId(cards []*cmd.BattleCard) []int32 {
	ids := make([]int32, 0)
	for _, v := range cards {
		ids = append(ids, int32(v.CardInfo.Common.CardId))
	}
	return ids
}

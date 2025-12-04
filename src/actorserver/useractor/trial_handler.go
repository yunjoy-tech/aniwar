package useractor

import (
	"context"
	"fmt"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type TrialHandler struct {
	*UABaseHandler
}

func NewTrialHandler(actor *UserActor) *TrialHandler {
	h := &TrialHandler{UABaseHandler: NewUABaseHandler(actor, "TrialHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_InitTrialInfoReq), h.InitTrialInfoReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TrialBattleStartReq), h.BattleStartReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TrialBattleEndReq), h.BattleEndReq)

	return h
}

// Init 初始化模块数据
func (h *TrialHandler) Init() error {
	// 初始化
	h.actor.Data.TrialData = &cmd.PUserTrial{
		Createtime: time.Now().Unix(),
		CurTroop:   0,
		LastUse:    0,
		Trial:      make(map[int32]*cmd.PTrialInfo),
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init trial data success.")
	return nil
}

func (h *TrialHandler) EnterGame() error {
	return nil
}

func (h *TrialHandler) DailyRefresh() error {
	return nil
}

func (h *TrialHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUserTrial); ok {
		h.actor.Data.TrialData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *TrialHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserTrial(h.actor.ID()), h.actor.Data.TrialData
}

// 尝试解锁关卡
func (h *TrialHandler) tryUnlock(e event.IEvent) error {
	questId, ok := e.Get("quest_id").(int)
	if !ok {
		return nil
	}

	firstTrialId, index := getFirstTrialId(int32(questId))
	if firstTrialId == 0 {
		return nil
	}

	trialData := h.actor.GetTrialData()
	// 做一下容错
	if _, ok = trialData.Trial[firstTrialId]; ok {
		return nil
	}

	newTrial := &cmd.PTrialInfo{
		Index:      index,
		CurLevelId: firstTrialId,
		IsNew:      make([]int32, 0),
	}
	trialData.Trial[index] = newTrial
	if err := h.SaveDB(); err != nil {
		h.Errorf("%+v", err)
		return nil
	}

	h.Infof("tryUnlock trial success, data: %+v", newTrial)
	return nil
}

func (h *TrialHandler) InitTrialInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_TRIAL)
	if err != nil {
		return nil, err, int32(code)
	}
	trialData := h.actor.GetTrialData()
	ret := make([]*cmd.PTrialInfo, 0, len(trialData.Trial))
	for _, trial := range trialData.Trial {
		ret = append(ret, trial)
	}
	return &cmd.LS2C_InitTrialInfoRes{
		Trial:   ret,
		LastUse: trialData.LastUse,
	}, nil, 0
}

func (h *TrialHandler) BattleStartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_TRIAL)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_TrialBattleStartReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 编队校验
	if code := h.actor.TroopHandler.CheckTroopTypAndId(int32(cmd.CardTroopType_CardTroopType_Trial), req.TroopId); code != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("troop check failed %d", req.TroopId), int32(code)
	}
	// 关卡校验
	cfg := excel.GetChallengeMgr().GetById(req.LevelId)
	if cfg == nil {
		return nil, fmt.Errorf("cfg not found %d", req.LevelId), int32(cmd.ErrorCode_NotFoundConfig)
	}
	trialData := h.actor.GetTrialData()
	trial, ok := trialData.Trial[cfg.Belong]
	if !ok {
		return nil, fmt.Errorf("trial not unlock %d", cfg.Belong), int32(cmd.ErrorCode_ParamError)
	}
	if trial.CurLevelId != req.LevelId {
		return nil, fmt.Errorf("trial id not match %d", req.LevelId), int32(cmd.ErrorCode_ParamError)
	}

	battleId := uint64(utils.GenIntUUID())
	randSeed := utils.GenIntUUID()

	// 记录数据
	trialData.CurTroop = req.TroopId
	trialData.CurLevelId = req.LevelId
	trialData.CurBattleId = battleId
	trialData.CurRandSeed = randSeed
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 返回
	return &cmd.LS2C_TrialBattleStartRes{BattleId: battleId, BattleRandomSeed: randSeed}, nil, 0
}

func (h *TrialHandler) BattleEndReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_TrialBattleEndReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	trialData := h.actor.GetTrialData()
	// 校验
	if trialData.CurLevelId != req.LevelId {
		return nil, fmt.Errorf("trial id not match %d", req.LevelId), int32(cmd.ErrorCode_ParamError)
	}
	cfg := excel.GetChallengeMgr().GetById(req.LevelId)
	if cfg == nil {
		return nil, fmt.Errorf("cfg not found %d", req.LevelId), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// 胜利了才校验有效性
	var (
		checkBattle *cmd.CheckBattleRes
		errCode     cmd.ErrorCode
	)
	if req.BattleResult == cmd.BattleResult_BattleResult_Winer {
		selfBattleCards := h.actor.BattleHandler.buildSelfBattleCards(cmd.CardTroopType_CardTroopType_Trial, trialData.CurTroop, nil)
		checkBattle, err, errCode = h.actor.BattleHandler.CheckBattle(
			trialData.CurBattleId, trialData.CurRandSeed, req.BattleResult,
			selfBattleCards, cfg.BattleID, req.BattleFrameData, req.VersionData)
		if err != nil {
			return nil, err, int32(errCode)
		}
		// 使用服务器校验结果
		if checkBattle != nil {
			if checkBattle.CheckBattleResult == cmd.CheckBattleResult_CBR_fail || checkBattle.BattleResult != req.BattleResult {
				return nil, fmt.Errorf("check battle failed"), int32(cmd.ErrorCode_CheckBattle_fail)
			}
		} else {
			checkBattle = &cmd.CheckBattleRes{CostFoods: req.CostFoods}
		}
	}

	rsp := &cmd.LS2C_TrialBattleEndRes{}
	if req.BattleResult == cmd.BattleResult_BattleResult_Winer {
		// 胜利了，食物扣除
		costs := myUtils.ConvertItem(checkBattle.CostFoods)
		if !GetConsumeMgr(h.actor).CheckMapEnough(costs) {
			return nil, err, int32(cmd.ErrorCode_FoodNotEnough)
		}
		if err = GetConsumeMgr(h.actor).ConsumeList(costs, h.actor.comData, common.CR_TRIAL); err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}

		// 胜利了，数据记录
		trial, ok := trialData.Trial[cfg.Belong]
		if !ok {
			return nil, fmt.Errorf("trial data not found %d", cfg.Belong), int32(cmd.ErrorCode_NotFoundConfig)
		}
		trial.CurLevelId = getNextTrialId(trial.CurLevelId)
		if cfg.IsStory == 1 {
			trial.IsNew = append(trial.IsNew, cfg.Id)
		}
		rsp.Trial = trial

		// 胜利了，下发奖励
		reward := datahelper.ConvertItem3(cfg.Reward)
		rsp.DropChange, err = GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_TRIAL)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 临时数据清除
	trialData.LastUse = req.LevelId
	trialData.CurLevelId = 0
	trialData.CurBattleId = 0
	trialData.CurRandSeed = 0
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 返回结果
	rsp.BattleResult = req.BattleResult
	rsp.CommonData = h.actor.comData.FixDownComData()
	return rsp, nil, 0
}

// 处理红点标识
func (h *TrialHandler) handleRedPoint(commonData *clidto.Comdata, ids []int64) error {
	trialData := h.actor.GetTrialData()
	var f bool
	for _, id := range ids {
		// 解析规则
		cfg := excel.GetChallengeMgr().GetById(int32(id))
		if cfg == nil {
			continue
		}
		trial, ok := trialData.Trial[cfg.Belong]
		if !ok {
			continue
		}
		// 清除红点
		trial.IsNew = myUtils.DeleteAllByElement(trial.IsNew, int32(id))
		f = true
	}
	if f {
		return h.SaveDB()
	}
	return nil
}

// 获取下一关的id，没有下一关时返回0
func getNextTrialId(curTrialId int32) int32 {
	var trialId int32
	excel.GetChallengeMgr().Foreach(func(cfg *excel.ChallengeCfg) bool {
		if cfg.GetPre() == curTrialId {
			trialId = cfg.GetId()
			return false
		}
		return true
	}, false)
	return trialId
}

// 获取解锁的首关id，没有解锁数据返回0
func getFirstTrialId(questId int32) (int32, int32) {
	var trialId int32
	var index int32
	excel.GetChallengeMgr().Foreach(func(cfg *excel.ChallengeCfg) bool {
		if cfg.GetOpen() == questId {
			trialId = cfg.GetId()
			index = cfg.GetBelong()
			return false
		}
		return true
	}, false)

	return trialId, index
}

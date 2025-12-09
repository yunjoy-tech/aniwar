package useractor

import (
	"context"
	"fmt"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

const (
	EventStatusNotTrigger = 0 // 未触发
	EventStatusDoing      = 1 // 进行中

	EventTypeVisitor = 1 // 访客事件
	EventTypeHelper  = 2 // 商人事件
	EventTypeBoss    = 3 // 怪物事件
)

type BlockWayHandler struct {
	*UABaseHandler
}

func NewBlockWayHandler(actor *UserActor) *BlockWayHandler {
	h := &BlockWayHandler{UABaseHandler: NewUABaseHandler(actor, "BlockWayHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_TriggerEventReq), h.TriggerEventReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ReceiveEventGiftReq), h.ReceiveEventGiftReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_EventBattleStartReq), h.BattleStartReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_EventBattleEndReq), h.BattleEndReq)

	return h
}

// Init 初始化模块数据
func (h *BlockWayHandler) Init() error {
	// 初始化
	h.actor.Data.BlockWayData = &cmd.PBlockWay{
		Createtime: time.Now().Unix(),
		EventList:  make(map[int64]*cmd.PBlockWayEvent),
		EventType:  make(map[int32]*cmd.PBlockWayEventGroup),
		Stamina:    0,
		TriggerNum: excel.GetConfigMgr().GetCfg().ROAD_EVENT_LIMIT,
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init blockway data success.")
	return nil
}

func (h *BlockWayHandler) EnterGame() error {
	return nil
}

func (h *BlockWayHandler) DailyRefresh() error {
	return h.tryRefreshEvent()
}

func (h *BlockWayHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PBlockWay); ok {
		h.actor.Data.BlockWayData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *BlockWayHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserBlockWay(h.actor.ID()), h.actor.Data.BlockWayData
}

func (h *BlockWayHandler) buildBlockWayEvents() []*cmd.PBlockWayEvent {
	blockWayData := h.actor.GetBlockWayData()
	tryClearEvent(blockWayData.EventList)

	ret := make([]*cmd.PBlockWayEvent, 0, len(blockWayData.EventList))
	for _, e := range blockWayData.EventList {
		ret = append(ret, e)
	}
	return ret
}

// 尝试删除无效事件, 返回保留的数量
func tryClearEvent(events map[int64]*cmd.PBlockWayEvent) (int32, map[int32]*cmd.PBlockWayEventGroup) {
	var (
		doingNum  int32
		eventData = make(map[int32]*cmd.PBlockWayEventGroup)
		now       = time.Now().Unix()
	)

	for id, e := range events {
		// 未触发的情况
		if e.State == EventStatusNotTrigger {
			if e.RefreshTime <= now {
				delete(events, id)
				continue
			}
		}
		// 已触发的情况
		if e.State == EventStatusDoing {
			if e.EndTime <= now {
				delete(events, id)
				continue
			}
		}
		doingNum++
		// 记录事件类型计数
		cfg := excel.GetRoadEventMgr().GetById(e.CfgId)
		if cfg == nil {
			continue
		}
		if group, ok := eventData[cfg.EventType]; ok {
			group.Trigger++
		} else {
			group = &cmd.PBlockWayEventGroup{
				GroupId: cfg.EventType,
				Trigger: 1,
			}
			eventData[cfg.EventType] = group
		}
	}

	return doingNum, eventData
}

// 跨天逻辑处理
func (h *BlockWayHandler) tryRefreshEvent() error {
	blockWayData := h.actor.GetBlockWayData()
	costNum, eventData := tryClearEvent(blockWayData.EventList)
	blockWayData.EventType = eventData
	blockWayData.Stamina = 0
	blockWayData.TriggerNum = excel.GetConfigMgr().GetCfg().ROAD_EVENT_LIMIT - costNum

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Infof("tryRefreshEvent success. events: %+v", blockWayData.EventList)
	return nil
}

// 体力消耗事件处理
func (h *BlockWayHandler) tryTriggerEvent(e event.IEvent) error {
	// 是否解锁
	err, _ := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_BLOCKWAY)
	if err != nil {
		return nil
	}

	// 累计体力消耗值
	val, ok := e.Get("count").(int32)
	if !ok {
		h.Warnf("tryTriggerEvent get val failed.")
		return nil
	}

	blockWayData := h.actor.GetBlockWayData()
	if blockWayData.TriggerNum <= 0 {
		return nil
	}
	blockWayData.Stamina += val

	// 计算刷新次数
	var tryNum = blockWayData.TriggerNum
	calNum := blockWayData.Stamina / excel.GetConfigMgr().GetCfg().ROAD_EVENT_COST
	if calNum < tryNum {
		tryNum = calNum
	}
	// 尝试触发事件
	for i := int32(0); i < tryNum; i++ {
		h.handleTriggerEvent(blockWayData)
	}

	if err := h.SaveDB(); err != nil {
		h.Error(err)
	}
	return nil
}

// 随机触发事件逻辑
func (h *BlockWayHandler) handleTriggerEvent(blockWayData *cmd.PBlockWay) {
	var (
		e            *cmd.PBlockWayEvent
		tempType     = make(map[interface{}]int32)
		tempEvent    = make(map[interface{}]int32)
		tempQuestIds = make(map[int32]int32)
	)
	// 随机事件类型
	excel.GetRoadConfigMgr().Foreach(func(cfg *excel.RoadConfigCfg) bool {
		group := blockWayData.EventType[cfg.Type]
		if group != nil && group.Trigger >= cfg.Limit {
			return true
		}
		tempType[cfg.Type] = cfg.Rate
		return true
	}, true)
	// 容错
	if len(tempType) == 0 {
		h.Warnf("handleTriggerEvent config error, data:%+v", blockWayData.EventType)
		return
	}
	typeCfg := myUtils.RandomMap(tempType, true)
	if _, ok := typeCfg.(int32); !ok {
		h.Warnf("handleTriggerEvent assert failed. target:%v, weightMap:%+v", typeCfg, tempType)
		return
	}
	targetType := typeCfg.(int32)

	// 确定触发条件（剧情任务id）
	excel.GetRoadEventMgr().Foreach(func(cfg *excel.RoadEventCfg) bool {
		if cfg.EventType != targetType {
			return true
		}
		tempQuestIds[cfg.Trigger] = 0
		return true
	}, true)

	if len(tempQuestIds) == 0 {
		h.Warnf("handleTriggerEvent type not found. type: %d", targetType)
		return
	}
	var lastQuestId int32 /*h.actor.QuestHandler.GetBlockTriggerId(tempQuestIds)*/

	// 随机掉落事件
	excel.GetRoadEventMgr().Foreach(func(cfg *excel.RoadEventCfg) bool {
		if cfg.EventType != targetType {
			return true
		}
		if cfg.Trigger != lastQuestId {
			return true
		}
		tempEvent[cfg] = cfg.Rate
		return true
	}, true)

	// 容错
	if len(tempEvent) == 0 {
		h.Warnf("handleTriggerEvent config error, type:%d, questId:%d", targetType, lastQuestId)
		return
	}
	eventCfg := myUtils.RandomMap(tempEvent, true)
	if _, ok := eventCfg.(*excel.RoadEventCfg); !ok {
		h.Warnf("handleTriggerEvent assert failed. target:%v, weightMap:%+v", eventCfg, tempEvent)
		return
	}
	targetEvent := eventCfg.(*excel.RoadEventCfg)
	e = h.createEvent(targetEvent)

	// 记录触发事件的数据
	if group, ok := blockWayData.EventType[targetType]; ok {
		group.Trigger++
	} else {
		group = &cmd.PBlockWayEventGroup{
			GroupId: targetType,
			Trigger: 1,
		}
		blockWayData.EventType[targetType] = group
	}
	blockWayData.EventList[e.EventId] = e

	// 扣除
	blockWayData.TriggerNum -= 1
	blockWayData.Stamina -= excel.GetConfigMgr().GetCfg().ROAD_EVENT_COST

	// 同步给前端
	h.actor.comData.Data.BlockWayEvents = append(h.actor.comData.Data.BlockWayEvents, e)
	h.Infof("blockway event trigger success, triggerNum: %d, stamina: %d, event: %+v", blockWayData.TriggerNum, blockWayData.Stamina, e)
}

func (h *BlockWayHandler) createEvent(cfg *excel.RoadEventCfg) *cmd.PBlockWayEvent {
	return &cmd.PBlockWayEvent{
		EventId:     int64(h.actor.Srv.GenGUID(guid.GUID_EVENT)),
		State:       EventStatusNotTrigger,
		EndTime:     0,
		CfgId:       cfg.Id,
		RefreshTime: common.GetNextDailyRefreshTime(),
	}
}

// TriggerEventReq 触发事件，开始倒计时
func (h *BlockWayHandler) TriggerEventReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_BLOCKWAY)
	if err != nil {
		return nil, err, int32(code)
	}

	var req cmd.C2LS_TriggerEventReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	blockWayData := h.actor.GetBlockWayData()
	tryClearEvent(blockWayData.EventList)

	// 事件不存在
	blockWayEvent, ok := blockWayData.EventList[req.EventId]
	if !ok {
		return nil, fmt.Errorf("event is not found %d", req.EventId), int32(cmd.ErrorCode_ParamError)
	}

	// 事件进行中
	if blockWayEvent.State != EventStatusNotTrigger {
		return nil, fmt.Errorf("event had trigger %d", blockWayEvent.State), int32(cmd.ErrorCode_ParamError)
	}

	// 是否最老的事件
	var oldEvent *cmd.PBlockWayEvent
	for _, e := range blockWayData.EventList {
		if oldEvent == nil {
			oldEvent = e
		} else if e.EventId < oldEvent.EventId {
			oldEvent = e
		}
	}
	if oldEvent.EventId != req.EventId {
		return nil, fmt.Errorf("event is not old %d", req.EventId), int32(cmd.ErrorCode_ParamError)
	}

	// 事件处理
	blockWayEvent.State = EventStatusDoing
	blockWayEvent.EndTime = time.Now().Add(time.Second * time.Duration(excel.GetConfigMgr().GetCfg().ROAD_EVENT_TIME)).Unix()
	h.actor.comData.Data.BlockWayEvents = append(h.actor.comData.Data.BlockWayEvents, blockWayEvent)

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 返回
	return &cmd.LS2C_TriggerEventRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *BlockWayHandler) ReceiveEventGiftReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_BLOCKWAY)
	if err != nil {
		return nil, err, int32(code)
	}

	var req cmd.C2LS_ReceiveEventGiftReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// check
	err, code = h.eventCommonCheck(req.EventId, []int32{EventTypeVisitor, EventTypeHelper})
	if err != nil {
		return nil, err, int32(code)
	}

	blockWayData := h.actor.GetBlockWayData()
	blockWayEvent := blockWayData.EventList[req.EventId]

	cfg := excel.GetRoadEventMgr().GetById(blockWayEvent.CfgId)

	// 交换事件
	if cfg.EventType == EventTypeHelper {
		// 兑换材料校验
		if !GetConsumeMgr(h.actor).CheckKeyValEnough(cfg.Cost) {
			return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
		}

		if err = GetConsumeMgr(h.actor).ConsumeKeyValList(cfg.Cost, h.actor.comData, common.CR_ROAD_SHOP); err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 处理事件数据
	delete(blockWayData.EventList, req.EventId)
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 给奖励
	reward := datahelper.ConvertItem3(cfg.Reward)
	change, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_ROAD_SHOP)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	return &cmd.LS2C_ReceiveEventGiftRes{EventId: req.EventId, CommonData: h.actor.comData.FixDownComData(), DropChange: change}, nil, 0
}

// 事件有效性通用检查
func (h *BlockWayHandler) eventCommonCheck(eventId int64, types []int32) (error, cmd.ErrorCode) {
	blockWayData := h.actor.GetBlockWayData()
	tryClearEvent(blockWayData.EventList)

	// 事件不存在
	blockWayEvent, ok := blockWayData.EventList[eventId]
	if !ok {
		return fmt.Errorf("event is not found %d", eventId), cmd.ErrorCode_ParamError
	}

	// 事件未触发
	if blockWayEvent.State != EventStatusDoing {
		return fmt.Errorf("event not doing %d", blockWayEvent.State), cmd.ErrorCode_ParamError
	}

	// 判定事件类型
	cfg := excel.GetRoadEventMgr().GetById(blockWayEvent.CfgId)
	if cfg == nil {
		return fmt.Errorf("config not found %d", blockWayEvent.CfgId), cmd.ErrorCode_ConfigError
	}
	curType := cfg.EventType
	var f bool
	for _, t := range types {
		if curType == t {
			f = true
		}
	}

	if !f {
		return fmt.Errorf("type not match %d", curType), cmd.ErrorCode_ParamError
	}

	return nil, cmd.ErrorCode_Success
}

func (h *BlockWayHandler) BattleStartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_BLOCKWAY)
	if err != nil {
		return nil, err, int32(code)
	}

	var req cmd.C2LS_EventBattleStartReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 编队校验
	if code = h.actor.TroopHandler.CheckTroopTypAndId(int32(cmd.CardTroopType_CardTroopType_Normal), req.TroopId); code != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("troop check failed %d", req.TroopId), int32(code)
	}
	// 事件校验
	err, code = h.eventCommonCheck(req.EventId, []int32{EventTypeBoss})
	if err != nil {
		return nil, err, int32(code)
	}

	battleId := uint64(utils.GenIntUUID())
	randSeed := utils.GenIntUUID()

	blockWayData := h.actor.GetBlockWayData()

	// 记录数据
	blockWayData.CurTroop = req.TroopId
	blockWayData.CurEventId = req.EventId
	blockWayData.CurBattleId = battleId
	blockWayData.CurRandSeed = randSeed
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 返回
	return &cmd.LS2C_EventBattleStartRes{BattleId: battleId, BattleRandomSeed: randSeed}, nil, 0
}

func (h *BlockWayHandler) BattleEndReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_EventBattleEndReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	blockWayData := h.actor.GetBlockWayData()
	// 校验
	if blockWayData.CurEventId != req.EventId {
		return nil, fmt.Errorf("road event id not match %d", req.EventId), int32(cmd.ErrorCode_ParamError)
	}
	blockWayEvent := blockWayData.EventList[req.EventId]
	cfg := excel.GetRoadEventMgr().GetById(blockWayEvent.CfgId)

	// 胜利了才校验有效性
	var (
		checkBattle *cmd.CheckBattleRes
		errCode     cmd.ErrorCode
	)
	if req.BattleResult == cmd.BattleResult_BattleResult_Winer {
		selfBattleCards := h.actor.BattleHandler.buildSelfBattleCards(cmd.CardTroopType_CardTroopType_Normal, blockWayData.CurTroop, nil)
		checkBattle, err, errCode = h.actor.BattleHandler.CheckBattle(
			blockWayData.CurBattleId, blockWayData.CurRandSeed, req.BattleResult,
			selfBattleCards, cfg.Ariable, req.BattleFrameData, req.VersionData)
		if err != nil {
			return nil, err, int32(errCode)
		}
		// 使用服务器校验结果
		if checkBattle != nil {
			if checkBattle.CheckBattleResult != cmd.CheckBattleResult_CBR_success || checkBattle.BattleResult != req.BattleResult {
				return nil, fmt.Errorf("check battle failed"), int32(cmd.ErrorCode_CheckBattle_fail)
			}
		} else {
			// 没有校验，那就用客户端给的数据
			checkBattle = &cmd.CheckBattleRes{CostFoods: req.CostFoods}
		}
	}

	rsp := &cmd.LS2C_EventBattleEndRes{}
	if req.BattleResult == cmd.BattleResult_BattleResult_Winer {
		// 食物扣除
		costs := myUtils.ConvertItem(checkBattle.CostFoods)
		if !GetConsumeMgr(h.actor).CheckMapEnough(costs) {
			return nil, err, int32(cmd.ErrorCode_FoodNotEnough)
		}
		if err = GetConsumeMgr(h.actor).ConsumeList(costs, h.actor.comData, common.CR_ROAD_SHOP); err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}

		// 下发奖励
		reward := datahelper.ConvertItem3(cfg.Reward)
		rsp.DropChange, err = GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_ROAD_SHOP)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 临时数据清除
	blockWayData.CurEventId = 0
	blockWayData.CurBattleId = 0
	blockWayData.CurRandSeed = 0
	// 事件清除
	delete(blockWayData.EventList, req.EventId)
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 返回结果
	rsp.BattleResult = req.BattleResult
	rsp.CommonData = h.actor.comData.FixDownComData()
	rsp.EventId = req.EventId
	return rsp, nil, 0
}

package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

const (
	TASK_TYPE_DAILY  = 1
	TASK_TYPE_WEEKLY = 2
)

type DutyHandler struct {
	*UABaseHandler
}

func NewDutyHandler(actor *UserActor) *DutyHandler {
	h := &DutyHandler{UABaseHandler: NewUABaseHandler(actor, "DutyHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_InitDutyInfoReq), h.InitDutyInfoReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ChangeDutyCardReq), h.ChangeDutyCardReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ReceiveDailyTaskRewardReq), h.ReceiveDailyTaskRewardReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ReceiveActiveRewardReq), h.ReceiveActiveRewardReq)
	return h
}

// 处理任务类型
func (h *DutyHandler) handleTaskType(e event.IEvent) error {
	temp := make([]*cmd.TaskInfoItem, 0) // 可能会缓存重复的任务,有容错问题不大
	for _, t := range e.Type() {
		for _, task := range h.actor.GetDutyData().UnlockTag {
			if task.CondId == t {
				if h.actor.TaskTypeMgr.CheckTaskConditionComplete(task, e, true) {
					h.actor.comData.GetDutyData().UnlockTag = append(h.actor.comData.GetDutyData().UnlockTag, task)
				}
			}
			if task.CondId == TASK_TYPE_502 {
				temp = append(temp, task)
			}
		}
		tasks := make([]*cmd.TaskInfoItem, 0)
		for _, v := range h.actor.GetDutyData().DailyTask {
			tasks = append(tasks, v)
		}
		for _, v := range h.actor.GetDutyData().WeeklyTask {
			tasks = append(tasks, v)
		}

		for _, task := range tasks {
			if task.CondId == t {
				// 特殊前提
				b := false
				if task.Param1 > 0 {
					if cards := e.Get("card_id"); cards != nil {
						if card, ok := cards.([]int32); ok {
							for _, v := range card {
								if task.Param1 == v {
									b = true
									break
								}
							}
						}
					}
				}
				if task.Param1 == 0 {
					b = true
				}

				if b {
					if h.actor.TaskTypeMgr.CheckTaskConditionComplete(task, e, true) {
						h.actor.comData.GetDutyData().DailyTasks = append(h.actor.comData.GetDutyData().DailyTasks, task)
					}
				}
			}
			if task.CondId == TASK_TYPE_502 {
				temp = append(temp, task)
			}
		}
	}

	// 502类型：不能发布事件处理，只能单独调用
	for _, item := range temp {
		h.actor.TaskTypeMgr.CheckTaskConditionComplete(item, nil, true)
		h.actor.comData.GetDutyData().DailyTasks = append(h.actor.comData.GetDutyData().DailyTasks, item)
	}
	if err := h.SaveDB(); err != nil {
		h.Error(err)
		return nil
	}
	h.Debugf("handleTaskType duty %+v", e)
	return nil
}

// Init 初始化模块数据
func (h *DutyHandler) Init() error {
	// 初始化
	data := &cmd.PDutyData{}
	data.Createtime = time.Now().Unix()
	data.CardId = excel.GetConfigMgr().GetCfg().DUTY_DEFAULT_HERO // 默认值日生
	data.ShowTime = data.Createtime
	data.RefreshTime = data.Createtime
	data.OldCardId = data.CardId
	h.actor.Data.DutyData = data

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init duty data success. player: %s", h.actor.ID())
	return nil
}

func (h *DutyHandler) EnterGame() error {
	return nil
}

func (h *DutyHandler) DailyRefresh() error {
	dutyData := h.actor.GetDutyData()
	err := h.tryClearDutyInfo(dutyData)
	if err != nil {
		return err
	}

	return nil
}

func (h *DutyHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PDutyData); ok {
		h.actor.Data.DutyData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *DutyHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserDutyInfo(h.actor.ID()), h.actor.Data.DutyData
}

// 解锁尝试初始化数据
func (h *DutyHandler) tryInitData(e event.IEvent) error {
	// 先判定解锁
	err, _ := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_DUTY)
	if err != nil {
		return nil
	}

	// 初始化过了
	dutyData := h.actor.Data.DutyData
	if len(dutyData.DailyTask) > 0 {
		return nil
	}

	// 初始任务数据
	dutyData.UnlockTag = h.refreshTags()
	dutyData.DailyTask = h.refreshTasks(TASK_TYPE_DAILY)
	dutyData.WeeklyTask = h.refreshTasks(TASK_TYPE_WEEKLY)
	dutyData.Active = refreshActiveInfo()
	if err = h.SaveDB(); err != nil {
		h.Error(err)
	}
	h.Infof("初始化值日生数据成功")
	return nil
}

func (h *DutyHandler) buildDutyInfo(refresh bool) *cmd.PCommonDutyInfo {
	dutyData := h.actor.GetDutyData()
	err := h.tryClearDutyInfo(dutyData)
	if err != nil {
		return nil
	}

	// 称号信息
	tags := make([]*cmd.TaskInfoItem, 0)
	for _, tag := range dutyData.UnlockTag {
		if tag.Status == TASK_STATUS_COMPLETE {
			tags = append(tags, tag)
		}
	}

	// 刷新显示标记
	isShow := true
	if refresh {
		now := time.Now()
		if !common.IsSameDayByOffset(time.Unix(dutyData.ShowTime, 0), now, common.GAME_DAILY_REFRESH_HOUR) {
			isShow = false
			dutyData.ShowTime = now.Unix()
		}
	}

	// 每日任务
	dailyTasks := make([]*cmd.TaskInfoItem, 0)
	for _, task := range dutyData.DailyTask {
		dailyTasks = append(dailyTasks, task)
	}
	for _, task := range dutyData.WeeklyTask {
		dailyTasks = append(dailyTasks, task)
	}

	// 活跃度信息
	actives := make([]*cmd.ActiveInfoItem, 0)
	for _, active := range dutyData.Active {
		actives = append(actives, active)
	}

	return &cmd.PCommonDutyInfo{
		UnlockTag:  tags,
		DailyTasks: dailyTasks,
		Active:     actives,
		Info: &cmd.DutyBaseInfo{
			CardId:    dutyData.CardId,
			IsShow:    isShow,
			NextTime:  common.GetNextDailyRefreshTime(),
			IsFirst:   common.IsSameDayByOffset(time.Unix(dutyData.Createtime, 0), time.Now(), common.GAME_DAILY_REFRESH_HOUR),
			OldCardId: dutyData.OldCardId,
		},
	}
}

func (h *DutyHandler) InitDutyInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_DUTY)
	if err != nil {
		return nil, err, int32(code)
	}
	commonData := &cmd.CliComData{
		Duty:       h.buildDutyInfo(true),
		SignGroups: h.actor.SignHandler.buildSignInfo(),
	}

	return &cmd.LS2C_InitDutyInfoRes{CommonData: commonData}, nil, 0
}

func (h *DutyHandler) ChangeDutyCardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_DUTY)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_ChangeDutyCardReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 是否拥有
	if !h.actor.CardHandler.IsExistCard(uint32(req.CardId)) {
		return nil, fmt.Errorf("card not exist %d", req.CardId), int32(cmd.ErrorCode_CardNotExist)
	}

	// 同一个？
	dutyData := h.actor.GetDutyData()
	if dutyData.CardId == req.CardId {
		return nil, fmt.Errorf("invalid param"), int32(cmd.ErrorCode_InvalidParam)
	}

	// 处理逻辑
	var old = dutyData.CardId
	dutyData.CardId = req.CardId
	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.ChangeDutyCard{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_ChangeDutyCard, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		BeforeCard:     old,
	//		AfterCard:      req.CardId,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.ChangeDutyCard{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			BeforeCard:        old,
			AfterCard:         req.CardId,
		}
		taptap.WriteDataLog(taptap.LogType_ChangeDutyCard, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 消息返回
	rsp := &cmd.LS2C_ChangeDutyCardRes{CardId: req.CardId}
	return rsp, nil, 0
}

func (h *DutyHandler) ReceiveDailyTaskRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_DUTY)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_ReceiveDailyTaskRewardReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	if req.GetTaskId() == 0 && req.GetCycleType() != TASK_TYPE_DAILY && req.GetCycleType() != TASK_TYPE_WEEKLY {
		return nil, fmt.Errorf("invalid param"), int32(cmd.ErrorCode_InvalidParam)
	}

	var totalActivePoint, cycleType int32
	var cfgReward []*excel.KeyVal
	var tasks []*cmd.TaskInfoItem

	dutyData := h.actor.GetDutyData()
	//step:1 获取要处理的task
	// step:2 统计奖励
	if req.GetTaskId() > 0 { // 领取单个

		cfg := excel.GetDailyTaskMgr().GetById(req.TaskId)
		if cfg == nil {
			return nil, fmt.Errorf("daily task config not found %d", req.TaskId), int32(cmd.ErrorCode_NotFoundConfig)
		}
		cycleType = cfg.CycleType

		if cycleType == TASK_TYPE_DAILY {
			tasks = append(tasks, dutyData.DailyTask[req.TaskId])
		} else if cycleType == TASK_TYPE_WEEKLY {
			tasks = append(tasks, dutyData.WeeklyTask[req.TaskId])
		}

	} else if req.GetTaskId() == 0 { // 一键领取
		cycleType = req.GetCycleType()

		dutyTask := make(map[int32]*cmd.TaskInfoItem, 0)
		if cycleType == TASK_TYPE_DAILY {
			dutyTask = dutyData.DailyTask
		} else if cycleType == TASK_TYPE_WEEKLY {
			dutyTask = dutyData.WeeklyTask
		}
		// 统计可以领取的任务
		for _, task := range dutyTask {
			if task.Status == TASK_STATUS_COMPLETE {
				tasks = append(tasks, task)
			}
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found %d", req.TaskId), int32(cmd.ErrorCode_NoTaskRewardToGet)
	}
	// 统计可以领取的任务
	res := &cmd.LS2C_ReceiveDailyTaskRewardRes{}
	for _, task := range tasks {
		if task == nil {
			return nil, fmt.Errorf("task not found %d", req.TaskId), int32(cmd.ErrorCode_TaskNotFound)
		}
		if task.Status != TASK_STATUS_COMPLETE {
			return nil, fmt.Errorf("task not complete status %d", task.Status), int32(cmd.ErrorCode_TaskStatusNotComplete)
		}
		// 处理逻辑
		task.Status = TASK_STATUS_RECEIVED
		cfg := excel.GetDailyTaskMgr().GetById(task.Id)
		if cfg == nil {
			return nil, fmt.Errorf("daily task config not found %d", req.TaskId), int32(cmd.ErrorCode_NotFoundConfig)
		}
		// 统计或于都和奖励
		totalActivePoint += cfg.ActivePoint
		cfgReward = append(cfgReward, cfg.Rewards...)
		res.TaskId = append(res.TaskId, task.Id)
		// 埋点log
		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.ReceiveDailyReward{
		//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_ReceiveDailyReward, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		TaskId:         task.Id,
		//		TaskType:       task.TaskType,
		//		CondId:         task.CondId,
		//		Target:         task.TargetValue,
		//		Active:         cfg.ActivePoint,
		//		Reward:         lilith.ConvertMap2Str(datahelper.ConvertItem3(cfg.Rewards)),
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.ReceiveDailyReward{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				TaskId:            task.Id,
				TaskType:          task.TaskType,
				CondId:            task.CondId,
				Target:            task.TargetValue,
				Active:            cfg.ActivePoint,
				Reward:            taptap.ConvertMap2Str(datahelper.ConvertItem3(cfg.Rewards)),
			}
			taptap.WriteDataLog(taptap.LogType_ReceiveDailyReward, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}
	// 处理活跃度
	active, ok := dutyData.Active[cycleType]
	if ok {
		old := active.CurValue
		active.CurValue += totalActivePoint
		h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_DUTY_ACTIVE_CHANGE, []int32{}, map[string]interface{}{
			"type":          cycleType,       // 1=每日，2=每周
			"before_active": old,             // 增加前活跃度
			"after_active":  active.CurValue, // 增加后活跃度
		}))
	} else {
		return nil, fmt.Errorf("task cycle type nor found %d", cycleType), int32(cmd.ErrorCode_ConfigError)
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	reward := datahelper.ConvertItem3(cfgReward)
	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Daily_Task_Reward)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	res.CommonData = h.actor.comData.FixDownComData()
	res.Active = active
	res.DropChange = dropChange
	return res, nil, 0
}

func (h *DutyHandler) ReceiveActiveRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_DUTY)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_ReceiveActiveRewardReq
	err = proto.Unmarshal(in.Data, &req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// check
	dutyData := h.actor.GetDutyData()
	active := dutyData.Active[req.ActiveType]
	if active == nil {
		return nil, fmt.Errorf("active not found %d", req.ActiveType), int32(cmd.ErrorCode_InvalidParam)
	}
	cfg := excel.GetDailyActiveMgr().GetById(req.ActiveNode)
	if cfg == nil {
		return nil, fmt.Errorf("daily active config not found %d", req.ActiveNode), int32(cmd.ErrorCode_NotFoundConfig)
	}
	if active.CurValue < req.ActiveNode {
		return nil, fmt.Errorf("task active not complete"), int32(cmd.ErrorCode_InvalidParam)
	}
	for _, v := range active.Received {
		if v == req.ActiveNode {
			return nil, fmt.Errorf("task active had received %d", req.ActiveNode), int32(cmd.ErrorCode_InvalidParam)
		}
	}

	// 处理逻辑
	active.Received = append(active.Received, req.ActiveNode)

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	var reward map[int32]int32
	if req.ActiveType == TASK_TYPE_DAILY {
		reward = datahelper.ConvertItem3(cfg.DailyRewards)
	} else {
		reward = datahelper.ConvertItem3(cfg.WeeklyRewards)
	}
	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Daily_Task_Active_Reward)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.ReceiveActiveReward{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_ReceiveActiveReward, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ActiveNode:     req.ActiveNode,
	//		ActiveType:     req.ActiveType,
	//		Reward:         lilith.ConvertMap2Str(reward),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.ReceiveActiveReward{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ActiveNode:        req.ActiveNode,
			ActiveType:        req.ActiveType,
			Reward:            taptap.ConvertMap2Str(reward),
		}
		taptap.WriteDataLog(taptap.LogType_ReceiveActiveReward, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	res := &cmd.LS2C_ReceiveActiveRewardRes{
		CommonData: h.actor.comData.FixDownComData(),
		Active:     active,
		DropChange: dropChange,
	}
	return res, nil, 0
}

// 尝试清空数据
func (h *DutyHandler) tryClearDutyInfo(dutyData *cmd.PDutyData) error {
	// 先判定解锁
	err, _ := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_DUTY)
	if err != nil {
		return nil
	}

	b := false
	now := time.Now()

	// 刷新每日数据
	refreshTime := time.Unix(dutyData.RefreshTime, 0)
	if !common.IsSameDayByOffset(refreshTime, now, common.GAME_DAILY_REFRESH_HOUR) {
		// 自动领取奖励
		h.tryAutoReceiveReward(dutyData.DailyTask)
		h.tryAutoReceiveActive(dutyData.Active[TASK_TYPE_DAILY])

		dutyData.UnlockTag = h.refreshTags()
		dutyData.DailyTask = h.refreshTasks(TASK_TYPE_DAILY)
		dutyData.OldCardId = dutyData.CardId
		dutyData.Active[TASK_TYPE_DAILY] = &cmd.ActiveInfoItem{
			ActiveType: TASK_TYPE_DAILY,
			CurValue:   0,
			Received:   make([]int32, 0),
		}
		b = true
	}

	// 刷新每周数据
	if !common.IsSameWeekByOffset(refreshTime, now, common.GAME_DAILY_REFRESH_HOUR) {
		// 自动领取奖励
		h.tryAutoReceiveReward(dutyData.WeeklyTask)
		h.tryAutoReceiveActive(dutyData.Active[TASK_TYPE_WEEKLY])

		dutyData.WeeklyTask = h.refreshTasks(TASK_TYPE_WEEKLY)
		dutyData.Active[TASK_TYPE_WEEKLY] = &cmd.ActiveInfoItem{
			ActiveType: TASK_TYPE_WEEKLY,
			CurValue:   0,
			Received:   make([]int32, 0),
		}
		b = true
	}

	if b {
		dutyData.RefreshTime = time.Now().Unix()
		if err := h.SaveDB(); err != nil {
			return err
		}
	}

	return nil
}

// 未领取每日任务奖励发送邮件
func (h *DutyHandler) tryAutoReceiveReward(tasks map[int32]*cmd.TaskInfoItem) {
	reward := make([]*excel.KeyVal, 0)
	for _, task := range tasks {
		if task.Status != TASK_STATUS_COMPLETE {
			continue
		}

		cfg := excel.GetDailyTaskMgr().GetById(task.Id)
		if cfg == nil {
			continue
		}

		// 累计奖励
		reward = append(reward, cfg.Rewards...)
	}

	if len(reward) > 0 {
		attachment := datahelper.ConvertItem3(reward)
		h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_3, attachment, h.actor.comData)
	}
}

// 未领取活跃度奖励发送邮件
func (h *DutyHandler) tryAutoReceiveActive(active *cmd.ActiveInfoItem) {
	reward := make([]*excel.KeyVal, 0)
	received := make(map[int32]int32)
	for _, v := range active.Received {
		received[v] = v
	}
	excel.GetDailyActiveMgr().Foreach(func(cfg *excel.DailyActiveCfg) bool {
		if _, ok := received[cfg.Id]; !ok && cfg.Id <= active.CurValue {
			if active.ActiveType == TASK_TYPE_DAILY {
				reward = append(reward, cfg.DailyRewards...)
			}
			if active.ActiveType == TASK_TYPE_WEEKLY {
				reward = append(reward, cfg.WeeklyRewards...)
			}
		}

		return true
	}, true)

	if len(reward) > 0 {
		attachment := datahelper.ConvertItem3(reward)
		h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_3, attachment, h.actor.comData)
	}
}

// 接取称号数据
func (h *DutyHandler) refreshTags() map[int32]*cmd.TaskInfoItem {
	tasks := make(map[int32]*cmd.TaskInfoItem)
	excel.GetDailyTagMgr().Foreach(func(cfg *excel.DailyTagCfg) bool {
		// 特殊处理
		if cfg.TagType == 0 {
			tasks[cfg.Id] = &cmd.TaskInfoItem{
				Id:     cfg.Id,
				Status: TASK_STATUS_COMPLETE,
			}
		} else {
			// 正常的数据
			task := h.actor.TaskTypeMgr.CreateTaskInfoItem(cfg.Id, cfg.TagType, cfg.Target, cfg.Tagparams, true)
			if task != nil {
				tasks[cfg.Id] = task
			}
		}

		return true
	}, true)

	return tasks
}

// 接取任务数据
func (h *DutyHandler) refreshTasks(cycleType int32) map[int32]*cmd.TaskInfoItem {
	tasks := make(map[int32]*cmd.TaskInfoItem)
	excel.GetDailyTaskMgr().Foreach(func(cfg *excel.DailyTaskCfg) bool {
		if cfg.CycleType != cycleType {
			return true
		}
		// 正常的数据
		task := h.actor.TaskTypeMgr.CreateTaskInfoItem(cfg.Id, cfg.TaskType, cfg.Target, cfg.Taskparams, true)
		if task != nil {
			if cfg.DutySp == 1 {
				task.Param1 = h.GetCurDutyCard()
			}
			task.TaskType = cycleType
			tasks[cfg.Id] = task
		}

		return true
	}, true)
	h.Debugf("refreshTasks uid:%s, data:%d", h.actor.ID(), len(tasks))
	return tasks
}

func refreshActiveInfo() map[int32]*cmd.ActiveInfoItem {
	m := make(map[int32]*cmd.ActiveInfoItem)
	m[TASK_TYPE_DAILY] = &cmd.ActiveInfoItem{
		ActiveType: TASK_TYPE_DAILY,
		CurValue:   0,
		Received:   make([]int32, 0),
	}
	m[TASK_TYPE_WEEKLY] = &cmd.ActiveInfoItem{
		ActiveType: TASK_TYPE_WEEKLY,
		CurValue:   0,
		Received:   make([]int32, 0),
	}
	return m
}

// GetCurDutyCard 获取当前的值日生
func (h *DutyHandler) GetCurDutyCard() int32 {
	data := h.actor.GetDutyData()
	if data != nil {
		return data.CardId
	}
	return excel.GetConfigMgr().GetCfg().DUTY_DEFAULT_HERO
}

func (h *DutyHandler) ResetTaskByGM(commonData *clidto.Comdata) error {
	data := h.actor.GetDutyData()
	data.DailyTask = h.refreshTasks(TASK_TYPE_DAILY)
	data.WeeklyTask = h.refreshTasks(TASK_TYPE_WEEKLY)

	dutyData := commonData.GetDutyData()
	for _, task := range data.DailyTask {
		dutyData.DailyTasks = append(dutyData.DailyTasks, task)
	}
	for _, task := range data.WeeklyTask {
		dutyData.DailyTasks = append(dutyData.DailyTasks, task)
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return fmt.Errorf("init duty data failed. err: %v", err)
	}
	return nil
}

func (h *DutyHandler) DirectCompleteTaskByGM(taskId int32, commonData *clidto.Comdata) error {
	cfg := excel.GetDailyTaskMgr().GetById(taskId)
	if cfg == nil {
		return fmt.Errorf("daily task config not found %d", taskId)
	}

	dutyData := h.actor.GetDutyData()
	var task *cmd.TaskInfoItem
	if cfg.CycleType == TASK_TYPE_DAILY {
		task = dutyData.DailyTask[taskId]
	} else if cfg.CycleType == TASK_TYPE_WEEKLY {
		task = dutyData.WeeklyTask[taskId]
	}

	if task == nil {
		return fmt.Errorf("task not found %d", taskId)
	}

	task.Status = TASK_STATUS_COMPLETE
	commonData.GetDutyData().DailyTasks = append(commonData.GetDutyData().DailyTasks, task)

	// 保存
	if err := h.SaveDB(); err != nil {
		return fmt.Errorf("init duty data failed. err: %v", err)
	}
	return nil
}

// 检查每日任务是否全部完成,排除任务类型502
func (h *DutyHandler) CheckDailyTaskAllComplete() bool {
	data := h.actor.GetDutyData()

	for _, item := range data.DailyTask {
		// 排除任务类型502
		if item.CondId == TASK_TYPE_502 {
			continue
		}

		if item.Status == TASK_STATUS_DOING {
			return false
		}
	}

	return true
}

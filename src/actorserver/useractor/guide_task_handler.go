package useractor

import (
	"context"
	"fmt"
	"time"

	"gitee.com/bychannel/aniwar/src/actorserver/useractor/event"
	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/common/datahelper"
	"gitee.com/bychannel/aniwar/src/common/db"
	excel "gitee.com/bychannel/aniwar/src/excel/data"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/base"
	"gitee.com/bychannel/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type GuideTaskHandler struct {
	*UABaseHandler
}

func NewGuideTaskHandler(actor *UserActor) *GuideTaskHandler {
	h := &GuideTaskHandler{UABaseHandler: NewUABaseHandler(actor, "GuideTaskHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ReceiveGuideTaskRewardReq), h.ReceiveGuideTaskRewardReq)
	return h
}

// Init 初始化模块数据
func (h *GuideTaskHandler) Init() error {
	// 初始化
	h.actor.Data.GuideTaskData = &pb.PGuideTaskData{
		Createtime: time.Now().Unix(),
		Tasks:      make(map[int32]*pb.TaskInfoItem),
		Complete:   make(map[int32]int32),
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init guide task data success. player: %s", h.actor.ID())
	return nil
}

func (h *GuideTaskHandler) EnterGame() error {
	return h.startTaskTimer()
}

func (h *GuideTaskHandler) DailyRefresh() error {
	return nil
}

func (h *GuideTaskHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PGuideTaskData); ok {
		h.actor.Data.GuideTaskData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}
	return nil
}

func (h *GuideTaskHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserGuideTask(h.actor.ID()), h.actor.Data.GuideTaskData
}

// 启动timer时钟
func (h *GuideTaskHandler) startTaskTimer() error {
	err := h.tryCloseExpireTask(true)
	if err != nil {
		return err
	}

	// 尝试开启计时器
	data := h.actor.GetGuideTaskData()
	for _, task := range data.Tasks {
		h.addTaskTimer(task)
	}
	return nil
}

// 给任务加个计时器
func (h *GuideTaskHandler) addTaskTimer(task *pb.TaskInfoItem) {
	after := task.ExpireTs - time.Now().Unix()
	if after <= 0 {
		return
	}
	h.actor.Timer.AfterFunc(time.Second*time.Duration(after), func() {
		h.tryCloseExpireTask(true)
	}, false)
}

// 处理任务完成
func (h *GuideTaskHandler) handleTaskType(e event.IEvent) error {
	data := h.actor.GetGuideTaskData()
	for _, t := range e.Type() {
		for _, task := range data.Tasks {
			if task.CondId == t {
				if h.actor.TaskTypeMgr.CheckTaskConditionComplete(task, e, true) {
					h.actor.comData.Data.GuideTask = append(h.actor.comData.Data.GuideTask, task)
				}
			}
		}
	}
	if err := h.SaveDB(); err != nil {
		h.Error(err)
		return nil
	}
	h.Infof("handleTaskType guide task %+v", e)
	return nil
}

// 处理任务触发
func (h *GuideTaskHandler) handleTaskTrigger(e event.IEvent) error {
	return h.tryUnlockTask()
}

func (h *GuideTaskHandler) buildGuideTask() []*pb.TaskInfoItem {
	data := h.actor.GetGuideTaskData()
	tasks := make([]*pb.TaskInfoItem, 0)
	for _, task := range data.Tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// 领取任务奖励
func (h *GuideTaskHandler) ReceiveGuideTaskRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_GUIDETASK)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_ReceiveGuideTaskRewardReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 刷新一下数据
	h.tryCloseExpireTask(false)

	data := h.actor.GetGuideTaskData()
	cfg := excel.GetTaskMgr().GetById(req.TaskId)
	// 任务不存在
	task := data.Tasks[req.TaskId]
	if task == nil || cfg == nil {
		return nil, fmt.Errorf("task not found %d", req.TaskId), int32(pb.ErrorCode_TaskNotFound)
	}
	// 不是可领取状态
	if task.Status != TASK_STATUS_COMPLETE {
		return nil, fmt.Errorf("task not complete status %d", task.Status), int32(pb.ErrorCode_TaskStatusNotComplete)
	}

	// 领取
	delete(data.Tasks, req.TaskId)
	data.Complete[req.TaskId] = 0

	// 尝试接取
	h.tryUnlockTask()

	reward := datahelper.ConvertItem3(cfg.Reward)
	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_FINISH_GUIDE_TASK)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	res := &pb.LS2C_ReceiveGuideTaskRewardRes{
		TaskId:     req.TaskId,
		CommonData: h.actor.comData.FixDownComData(),
		DropChange: dropChange,
	}
	return res, nil, 0
}

// 尝试关闭限时任务
func (h *GuideTaskHandler) tryCloseExpireTask(tryUnlock bool) error {
	data := h.actor.GetGuideTaskData()
	for id, task := range data.Tasks {
		// 不是限时任务
		if task.ExpireTs == 0 {
			continue
		}
		// 没过期
		if time.Now().Unix() < task.ExpireTs {
			continue
		}

		// 尝试发奖励邮件
		if task.Status == TASK_STATUS_COMPLETE {
			cfg := excel.GetTaskMgr().GetById(id)
			if cfg != nil {
				attachment := datahelper.ConvertItem3(cfg.Reward)
				err := h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_3, attachment, h.actor.comData) // fixme 奖励邮件模板
				if err != nil {
					return err
				}
			}
		}
		delete(data.Tasks, id)
	}
	// 尝试接取任务
	if tryUnlock {
		return h.tryUnlockTask()
	}
	return nil
}

// 尝试接取新任务
func (h *GuideTaskHandler) tryUnlockTask() error {
	// 未解锁
	err, _ := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_GUIDETASK)
	if err != nil {
		return nil
	}

	// 任务队列是否满了
	data := h.actor.GetGuideTaskData()
	if len(data.Tasks) >= int(excel.GetConfigMgr().GetCfg().GUIDETASK_MAX_NUM) {
		return nil
	}

	// 尝试解锁任务
	tasks := h.actor.TaskTriggerMgr.TriggerTaskGroupByType(TASK_GROUP_TYPE_6)
	if len(tasks) == 0 {
		return nil
	}
	// 创建任务数据
	for _, task := range tasks {
		// 是否完成
		if _, ok := data.Complete[task.Id]; ok {
			continue
		}
		// 是否队列中
		if _, ok := data.Tasks[task.Id]; ok {
			continue
		}
		// 是否满了
		if len(data.Tasks) >= int(excel.GetConfigMgr().GetCfg().GUIDETASK_MAX_NUM) {
			continue
		}

		// 创建任务
		taskInfo := h.actor.TaskTypeMgr.CreateTaskInfoItemNew(task, true)
		if taskInfo != nil {
			data.Tasks[task.Id] = taskInfo
			h.actor.comData.Data.GuideTask = append(h.actor.comData.Data.GuideTask, taskInfo)
			h.addTaskTimer(taskInfo)
		}
	}
	return h.SaveDB()
}

// 检查给定的任务id列表是否都完成过，全部完成返回true
func (h *GuideTaskHandler) CheckTaskComplete(taskIds []int32) bool {
	data := h.actor.GetGuideTaskData()
	for _, id := range taskIds {
		if _, ok := data.Complete[id]; !ok {
			return false
		}
	}
	return true
}

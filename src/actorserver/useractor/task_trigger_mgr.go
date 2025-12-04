package useractor

import (
	"strconv"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

type TaskTriggerMgr struct {
	actor *UserActor
}

func NewTaskTriggerMgr(actor *UserActor) *TaskTriggerMgr {
	return &TaskTriggerMgr{actor: actor}
}

// 任务组类型
const (
	TASK_GROUP_TYPE_1 = 1 // 主线任务组
	TASK_GROUP_TYPE_2 = 2 // 成就任务组
	TASK_GROUP_TYPE_3 = 3 // 日常任务组
	TASK_GROUP_TYPE_4 = 4 // 周常任务组
	TASK_GROUP_TYPE_6 = 6 // 引导任务组
	TASK_GROUP_TYPE_7 = 7 // 活动任务组
)

// 任务组触发器
const (
	TASK_GROUP_TRIGGER_1  = 1  // 等级区间
	TASK_GROUP_TRIGGER_2  = 2  // 账号的等级≥N
	TASK_GROUP_TRIGGER_3  = 3  // 时间长度（分钟）
	TASK_GROUP_TRIGGER_4  = 4  // 时间范围(日期A+时期A~日期B+时期B)
	TASK_GROUP_TRIGGER_5  = 5  // 完成某任务id几次
	TASK_GROUP_TRIGGER_6  = 6  // 完成前置任务组ID
	TASK_GROUP_TRIGGER_7  = 7  // 通关某类型关卡id几次
	TASK_GROUP_TRIGGER_8  = 8  // 通关某个关卡id
	TASK_GROUP_TRIGGER_9  = 9  // 消耗某道具id几次
	TASK_GROUP_TRIGGER_10 = 10 // 满足条件类型表id包含xx、xx、xx的类型
	TASK_GROUP_TRIGGER_11 = 11 // 通关探索xx任务
	TASK_GROUP_TRIGGER_12 = 12 // 激活某个营地内建筑ID
	TASK_GROUP_TRIGGER_13 = 13 // 营地内某个建筑ID达到XX级
	TASK_GROUP_TRIGGER_14 = 14 // 角色等级到达xx级
	TASK_GROUP_TRIGGER_15 = 15 // 角色好感度达到xx级
	TASK_GROUP_TRIGGER_16 = 16 // 角色突破次数达到X次（1-5）
	TASK_GROUP_TRIGGER_17 = 17 // 特定角色血量达到指定比例
)

// 任务前置条件
const (
	TASK_TRIGGER_1 = 1 // 主线任务完成
	TASK_TRIGGER_2 = 2 // 系统功能开放
)

// --------------------- 触发器统一注册 ---------------------

func (m *TaskTriggerMgr) RegisterTaskTriggerHandler(handler event.IListener) {
	m.actor.eventManager.Listen(TASK_EVENT_ROLE_LEVEL_CHANGE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_LEVEL_WIN, handler)
	m.actor.eventManager.Listen(TASK_EVENT_ITEM_CONSUME, handler)
	m.actor.eventManager.Listen(TASK_EVENT_QUEST_COMPLETE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_LEVEL_UPGRADE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_BREAKTHROUGH, handler)
	m.actor.eventManager.Listen(TASK_EVENT_BUILDING_MAKE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_TASK_COMPLETE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_TASK_GROUP_COMPLETE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_CAMPAIGN_LEVEL, handler)
	m.actor.eventManager.Listen(TASK_EVENT_TRAVEL_LEVEL_WIN, handler)
}

// --------------------- 任务类型统一触发 ---------------------

func (m *TaskTriggerMgr) TriggerTaskGroupById(groupId int32) []*excel.TaskCfg {
	cfg := excel.GetTaskgroupMgr().GetById(groupId)
	if cfg == nil {
		return []*excel.TaskCfg{}
	}
	return m.checkTriggerCondition(map[int32]*excel.TaskgroupCfg{groupId: cfg})
}

// TriggerTaskGroupByType 对外统一触发任务组
func (m *TaskTriggerMgr) TriggerTaskGroupByType(groupType int32) []*excel.TaskCfg {
	// 获取任务组配置
	groups := make(map[int32]*excel.TaskgroupCfg)
	excel.GetTaskgroupMgr().Foreach(func(cfg *excel.TaskgroupCfg) bool {
		if cfg.TaskgroupType == groupType {
			groups[cfg.Id] = cfg
		}
		return true
	}, true)

	return m.checkTriggerCondition(groups)
}

func (m *TaskTriggerMgr) checkTriggerCondition(groups map[int32]*excel.TaskgroupCfg) []*excel.TaskCfg {
	tasks := make([]*cmd.KeyValueItem, 0)
	for _, cfg := range groups {
		// 触发条件
		if !m.checkTriggerType(cfg.TriggerType) {
			continue
		}
		// 前置任务组
		if !m.checkPreGroupComplete(cfg) {
			continue
		}
		for _, taskId := range cfg.TaskId {
			tasks = append(tasks, &cmd.KeyValueItem{
				Key:   cfg.TaskgroupType,
				Value: taskId,
			})
		}
	}

	// 处理任务触发
	openTasks := make([]*excel.TaskCfg, 0)
	for _, taskItem := range tasks {
		taskCfg := excel.GetTaskMgr().GetById(taskItem.Value)
		if taskCfg == nil {
			continue
		}
		// 前置条件
		if !m.checkPreCondition(taskCfg.PreCondition) {
			continue
		}
		// 前置任务
		if !m.checkPreTaskComplete(taskItem.Key, taskCfg) {
			continue
		}
		openTasks = append(openTasks, taskCfg)
	}
	return openTasks
}

// 判断前置任务是否完成，完成返回true
func (m *TaskTriggerMgr) checkPreTaskComplete(groupType int32, cfg *excel.TaskCfg) bool {
	// 没配置
	if len(cfg.PreTaskId) == 0 {
		return true
	}
	// 按类型处理
	// 不需要处理的类型直接返回true
	switch groupType {
	case TASK_GROUP_TYPE_6:
		return m.actor.GuideTaskHandler.CheckTaskComplete(cfg.PreTaskId)
	default:
		return false
	}
}

// 校验任务前置条件是否达成，达成返回true
func (m *TaskTriggerMgr) checkPreCondition(conditions map[int32]int32) bool {
	for k, v := range conditions {
		switch k {
		case TASK_TRIGGER_1:
			if !m.actor.QuestHandler.checkQuestFinish(v) {
				return false
			}
		case TASK_TRIGGER_2:
			err, _ := m.actor.FuncUnlockHandler.CheckFuncUnlockBase(v)
			if err != nil {
				return false
			}
		default:
			m.actor.Errorf("unrealized task trigger type %d", k)
			return false
		}
	}
	return true
}

// 判断前置任务组是否完成，完成返回true
func (m *TaskTriggerMgr) checkPreGroupComplete(cfg *excel.TaskgroupCfg) bool {
	// 没配置
	if cfg.PreTaskgroupId <= 0 {
		return true
	}
	// 按类型处理
	// 不需要处理的类型直接返回true
	switch cfg.TaskgroupType {
	case TASK_GROUP_TYPE_6:
		return true
	default:
		return false
	}
}

// 校验触发器是否达成，达成返回true
func (m *TaskTriggerMgr) checkTriggerType(triggerIds []int32) bool {
	for _, id := range triggerIds {
		if id == 0 {
			continue
		}

		// 获取触发器配置
		cfg := excel.GetTriggerMgr().GetById(id)
		if cfg == nil {
			return false
		}
		// 按类型处理参数
		f := true
		switch cfg.TriggerType {
		case TASK_GROUP_TRIGGER_1:
			f = m.checkTrigger1(cfg)
		case TASK_GROUP_TRIGGER_2:
			f = m.checkTrigger2(cfg)
		case TASK_GROUP_TRIGGER_5:
			f = m.checkTrigger5(cfg)
		case TASK_GROUP_TRIGGER_6:
			f = m.checkTrigger6(cfg)
		case TASK_GROUP_TRIGGER_7:
			f = m.checkTrigger7(cfg)
		case TASK_GROUP_TRIGGER_9:
			f = m.checkTrigger9(cfg)
		case TASK_GROUP_TRIGGER_11:
			f = m.checkTrigger11(cfg)
		case TASK_GROUP_TRIGGER_12:
			f = m.checkTrigger12(cfg)
		case TASK_GROUP_TRIGGER_14:
			f = m.checkTrigger14(cfg)
		case TASK_GROUP_TRIGGER_16:
			f = m.checkTrigger16(cfg)
		default:
			m.actor.Errorf("unrealized task group trigger type %d", id)
			return false
		}
		// 当前触发器校验失败了，后续不再处理
		if f == false {
			return false
		}
	}
	return true
}

// 触发器:账号等级区间
func (m *TaskTriggerMgr) checkTrigger1(cfg *excel.TriggerCfg) bool {
	level := int(m.actor.LoginHandler.getRoleLevel())
	min, err := strconv.Atoi(cfg.Param01)
	max, err := strconv.Atoi(cfg.Param02)
	if err != nil {
		return false
	}
	if level < min || level > max {
		return false
	}
	return true
}

// 触发器:账号等级>=N
func (m *TaskTriggerMgr) checkTrigger2(cfg *excel.TriggerCfg) bool {
	level := int(m.actor.LoginHandler.getRoleLevel())
	min, err := strconv.Atoi(cfg.Param01)
	if err != nil {
		return false
	}
	return level >= min
}

// 触发器:完成指定任务X次
func (m *TaskTriggerMgr) checkTrigger5(cfg *excel.TriggerCfg) bool {
	taskId, err := strconv.Atoi(cfg.Param01)
	count, err := strconv.Atoi(cfg.Param02)
	if err != nil {
		return false
	}
	total := m.actor.AchieveHandler.GetAchieveNum(buildTaskKey(taskId))
	return total >= int32(count)
}

// 触发器:完成指定任务组
func (m *TaskTriggerMgr) checkTrigger6(cfg *excel.TriggerCfg) bool {
	taskGroupId, err := strconv.Atoi(cfg.Param01)
	if err != nil {
		return false
	}
	total := m.actor.AchieveHandler.GetAchieveNum(buildTaskGroupKey(taskGroupId))
	return total >= 1
}

// 触发器:通关某类型关卡X次
func (m *TaskTriggerMgr) checkTrigger7(cfg *excel.TriggerCfg) bool {
	levelType, err := strconv.Atoi(cfg.Param01)
	count, err := strconv.Atoi(cfg.Param02)
	if err != nil {
		return false
	}
	total := m.actor.AchieveHandler.GetAchieveNum(buildLevelTypeKey(levelType))
	return total >= int32(count)
}

// 触发器：消耗某道具id几次
func (m *TaskTriggerMgr) checkTrigger9(cfg *excel.TriggerCfg) bool {
	itemId, err := strconv.Atoi(cfg.Param01)
	count, err := strconv.Atoi(cfg.Param02)
	if err != nil {
		return false
	}
	total := m.actor.AchieveHandler.GetAchieveNum(buildItemCountKey(itemId))
	return total >= int32(count)
}

// 触发器：通关探索xx任务
func (m *TaskTriggerMgr) checkTrigger11(cfg *excel.TriggerCfg) bool {
	questId, err := strconv.Atoi(cfg.Param01)
	if err != nil {
		return false
	}
	return m.actor.QuestHandler.checkQuestFinish(int32(questId))
}

// 触发器：激活某个营地内建筑ID
func (m *TaskTriggerMgr) checkTrigger12(cfg *excel.TriggerCfg) bool {
	buildingId, err := strconv.Atoi(cfg.Param01)
	if err != nil {
		return false
	}
	return m.actor.CampHandler.BuildingExist(int32(buildingId))
}

// 触发器：角色等级到达xx级
func (m *TaskTriggerMgr) checkTrigger14(cfg *excel.TriggerCfg) bool {
	level, err := strconv.Atoi(cfg.Param01)
	if err != nil {
		return false
	}
	total := m.actor.AchieveHandler.GetAchieveNum(CARD_MAX_LEVEL)
	return total >= int32(level)
}

// 触发器：角色突破次数达到X次
func (m *TaskTriggerMgr) checkTrigger16(cfg *excel.TriggerCfg) bool {
	count, err := strconv.Atoi(cfg.Param01)
	if err != nil {
		return false
	}
	total := m.actor.AchieveHandler.GetAchieveNum(CARD_MAX_BREAKTHROUGH)
	return total >= int32(count)
}

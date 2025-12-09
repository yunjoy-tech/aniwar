package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"time"
)

type TaskTypeMgr struct {
	actor *UserActor
}

func NewTaskTypeMgr(actor *UserActor) *TaskTypeMgr {
	return &TaskTypeMgr{actor: actor}
}

const (
	TASK_STATUS_DOING    = 0 // 进行中
	TASK_STATUS_COMPLETE = 1 // 已完成
	TASK_STATUS_RECEIVED = 2 // 已领取
)

const (
	TASK_COND_TYPE_NORMAL = 0 // 常规
	TASK_COND_TYPE_LIFE   = 1 // 生涯
)

const (
	TASK_TYPE_1   = 1   // 击败怪物X个
	TASK_TYPE_2   = 2   // monster_id 击败XX怪物Y个
	TASK_TYPE_3   = 3   // 击败大地图怪物Y个
	TASK_TYPE_11  = 11  // 完成采集X次
	TASK_TYPE_12  = 12  // resource_id	完成XX采集Y次
	TASK_TYPE_31  = 31  // 开启宝箱X次
	TASK_TYPE_41  = 41  // 完成日替碰碰车玩法X次
	TASK_TYPE_91  = 91  // level_id	完成关卡X总共Y次
	TASK_TYPE_92  = 92  // X次战斗胜利
	TASK_TYPE_101 = 101 // 提升角色技能等级X次
	TASK_TYPE_102 = 102 // 提升角色潜力X次
	TASK_TYPE_103 = 103 // 提升角色性格X次
	TASK_TYPE_104 = 104 // 突破角色X次
	TASK_TYPE_105 = 105 // 进行角色养成X次（包括上面101 102 103 104 121）
	TASK_TYPE_111 = 111 // 使用食物X个
	TASK_TYPE_112 = 112 // 消耗体力X点
	TASK_TYPE_113 = 113 // 消耗指定道具X个
	TASK_TYPE_121 = 121 // 角色升级X次
	TASK_TYPE_201 = 201 // 获得装备X次
	TASK_TYPE_202 = 202 // quality	获得品质X的装备Y次
	TASK_TYPE_203 = 203 // 打造装备次数X次
	TASK_TYPE_211 = 211 // 升级装备次数
	TASK_TYPE_212 = 212 // 重铸装备次数
	TASK_TYPE_301 = 301 // 收取光合树次数
	TASK_TYPE_302 = 302 // 建筑升级次数
	TASK_TYPE_303 = 303 // build_id	升级X建筑Y次
	TASK_TYPE_311 = 311 // 打造家具X次
	TASK_TYPE_312 = 312 // 食物制作X次
	TASK_TYPE_313 = 313 // 生产电力X次
	TASK_TYPE_401 = 401 // hero_id	获得id为X的角色
	TASK_TYPE_402 = 402 // 累计获得X个角色
	TASK_TYPE_403 = 403 // rarity	获得品质为X的角色Y个
	TASK_TYPE_404 = 404 // skin_id	获得id为X的皮肤
	TASK_TYPE_406 = 406 // 累计获得X个皮肤
	TASK_TYPE_407 = 407 // 抽卡X次
	TASK_TYPE_408 = 408 // 指定类型卡池抽卡
	TASK_TYPE_501 = 501 // 累计登录天数
	TASK_TYPE_502 = 502 // 完成所有每日任务
	TASK_TYPE_503 = 503 // 累计登录天数(非生涯)
	TASK_TYPE_504 = 504 // 玩家账号等级x级
	TASK_TYPE_505 = 505 // 累计获得x个n等级的角色
	TASK_TYPE_507 = 507 // 通关xx类型副本n次
	TASK_TYPE_508 = 508 // 通关游戏日替副本n次
	TASK_TYPE_510 = 510 // 收取n次营地指定产物
	TASK_TYPE_511 = 511 // 指定建筑升级到n级
	TASK_TYPE_512 = 512 // x个营地建筑升级到n级
	TASK_TYPE_517 = 517 // 解锁指定营地建筑
	TASK_TYPE_518 = 518 // 指定角色在指定建筑工作
	TASK_TYPE_521 = 521 // 完成指定主线任务
	TASK_TYPE_525 = 525 // 采集n个指定类型采集物
	TASK_TYPE_527 = 527 // 采集n次指定类型采集物
)

const (
	TASK_EVENT_MONSTER_BATTLE       = "chapter.monster.battle"
	TASK_EVENT_LEVEL_COLLECT        = "chapter.level.collect"
	TASK_EVENT_LEVEL_BOX            = "chapter.level.box"
	TASK_EVENT_LEVEL_WIN            = "chapter.level.win"
	TASK_EVENT_LEVEL_ENTER          = "chapter.level.enter"
	TASK_EVENT_BATTLE_WIN           = "chapter.battle.win"
	TASK_EVENT_UNLOCK_POINT         = "chapter.unlock.point"
	TASK_EVENT_SKILL_UPGRADE        = "card.cultivate.skill.upgrade"
	TASK_EVENT_COMPOUND             = "card.cultivate.compound"
	TASK_EVENT_CHARACTER_ALL        = "card.cultivate.character.*"
	TASK_EVENT_CHARACTER_BREAK      = "card.cultivate.character.break"
	TASK_EVENT_BREAKTHROUGH         = "card.cultivate.breakthrough"
	TASK_EVENT_LEVEL_UPGRADE        = "card.cultivate.level.upgrade"
	TASK_EVENT_USE_FOOD             = "card.cultivate.usefood"
	TASK_EVENT_CHANGE_HP            = "card.change.hp"
	TASK_EVENT_CURRENCY_SUB         = "currency.sub"
	TASK_EVENT_EQUIP_CREATE         = "equip.create"
	TASK_EVENT_EQUIP_BUILD          = "camp.equip.build"
	TASK_EVENT_EQUIP_LEVELUP        = "equip.levelup"
	TASK_EVENT_CARD_CREATE          = "card.create"
	TASK_EVENT_POOL_EXTRACT         = "cardpool.extract"
	TASK_EVENT_BUILDING_MAKE        = "camp.building.make"
	TASK_EVENT_BUILDING_CREATE      = "camp.building.create"
	TASK_EVENT_BUILDING_REWARD      = "camp.building.func.reward"
	TASK_EVENT_TREE_COLLECT         = "camp.tree.collect"
	TASK_EVENT_BUILDING_LEVELUP     = "camp.building.levelup"
	TASK_EVENT_BUILDING_UP_CARD     = "camp.func.up.card"
	TASK_EVENT_SKIN_ADD             = "card.skin.add"
	TASK_EVENT_CAMPAIGN_ENTER_LEVEL = "general.campaign.enter.level"
	TASK_EVENT_CAMPAIGN_CAR_SETTLE  = "general.campaign.car.settle"
	TASK_EVENT_CAMPAIGN_LEVEL       = "general.campaign.level"
	TASK_EVENT_TRAVEL_LEVEL_WIN     = "travel.level.win"
	TASK_EVENT_PLAYER_LOGIN         = "player.login"
	TASK_EVENT_ENTER_GAME           = "player.enter.game"
	TASK_EVENT_STAMINA_SUB          = "stamina.sub"
	TASK_EVENT_QUEST_COMPLETE       = "quest.complete"
	TASK_EVENT_STORY_FLAG_CHANGE    = "story.flag.change"
	TASK_EVENT_ROLE_LEVEL_CHANGE    = "role.level.change"
	TASK_EVENT_ITEM_DROP            = "item.drop"
	TASK_EVENT_ITEM_CONSUME         = "item.consume"
	TASK_EVENT_ACHIEVE_COMPLETE     = "achieve.complete"
	TASK_EVENT_DUTY_ACTIVE_CHANGE   = "duty.active.change"
	TASK_EVENT_TASK_COMPLETE        = "task.complete"
	TASK_EVENT_TASK_GROUP_COMPLETE  = "task.group.complete"
)

// --------------------- 任务类型统一监听 ---------------------

// RegisterTaskTypeHandler 对外统一监听任务类型的方法
func (m *TaskTypeMgr) RegisterTaskTypeHandler(handler event.IListener) {
	m.actor.eventManager.Listen(TASK_EVENT_MONSTER_BATTLE, handler)       // 1 2 3
	m.actor.eventManager.Listen(TASK_EVENT_LEVEL_COLLECT, handler)        // 11 12
	m.actor.eventManager.Listen(TASK_EVENT_LEVEL_BOX, handler)            // 31
	m.actor.eventManager.Listen(TASK_EVENT_LEVEL_WIN, handler)            // 91
	m.actor.eventManager.Listen(TASK_EVENT_BATTLE_WIN, handler)           // 92
	m.actor.eventManager.Listen(TASK_EVENT_SKILL_UPGRADE, handler)        // 101 105
	m.actor.eventManager.Listen(TASK_EVENT_COMPOUND, handler)             // 102 105
	m.actor.eventManager.Listen(TASK_EVENT_CHARACTER_ALL, handler)        // 103 105
	m.actor.eventManager.Listen(TASK_EVENT_BREAKTHROUGH, handler)         // 104 105
	m.actor.eventManager.Listen(TASK_EVENT_LEVEL_UPGRADE, handler)        // 121 105
	m.actor.eventManager.Listen(TASK_EVENT_USE_FOOD, handler)             // 111
	m.actor.eventManager.Listen(TASK_EVENT_CURRENCY_SUB, handler)         //
	m.actor.eventManager.Listen(TASK_EVENT_EQUIP_CREATE, handler)         // 201 202
	m.actor.eventManager.Listen(TASK_EVENT_EQUIP_BUILD, handler)          // 203
	m.actor.eventManager.Listen(TASK_EVENT_EQUIP_LEVELUP, handler)        // 211
	m.actor.eventManager.Listen(TASK_EVENT_CARD_CREATE, handler)          // 401 402 403
	m.actor.eventManager.Listen(TASK_EVENT_POOL_EXTRACT, handler)         // 407
	m.actor.eventManager.Listen(TASK_EVENT_BUILDING_CREATE, handler)      // 311
	m.actor.eventManager.Listen(TASK_EVENT_BUILDING_REWARD, handler)      // 312 313
	m.actor.eventManager.Listen(TASK_EVENT_TREE_COLLECT, handler)         // 301
	m.actor.eventManager.Listen(TASK_EVENT_BUILDING_LEVELUP, handler)     // 302 303
	m.actor.eventManager.Listen(TASK_EVENT_SKIN_ADD, handler)             // 404 405 406
	m.actor.eventManager.Listen(TASK_EVENT_CAMPAIGN_ENTER_LEVEL, handler) //
	m.actor.eventManager.Listen(TASK_EVENT_CAMPAIGN_CAR_SETTLE, handler)  //
	m.actor.eventManager.Listen(TASK_EVENT_PLAYER_LOGIN, handler)         // 501
	m.actor.eventManager.Listen(TASK_EVENT_STAMINA_SUB, handler)          // 112
	m.actor.eventManager.Listen(TASK_EVENT_ITEM_CONSUME, handler)
	m.actor.eventManager.Listen(TASK_EVENT_ROLE_LEVEL_CHANGE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_QUEST_COMPLETE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_CAMPAIGN_LEVEL, handler)
	m.actor.eventManager.Listen(TASK_EVENT_TRAVEL_LEVEL_WIN, handler)
	m.actor.eventManager.Listen(TASK_EVENT_BUILDING_UP_CARD, handler)
	m.actor.eventManager.Listen(TASK_EVENT_BUILDING_MAKE, handler)
	m.actor.eventManager.Listen(TASK_EVENT_CHARACTER_BREAK, handler)
}

// --------------------- 任务数据统一创建 ---------------------

func (m *TaskTypeMgr) CreateTaskInfoItemNew(cfg *excel.TaskCfg, canCreate bool) *cmd.TaskInfoItem {
	return m.CreateTaskInfoItem(cfg.Id, cfg.TaskType, cfg.TaskValue, cfg.TaskParam1, canCreate) // fixme 过期时间修正
}

// CreateTaskInfoItem 对外统一创建任务数据的方法
//
//	任务创建
//	-- 生涯型 	直接创建
//	-- 非生涯型 	判断canCreate  true则判断进度值，false则进度值强制为0
func (m *TaskTypeMgr) CreateTaskInfoItem(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	var task *cmd.TaskInfoItem
	switch typ {
	case TASK_TYPE_11, TASK_TYPE_12, TASK_TYPE_31, TASK_TYPE_91, TASK_TYPE_92, TASK_TYPE_507, TASK_TYPE_525,
		TASK_TYPE_104, TASK_TYPE_105, TASK_TYPE_41, TASK_TYPE_313, TASK_TYPE_408, TASK_TYPE_510, TASK_TYPE_527,
		TASK_TYPE_111, TASK_TYPE_112, TASK_TYPE_113, TASK_TYPE_201, TASK_TYPE_312, TASK_TYPE_502, TASK_TYPE_508,
		TASK_TYPE_202, TASK_TYPE_203, TASK_TYPE_211, TASK_TYPE_301, TASK_TYPE_1, TASK_TYPE_2,
		TASK_TYPE_3, TASK_TYPE_302, TASK_TYPE_303, TASK_TYPE_311, TASK_TYPE_407:
		task = m.createCommon(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_101:
		task = m.createType101(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_102:
		task = m.createType102(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_103:
		task = m.createType103(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_121:
		task = m.createType121(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_505:
		task = m.createType505(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_401:
		task = m.createType401(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_402:
		task = m.createType402(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_403:
		task = m.createType403(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_404:
		task = m.createType404(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_406:
		task = m.createType406(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_501:
		task = m.createType501(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_503:
		task = m.createType503(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_504:
		task = m.createType504(id, typ, target, params, canCreate, times...)
	case TASK_TYPE_521:
		task = m.createType521(id, typ, target, params, canCreate, times...)
	default:
		m.actor.Errorf("create task unrealized task type %d", typ)
		return nil
	}
	// 容错
	if task == nil {
		return nil
	}
	// 后续处理
	if task.CurValue >= task.TargetValue {
		task.CurValue = task.TargetValue
		task.Status = TASK_STATUS_COMPLETE
	}
	m.actor.Infof("create task id:%d type:%d", id, typ)
	return task
}

// 通用的构建数据方法
func (m *TaskTypeMgr) createCommon(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    0,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType101(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    0,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	// 特殊处理: 获取当前卡牌技能总等级
	if len(params) > 0 {
		task.CurValue = m.actor.CardHandler.GetCardSkillLvSum(uint32(params[0]))
	}
	return task
}

func (m *TaskTypeMgr) createType102(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    0,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	// 特殊处理: 获取当前卡牌潜力等级
	if len(params) > 0 {
		card, _ := m.actor.CardHandler.GetCard(uint32(params[0]))
		if card != nil {
			task.CurValue = int32(card.AwakenLevel)
		}
	}
	return task
}

func (m *TaskTypeMgr) createType103(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    0,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	// 特殊处理: 获取当前卡牌性格等级
	if len(params) > 0 {
		card, _ := m.actor.CardHandler.GetCard(uint32(params[0]))
		if card != nil {
			task.CurValue = int32(card.CharacterLevel)
		}
	}
	return task
}

func (m *TaskTypeMgr) createType121(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    0,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	// 特殊处理: 获取当前卡牌等级
	if len(params) > 0 {
		card, _ := m.actor.CardHandler.GetCard(uint32(params[0]))
		if card != nil {
			task.CurValue = int32(card.CardLevel)
		}
	}
	return task
}

func (m *TaskTypeMgr) createType505(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	if len(params) == 0 {
		m.actor.Debugf("task config err %d", typ)
		return nil
	}

	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    m.actor.CardHandler.GetCardCountByLevel(params[0]),
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType401(id, typ, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	if len(params) == 0 {
		m.actor.Debugf("task config err %d", typ)
		return nil
	}

	var cur int32
	if m.actor.CardHandler.IsExistCard(uint32(params[0])) {
		cur = 1
	}

	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    cur,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType402(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {

	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    int32(m.actor.CardHandler.GetCardCount()),
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType403(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	if len(params) == 0 {
		m.actor.Debugf("task config err %d", typ)
		return nil
	}

	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    m.actor.CardHandler.GetCardCountByQuality(params[0]),
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType404(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	if len(params) == 0 {
		m.actor.Debugf("task config err %d", typ)
		return nil
	}
	cur := int32(0)
	if len(params) > 0 && m.actor.SkinHandler.IsExistSkinId(params[0]) {
		cur = 1
	}

	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    cur,
		TargetValue: 1,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType406(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {

	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    m.actor.SkinHandler.getSkinCount(),
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType504(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    int32(m.actor.LoginHandler.getRoleLevel()),
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType521(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	if len(params) == 0 {
		m.actor.Debugf("task config err %d", typ)
		return nil
	}
	var cur int32
	// if m.actor.QuestHandler.checkQuestFinish(params[0]) {
	// 	cur = 1
	// }
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    cur,
		TargetValue: 1,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType501(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    m.actor.LoginHandler.getLoginDay(),
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
		CondType:    TASK_COND_TYPE_LIFE,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

func (m *TaskTypeMgr) createType503(id int32, typ int32, target int32, params []int32, canCreate bool, times ...int32) *cmd.TaskInfoItem {
	// 是否真正创建
	var cur int32
	if canCreate {
		cur = 1
	}
	task := &cmd.TaskInfoItem{
		Id:          id,
		CondId:      typ,
		CurValue:    cur,
		TargetValue: target,
		Status:      TASK_STATUS_DOING,
		Create:      time.Now().Unix(),
		Params:      params,
	}
	if len(times) > 0 {
		task.ExpireTs = task.Create + int64(times[0])
	}
	return task
}

// --------------------- 任务类型统一达成 ---------------------

// CheckTaskConditionComplete 对外统一的完成任务check方法 bool 状态是否有变化
//
//	任务完成
//	-- 生涯型	直接增加
//	-- 非生涯型	判断canCheck	 true则增加进度值，false则进度值保持不变
func (m *TaskTypeMgr) CheckTaskConditionComplete(task *cmd.TaskInfoItem, e event.IEvent, canCheck bool) bool {
	if task == nil {
		m.actor.Debug("task is nil")
		return false
	}
	// 只处理进行中
	if task.Status != TASK_STATUS_DOING {
		return false
	}
	// 非生涯+canCheck=false
	if task.CondType == TASK_COND_TYPE_NORMAL && !canCheck {
		return false
	}
	oldValue := task.CurValue
	switch task.CondId {
	case TASK_TYPE_11, TASK_TYPE_31, TASK_TYPE_41, TASK_TYPE_92,
		TASK_TYPE_104, TASK_TYPE_105, TASK_TYPE_508, TASK_TYPE_503,
		TASK_TYPE_211, TASK_TYPE_301, TASK_TYPE_302, TASK_TYPE_501,
		TASK_TYPE_402, TASK_TYPE_406:
		m.checkCommon(task, e)
	case TASK_TYPE_111, TASK_TYPE_112, TASK_TYPE_201, TASK_TYPE_203, TASK_TYPE_311, TASK_TYPE_407:
		m.checkCommonCount(task, e)
	case TASK_TYPE_1:
		m.checkType1(task, e)
	case TASK_TYPE_2:
		m.checkType2(task, e)
	case TASK_TYPE_3:
		m.checkType3(task, e)
	case TASK_TYPE_12:
		m.checkType12(task, e)
	case TASK_TYPE_91:
		m.checkType91(task, e)
	case TASK_TYPE_101:
		m.checkType101(task, e)
	case TASK_TYPE_102, TASK_TYPE_103, TASK_TYPE_121:
		m.checkType121(task, e)
	case TASK_TYPE_113:
		m.checkType113(task, e)
	case TASK_TYPE_202:
		m.checkType202(task, e)
	case TASK_TYPE_303:
		m.checkType303(task, e)
	case TASK_TYPE_401:
		m.checkType401(task, e)
	case TASK_TYPE_403:
		m.checkType403(task, e)
	case TASK_TYPE_404:
		m.checkType404(task, e)
	case TASK_TYPE_408:
		m.checkType408(task, e)
	case TASK_TYPE_502:
		m.checkType502(task, e)
	case TASK_TYPE_504:
		m.checkType504(task, e)
	case TASK_TYPE_505:
		m.checkType505(task, e)
	case TASK_TYPE_507:
		m.checkType507(task, e)
	case TASK_TYPE_510:
		m.checkType510(task, e)
	case TASK_TYPE_511:
		m.checkType511(task, e)
	case TASK_TYPE_517:
		m.checkType517(task, e)
	case TASK_TYPE_518:
		m.checkType518(task, e)
	case TASK_TYPE_521:
		m.checkType521(task, e)
	case TASK_TYPE_525:
		m.checkType525(task, e)
	case TASK_TYPE_527:
		m.checkType527(task, e)
	default:
		m.actor.Errorf("complete task unrealized task type %d", task.CondId)
		return false
	}
	// 后续处理
	change := oldValue != task.CurValue
	if task.CurValue >= task.TargetValue {
		task.CurValue = task.TargetValue
		task.Status = TASK_STATUS_COMPLETE
	}
	// 任务完成了，生涯计数处理
	if change && task.Status == TASK_STATUS_COMPLETE {
		m.handleTaskAchieveChange(task)
	}
	m.actor.Infof("handle task %+v", task)
	return change
}

// 任务完成，生涯计数处理
func (m *TaskTypeMgr) handleTaskAchieveChange(task *cmd.TaskInfoItem) {
	// 任务处理
	err := m.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_TASK_COMPLETE, []int32{}, map[string]interface{}{
		"task_id": task.Id,
	}))
	if err != nil {
		m.actor.Error(err)
	}
	// 任务组处理
	excel.GetTaskgroupMgr().Foreach(func(cfg *excel.TaskgroupCfg) bool {
		// for _, id := range cfg.TaskId {
		// 	num := m.actor.AchieveHandler.GetAchieveNum(buildTaskKey(int(id)))
		// 	if num == 0 {
		// 		return true
		// 	}
		// }
		err = m.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_TASK_GROUP_COMPLETE, []int32{}, map[string]interface{}{
			"group_id": cfg.Id,
		}))
		if err != nil {
			m.actor.Error(err)
		}
		return true
	}, true)
}

// 通用的check逻辑
func (m *TaskTypeMgr) checkCommon(task *cmd.TaskInfoItem, e event.IEvent) {
	task.CurValue += 1
}

// 增加指定数量的通用逻辑
func (m *TaskTypeMgr) checkCommonCount(task *cmd.TaskInfoItem, e event.IEvent) {
	v, ok := e.Get("count").(int32)
	if !ok {
		return
	}
	task.CurValue += v
}

func (m *TaskTypeMgr) checkType1(task *cmd.TaskInfoItem, e event.IEvent) {
	monsterIds, ok := e.Get("monster_ids").([]int32)
	if !ok {
		return
	}
	task.CurValue += int32(len(monsterIds))
}

func (m *TaskTypeMgr) checkType2(task *cmd.TaskInfoItem, e event.IEvent) {
	monsterIds, ok := e.Get("monster_ids").([]int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	for _, id := range monsterIds {
		if id == task.Params[0] {
			task.CurValue++
		}
	}
}

func (m *TaskTypeMgr) checkType3(task *cmd.TaskInfoItem, e event.IEvent) {
	// 不是大地图
	if typ, ok := e.Get("type").(bool); ok && typ {
		return
	}

	task.CurValue++
}

func (m *TaskTypeMgr) checkType12(task *cmd.TaskInfoItem, e event.IEvent) {
	resId, ok := e.Get("resource_id").(int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if resId == task.Params[0] {
		task.CurValue += 1
	}
}

func (m *TaskTypeMgr) checkType91(task *cmd.TaskInfoItem, e event.IEvent) {
	levelId, ok := e.Get("level_id").(int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if levelId == task.Params[0] {
		task.CurValue += 1
	}
}

func (m *TaskTypeMgr) checkType101(task *cmd.TaskInfoItem, e event.IEvent) {
	if len(task.Params) == 0 {
		task.CurValue += 1
	} else {
		// 特殊处理: 卡牌技能总等级
		if cardId, ok := e.Get("card_id").(uint32); ok && cardId == uint32(task.Params[0]) {
			task.CurValue += 1
		}
	}
}

func (m *TaskTypeMgr) checkType121(task *cmd.TaskInfoItem, e event.IEvent) {
	if len(task.Params) == 0 {
		task.CurValue += 1
	} else {
		// 特殊处理: 卡牌等级
		lv, ok := e.Get("level").(uint32)
		cardId, ok1 := e.Get("card_id").(uint32)
		if ok && ok1 && cardId == uint32(task.Params[0]) {
			task.CurValue = int32(lv)
		}
	}
}

func (m *TaskTypeMgr) checkType113(task *cmd.TaskInfoItem, e event.IEvent) {
	items, ok := e.Get("items").(map[int32]int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	task.CurValue += items[task.Params[0]]
}

func (m *TaskTypeMgr) checkType202(task *cmd.TaskInfoItem, e event.IEvent) {
	quality, ok := e.Get("quality").(map[int32]int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	task.CurValue += quality[task.Params[0]]
}

func (m *TaskTypeMgr) checkType303(task *cmd.TaskInfoItem, e event.IEvent) {
	buildId, ok := e.Get("build_id").(int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if buildId == task.Params[0] {
		task.CurValue++
	}
}

func (m *TaskTypeMgr) checkType401(task *cmd.TaskInfoItem, e event.IEvent) {
	cardId, ok := e.Get("cardId").(uint32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if int32(cardId) == task.Params[0] {
		task.CurValue = task.TargetValue
	}
}

func (m *TaskTypeMgr) checkType403(task *cmd.TaskInfoItem, e event.IEvent) {
	rarity, ok := e.Get("rarity").(int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if rarity == task.Params[0] {
		task.CurValue += 1
	}
}

func (m *TaskTypeMgr) checkType408(task *cmd.TaskInfoItem, e event.IEvent) {
	poolId, ok := e.Get("pool_id").(int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if poolId == task.Params[0] {
		if count, ok1 := e.Get("count").(int32); ok1 {
			task.CurValue += count
		}
	}
}

func (m *TaskTypeMgr) checkType404(task *cmd.TaskInfoItem, e event.IEvent) {
	rarity, ok := e.Get("skin_id").(int32)
	if !ok || len(task.Params) == 0 {
		return
	}
	if rarity == task.Params[0] {
		task.CurValue += 1
	}
}

func (m *TaskTypeMgr) checkType502(task *cmd.TaskInfoItem, e event.IEvent) {
	if m.actor.DutyHandler.CheckDailyTaskAllComplete() {
		task.CurValue = task.TargetValue
	}
}

func (m *TaskTypeMgr) checkType504(task *cmd.TaskInfoItem, e event.IEvent) {
	level, ok := e.Get("level").(int32)
	if !ok {
		return
	}
	task.CurValue = level
}

func (m *TaskTypeMgr) checkType505(task *cmd.TaskInfoItem, e event.IEvent) {
	if len(task.Params) == 0 {
		return
	}
	task.CurValue = m.actor.CardHandler.GetCardCountByLevel(task.Params[0])
}

func (m *TaskTypeMgr) checkType507(task *cmd.TaskInfoItem, e event.IEvent) {
	typ, ok := e.Get("type").(int32)
	if !ok {
		return
	}

	if typ == task.Params[0] {
		task.CurValue++
	}
}

func (m *TaskTypeMgr) checkType510(task *cmd.TaskInfoItem, e event.IEvent) {
	reward, ok := e.Get("reward").(map[int32]int32)
	if !ok {
		return
	}
	var f bool
	for id := range reward {
		if id == task.Params[0] {
			f = true
		}
	}
	if f {
		task.CurValue++
	}
}

func (m *TaskTypeMgr) checkType511(task *cmd.TaskInfoItem, e event.IEvent) {
	buildingId, ok := e.Get("build_id").(int32)
	if ok && buildingId == task.Params[0] {
		level, ok1 := e.Get("level").(int32)
		if ok1 {
			task.CurValue = level
		}
	}
}

func (m *TaskTypeMgr) checkType517(task *cmd.TaskInfoItem, e event.IEvent) {
	buildId, ok := e.Get("build_id").(int32)
	if !ok {
		return
	}
	if task.Params[0] == buildId {
		task.CurValue = 1
	}
}

func (m *TaskTypeMgr) checkType518(task *cmd.TaskInfoItem, e event.IEvent) {
	buildId, ok := e.Get("build_id").(int32)
	cardId, ok1 := e.Get("card_id").(int32)
	if !ok || !ok1 {
		return
	}
	if task.Params[0] == cardId && task.Params[1] == buildId {
		task.CurValue = 1
	}
}

func (m *TaskTypeMgr) checkType521(task *cmd.TaskInfoItem, e event.IEvent) {
	questId, ok := e.Get("quest_id").(int)
	if ok && task.Params[0] == int32(questId) {
		task.CurValue = 1
	}
}

func (m *TaskTypeMgr) checkType525(task *cmd.TaskInfoItem, e event.IEvent) {
	reward, ok := e.Get("reward").(map[int32]int32)
	if !ok {
		return
	}
	for itemId, num := range reward {
		cfg := excel.GetItemMgr().GetById(itemId)
		if cfg != nil && cfg.Type == task.Params[0] && cfg.SubType == task.Params[1] {
			task.CurValue += num
		}
	}
}

func (m *TaskTypeMgr) checkType527(task *cmd.TaskInfoItem, e event.IEvent) {
	reward, ok := e.Get("reward").(map[int32]int32)
	if !ok {
		return
	}
	for itemId := range reward {
		cfg := excel.GetItemMgr().GetById(itemId)
		if cfg != nil && cfg.Type == task.Params[0] && cfg.SubType == task.Params[1] {
			task.CurValue++
		}
	}
}

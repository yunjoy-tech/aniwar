package useractor

import (
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"google.golang.org/protobuf/proto"
)

type AchieveHandler struct {
	*UABaseHandler
}

func NewAchieveHandler(actor *UserActor) *AchieveHandler {
	h := &AchieveHandler{UABaseHandler: NewUABaseHandler(actor, "AchieveHandler")}
	h.ChildHandler = h

	return h
}

// Init 初始化模块数据
func (h *AchieveHandler) Init() error {
	// 初始化
	h.actor.Data.AchieveData = &cmd.PUserAchieves{
		Createtime:     time.Now().Unix(),
		Achieves:       make(map[string]int32),                // key=成就id,value=总值 (key拼接规则：groupId-conditionId-paramId) 完成的数量
		SectionReceive: make(map[string]*cmd.PAchieveReceive), // 成就档位领取记录 (key拼接规则：groupId-conditionId)
		GroupReceive:   make(map[string]int32),                //成就集领取记录 (key拼接规则：groupId)
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init achieve data success. player: %s", h.actor.ID())
	return nil
}

func (h *AchieveHandler) EnterGame() error {
	return nil
}

func (h *AchieveHandler) DailyRefresh() error {
	return nil
}

func (h *AchieveHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUserAchieves); ok {
		h.actor.Data.AchieveData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *AchieveHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserAchieve(h.actor.ID()), h.actor.Data.AchieveData
}

// AddAchieveNum
//
//	@Description: 增加成就记录值,并判断是否完成
//	@receiver h
//	@param key 记录的key
//	@param addNum 本次新增值
func (h *AchieveHandler) AddAchieveNum(key string, addNum int32) {
	data := h.actor.GetAchieveData()
	data.Achieves[key] += addNum
	if err := h.SaveDB(); err != nil {
		h.Error(err)
		return
	}

	h.Debugf("AddAchieveNum success. key: %s, addNum: %d", key, addNum)
	//return data.Achieves[key] >= cfg.AchievementNum
}

// AddAchieveNumOnce 同一个Key之加一次
func (h *AchieveHandler) AddAchieveNumOnce(key string, addNum int32) bool {
	data := h.actor.GetAchieveData()
	if _, ok := data.Achieves[key]; ok {
		return false
	}
	data.Achieves[key] += addNum
	if err := h.SaveDB(); err != nil {
		h.Error(err)
		return false
	}

	h.Debugf("AddAchieveNum success. key: %s, addNum: %d", key, addNum)
	//return data.Achieves[key] >= cfg.AchievementNum
	return true
}

// GetAchieveNum
//
//	@Description: 查询成就记录值
//	@receiver h
//	@param key 查询的key
//	@return int32 记录值，key不存在返回0
func (h *AchieveHandler) GetAchieveNum(key string) int32 {
	data := h.actor.GetAchieveData()
	return data.Achieves[key]
}

// CheckSectionComplete
//
//	@Description: 校验指定key的档位值是否达成
//	@receiver h
//	@param key 记录的key
//	@param section 档位达成值
//	@return bool 达成返回true，否则返回false
func (h *AchieveHandler) CheckSectionComplete(key string, section int32) bool {
	data := h.actor.GetAchieveData()
	val, ok := data.Achieves[key]
	if !ok {
		return false
	}
	return val >= section
}

// LogSectionReceive
//
//	@Description: 成就档位奖励领取记录
//	@receiver h
//	@param key 记录的key
//	@param section 档位值
func (h *AchieveHandler) LogSectionReceive(key string, section int32) error {
	data := h.actor.GetAchieveData()
	receive, ok := data.SectionReceive[key]
	if !ok {
		receive = &cmd.PAchieveReceive{Section: make(map[int32]int32)}
	}
	receive.Section[section] = 0
	data.SectionReceive[key] = receive
	if err := h.SaveDB(); err != nil {
		return err
	}
	return nil
}

// CheckSectionReceived
//
//	@Description: 检查给定key的档位是否已领取
//	@receiver h
//	@param key 记录的key
//	@param section 成就id
//	@return bool 已领取返回true，未领取返回false
func (h *AchieveHandler) CheckSectionReceived(key string, section int32) bool {
	data := h.actor.GetAchieveData()
	receive, ok := data.SectionReceive[key]
	if !ok {
		return false
	}
	_, ok = receive.Section[section]
	return ok
}

// LogGroupReceive
//
//	@Description: 成就集奖励领取记录
//	@receiver h
//	@param key 成就集的key
//	@return error
func (h *AchieveHandler) LogGroupReceive(key string) error {
	data := h.actor.GetAchieveData()
	data.GroupReceive[key] = 0
	if err := h.SaveDB(); err != nil {
		return err
	}
	return nil
}

// CheckGroupReceived
//
//	@Description: 检查给定key的成就集是否已领取
//	@receiver h
//	@param key 成就集的key
//	@return bool 已领取返回true，未领取返回false
func (h *AchieveHandler) CheckGroupReceived(key string) bool {
	data := h.actor.GetAchieveData()
	_, ok := data.GroupReceive[key]
	return ok
}

func (h *AchieveLevelHandler) GetAchieveCfg(achievementsId, conditionId int32, params string) []*excel.AchievementDataCfg {
	achievements := make([]*excel.AchievementDataCfg, 0)
	excel.GetAchievementDataMgr().Foreach(func(cfg *excel.AchievementDataCfg) bool {
		// 过滤掉已经达成的 h.getAchieveNum(cfg)>=cfg.AchievementNum
		if cfg.AchievementsId == achievementsId && cfg.FinishCondition == conditionId && h.getAchieveNum(cfg) < cfg.AchievementNum {
			if len(cfg.FinishParam) == 0 && params == "0" {
				achievements = append(achievements, cfg)
			}
			for _, v := range cfg.FinishParam {
				if v == params {
					achievements = append(achievements, cfg)
					break
				}
			}
		}
		return true
	}, true)
	return achievements
}

// GetCompleteCount 获取已达成的成就总数量
func (h *AchieveHandler) GetCompleteCount() int32 {
	//data := h.actor.GetAchieveData()
	//count := 0
	//for _, v := range data.SectionReceive {
	//	count += len(v.Section)
	//}
	achievementCfg := h.actor.AchieveLevelHandler.GetAllAchievementCfg()
	count := int32(0)
	for _, cfg := range achievementCfg {
		if h.actor.AchieveLevelHandler.getAchieveNum(cfg) >= cfg.AchievementNum {
			count++
		}
	}
	return count
}

// 成就集的存档key生成规则 "成就组id-条件id-参数id"
func buildAchieveKey(groupId, condition int32, param string) string {
	return fmt.Sprintf("%v-%v-%v", groupId, condition, param)
}

// 成就档位领奖存档key生成规则 "groupId-conditionId"
func buildSectionKey(groupId, conditionId int32) string {
	return fmt.Sprintf("%v-%v", groupId, conditionId)
}

// 玩家生涯成就数据key，非成就任务key
const (
	CARD_MAX_BREAKTHROUGH = "card_max_breakthrough" // 角色最大突破次数
	CARD_MAX_LEVEL        = "card_max_level"        // 角色最大突破次数
)

// 道具key
func buildItemKey(itemId int) string {
	return fmt.Sprintf("item-%v", itemId)
}

// 道具使用次数key
func buildItemCountKey(itemId int) string {
	return fmt.Sprintf("itemcount-%v", itemId)
}

// 关卡key
func buildLevelKey(levelId int) string {
	return fmt.Sprintf("level-%v", levelId)
}

// 关卡类型key
func buildLevelTypeKey(levelType int) string {
	return fmt.Sprintf("leveltype-%v", levelType)
}

// 任务key
func buildTaskKey(taskId int) string {
	return fmt.Sprintf("task-%v", taskId)
}

// 任务组key
func buildTaskGroupKey(taskGroupId int) string {
	return fmt.Sprintf("taskgroup-%v", taskGroupId)
}

// 通用成就数据计数，非成就任务
func (h *AchieveHandler) handleCommonAchieveData(e event.IEvent) error {
	switch e.Name() {
	case TASK_EVENT_BREAKTHROUGH:
		h.handleCardMaxBreakthrough(e)
	case TASK_EVENT_LEVEL_UPGRADE:
		h.handleCardMaxLevel(e)
	case TASK_EVENT_ITEM_CONSUME:
		h.handleItemsConsume(e)
	case TASK_EVENT_LEVEL_WIN:
		h.handleLevelWin(e)
	case TASK_EVENT_CAMPAIGN_LEVEL, TASK_EVENT_TRAVEL_LEVEL_WIN:
		h.handleLevelType(e)
	case TASK_EVENT_TASK_COMPLETE:
		h.handleTaskComplete(e)
	case TASK_EVENT_TASK_GROUP_COMPLETE:
		h.handleTaskGroupComplete(e)
	default:
		return nil
	}
	return nil
}

// 道具使用计数
func (h *AchieveHandler) handleItemsConsume(e event.IEvent) {
	items, ok := e.Get("items").(map[int32]int32)
	if !ok {
		return
	}
	for k, v := range items {
		h.AddAchieveNum(buildItemKey(int(k)), v)
		h.AddAchieveNum(buildItemCountKey(int(k)), 1)
	}
}

// 通关关卡x次
func (h *AchieveHandler) handleLevelWin(e event.IEvent) {
	levelId, ok := e.Get("level_id").(int32)
	if !ok {
		return
	}
	h.AddAchieveNum(buildLevelKey(int(levelId)), 1)
}

func (h *AchieveHandler) handleLevelType(e event.IEvent) {
	levelType, ok := e.Get("type").(int32)
	if !ok {
		return
	}
	h.AddAchieveNum(buildLevelTypeKey(int(levelType)), 1)
}

func (h *AchieveHandler) handleTaskComplete(e event.IEvent) {
	taskId, ok := e.Get("task_id").(int32)
	if !ok {
		return
	}
	h.AddAchieveNum(buildTaskKey(int(taskId)), 1)
}

func (h *AchieveHandler) handleTaskGroupComplete(e event.IEvent) {
	groupId, ok := e.Get("group_id").(int32)
	if !ok {
		return
	}
	h.AddAchieveNum(buildTaskGroupKey(int(groupId)), 1)
}

// 卡牌最大突破次数
func (h *AchieveHandler) handleCardMaxBreakthrough(e event.IEvent) {
	level, ok := e.Get("level").(int32)
	if !ok {
		return
	}
	cur := h.GetAchieveNum(CARD_MAX_BREAKTHROUGH)
	diff := level - cur
	if diff > 0 {
		h.AddAchieveNum(CARD_MAX_BREAKTHROUGH, diff)
	}
}

// 卡牌最大等级
func (h *AchieveHandler) handleCardMaxLevel(e event.IEvent) {
	level, ok := e.Get("level").(uint32)
	if !ok {
		return
	}
	cur := h.GetAchieveNum(CARD_MAX_LEVEL)
	diff := int32(level) - cur
	if diff > 0 {
		h.AddAchieveNum(CARD_MAX_LEVEL, diff)
	}
}

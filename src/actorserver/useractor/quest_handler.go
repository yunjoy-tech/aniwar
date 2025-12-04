package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"strconv"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

const (
	QUEST_TRIGGER_TYPE_0 = 0 // 触发类型0	表示立即触发
	QUEST_TRIGGER_TYPE_1 = 1 // 触发类型1	是否完成过任务id，任务id
	QUEST_TRIGGER_TYPE_2 = 2 // 触发类型2	获得一个特定的flag：flag open_ep1_hidden_chest
	QUEST_TRIGGER_TYPE_3 = 3 // 触发类型3	玩家等级，等级：10
	QUEST_TRIGGER_TYPE_4 = 4 // 触发类型4	获得了某个id的道具，道具id：40001
	QUEST_TRIGGER_TYPE_5 = 5 // 触发类型5	进入了某个关卡
	QUEST_STEP_TYPE_2    = 2 // 步骤类型2 子步骤完成或者事件组完成
	QUEST_OBJECT_TYPE_2  = 2 // 物件类型2 提交材料类型
)

type QuestHandler struct {
	*UABaseHandler
}

func NewQuestHandler(actor *UserActor) *QuestHandler {
	h := &QuestHandler{UABaseHandler: NewUABaseHandler(actor, "QuestHandler")}
	h.ChildHandler = h

	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CompleteQuestObjectReq), h.CompleteQuestObjectReq)

	return h
}

// Init 初始化模块数据
func (h *QuestHandler) Init() error {
	// 初始化
	h.actor.Data.QuestData = &cmd.PQuestData{
		Createtime:     time.Now().Unix(),
		CompleteQuests: make([]int32, 0),
		OpenQuests:     make(map[int32]*cmd.PCommonQuestInfo),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init quest data success. player: %s", h.actor.ID())
	return nil
}

func (h *QuestHandler) EnterGame() error {
	return nil
}

func (h *QuestHandler) DailyRefresh() error {
	return nil
}

func (h *QuestHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PQuestData); ok {
		h.actor.Data.QuestData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *QuestHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserQuestInfo(h.actor.ID()), h.actor.Data.QuestData
}

func (h *QuestHandler) buildQuestInfo() *cmd.PQuestInfo {
	questData := h.actor.GetQuestData()
	h.tryFixOldData(questData)

	_, err := h.tryCreateQuest(questData)
	if err != nil {
		h.Error(err)
	}
	openQuests := make([]*cmd.PCommonQuestInfo, 0)
	for _, v := range questData.OpenQuests {
		openQuests = append(openQuests, v)
	}

	return &cmd.PQuestInfo{
		CompleteQuests: questData.CompleteQuests,
		OpenQuests:     openQuests,
	}
}

func (h *QuestHandler) tryFixOldData(questData *cmd.PQuestData) {
	// 已完成的任务
	questData.CompleteQuests = checkConfig(questData.CompleteQuests, 1)

	for _, v := range questData.OpenQuests {
		// 当前任务没了,直接清除任务
		cfg := excel.GetQuestMgr().GetById(v.QuestId)
		if cfg == nil {
			delete(questData.OpenQuests, v.QuestId)
			continue
		}
		// 当前步骤没了，从第一个步骤开始
		stepCfg := excel.GetQuestStepMgr().GetById(v.StepId)
		if stepCfg == nil {
			v.StepId = cfg.FirstStep
			v.CompleteSteps = make([]int32, 0)
			v.CompleteObject = make([]int32, 0)
			v.Progress = 0
			continue
		}

		// 已完成的步骤没了
		v.CompleteSteps = checkConfig(v.CompleteSteps, 2)
		// 已完成的物件没了
		v.CompleteObject = checkConfig(v.CompleteObject, 3)
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		h.Error(err)
	}
}

// 配置容错检查 1=任务,2=步骤,3=物件
func checkConfig(ids []int32, typ int32) []int32 {
	temp := make([]int32, 0)
	for _, v := range ids {
		switch typ {
		case 1:
			if cfg := excel.GetQuestMgr().GetById(v); cfg != nil {
				temp = append(temp, v)
			}
		case 2:
			if cfg := excel.GetQuestStepMgr().GetById(v); cfg != nil {
				temp = append(temp, v)
			}
		case 3:
			if cfg := excel.GetQuestObjectMgr().GetById(v); cfg != nil {
				temp = append(temp, v)
			}
		}
	}
	logger.Infof("尝试修正废弃剧情数据 type: %v, before: %v, after: %v", typ, ids, temp)
	return temp
}

func (h *QuestHandler) TryCreateQuest(e event.IEvent) error {
	questData := h.actor.GetQuestData()
	quest, err := h.tryCreateQuest(questData)
	if err != nil {
		h.Error(err)
		return nil
	}
	h.actor.comData.GetQuestData().OpenQuests = append(h.actor.comData.GetQuestData().OpenQuests, quest...)
	return nil
}

// tryCreateQuest
//
//	@Description: 尝试接取剧情任务
//	@receiver h
//	@param questData 剧情任务数据
//	@return []*cmd.PCommonQuestInfo 返回解锁的新剧情
//	@return error
func (h *QuestHandler) tryCreateQuest(questData *cmd.PQuestData) ([]*cmd.PCommonQuestInfo, error) {
	var saveFlag bool
	var newQuests = make([]*cmd.PCommonQuestInfo, 0)

	tempComplete := make(map[int32]int32)
	for _, v := range questData.CompleteQuests {
		tempComplete[v] = v
	}

	excel.GetQuestMgr().Foreach(func(cfg *excel.QuestCfg) bool {
		// 已完成
		if _, ok := tempComplete[cfg.Id]; ok {
			return true
		}
		// 进行中
		if _, ok := questData.OpenQuests[cfg.Id]; ok {
			return true
		}

		// 是否满足解锁条件
		createFlag := false
		switch cfg.TriggerType {
		case QUEST_TRIGGER_TYPE_0:
			createFlag = true
		case QUEST_TRIGGER_TYPE_1:
			id, err := strconv.Atoi(cfg.TriggerParam)
			if err != nil {
				h.Error(err)
				return true
			}
			if _, ok := tempComplete[int32(id)]; ok {
				createFlag = true
			}
		case QUEST_TRIGGER_TYPE_2:
			if h.actor.StoryFlagHandler.checkExistFlags(cfg.TriggerParam) {
				createFlag = true
			}
		case QUEST_TRIGGER_TYPE_3:
			lv, err := strconv.Atoi(cfg.TriggerParam)
			if err != nil {
				h.Error(err)
				return true
			}
			if h.actor.LoginHandler.checkPlayerLevel(lv) {
				createFlag = true
			}
		case QUEST_TRIGGER_TYPE_4:
			itemId, err := strconv.Atoi(cfg.TriggerParam)
			if err != nil {
				h.Error(err)
				return true
			}
			if GetConsumeMgr(h.actor).CheckEnough(int32(itemId), 1) {
				createFlag = true
			}
		case QUEST_TRIGGER_TYPE_5:
			levelId, err := strconv.Atoi(cfg.TriggerParam)
			if err != nil {
				h.Error(err)
				return true
			}
			if h.actor.ChapterHandler.GetCurrLevelId() == int32(levelId) {
				createFlag = true
			}

		default:
			return true
		}

		// 创建数据
		if createFlag {
			quest, err, _ := h.createQuest(cfg.Id)
			if err != nil {
				h.Errorf("create quest failed. %v", err)
				return true
			}
			questData.OpenQuests[quest.QuestId] = quest
			saveFlag = true
			newQuests = append(newQuests, quest)
		}
		return true
	}, true)

	if saveFlag {
		if err := h.SaveDB(); err != nil {
			return nil, err
		}
	}

	h.Infof("tryCreateQuest newQuests:%+v", newQuests)
	return newQuests, nil
}

// 尝试刷新步骤进度
// @param 事件组id列表
func (h *QuestHandler) TryRefreshProgress(params []int32, commonData *clidto.Comdata) (*cmd.DropChange, []*cmd.PCommonQuestInfo, []int32, []*cmd.MappointEvent, error, cmd.ErrorCode) {
	var (
		dropChanges = &cmd.DropChange{}
		openQuests  = make([]*cmd.PCommonQuestInfo, 0)
		completeIds = make([]int32, 0)
		events      = make([]*cmd.MappointEvent, 0)
	)

	saveFlag := false
	questData := h.actor.GetQuestData()
	tempParam := make(map[int32]int32)
	for _, v := range params {
		tempParam[v] = v
	}

	for _, openQuest := range questData.OpenQuests {
		// 是否需要刷新
		questStepCfg := excel.GetQuestStepMgr().GetById(openQuest.StepId)
		if questStepCfg == nil || questStepCfg.QuestStepType != QUEST_STEP_TYPE_2 || questStepCfg.EventGroupFinishId == 0 {
			continue
		}

		// 事件组判定
		if _, ok := tempParam[questStepCfg.EventGroupFinishId]; !ok {
			continue
		}

		openQuest.Progress++
		// 完成了当前步骤
		if openQuest.Progress >= questStepCfg.MaxProgress {
			openQuest.Progress = 0
			// 最后一步
			if questStepCfg.Isfinalstep == 1 {
				dropChange, quest, oldId, e, err, code := h.subCompleteQuest(questData, questStepCfg.QuestId, []string{}, commonData)
				if err != nil {
					return &cmd.DropChange{}, nil, nil, nil, err, code
				}
				mergeDropChange(dropChanges, dropChange)
				openQuests = append(openQuests, quest...)
				completeIds = append(completeIds, oldId)
				events = append(events, e...)
			} else {
				if len(questStepCfg.NextStep) == 0 {
					dropChange, quest, oldId, e, err, code := h.subCompleteSonStep(questData, questStepCfg, 0, []string{}, commonData)
					if err != nil {
						return &cmd.DropChange{}, nil, nil, nil, err, code
					}
					mergeDropChange(dropChanges, dropChange)
					openQuests = append(openQuests, quest...)
					completeIds = append(completeIds, oldId)
					events = append(events, e...)
				} else {
					// 直接完成主步骤，尝试开启下一个
					h.subCompleteStep(openQuest, questStepCfg, 0, []string{}, commonData)
				}
				openQuests = append(openQuests, openQuest)
			}
		} else {
			openQuests = append(openQuests, openQuest)
		}
		saveFlag = true
	}

	if saveFlag {
		if err := h.SaveDB(); err != nil {
			return &cmd.DropChange{}, nil, nil, nil, err, cmd.ErrorCode_SaveDBError
		}
	}

	h.Debugf("TryRefreshProgress params:%v", params)
	return dropChanges, openQuests, completeIds, events, nil, 0
}

func (h *QuestHandler) CompleteQuestObjectReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_CompleteQuestObjectReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	objCfg := excel.GetQuestObjectMgr().GetById(req.ObjectId)
	if objCfg == nil {
		return nil, fmt.Errorf("config not found %d", req.ObjectId), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// check
	data := h.actor.GetQuestData()
	openQuest := data.OpenQuests[objCfg.QuestId]
	if openQuest == nil {
		return nil, fmt.Errorf("quest not open"), int32(cmd.ErrorCode_QuestNotOpen)
	}

	// 事件id配置判断
	if objCfg.EventId > 0 {
		return nil, fmt.Errorf("objectcfg eventId not 0"), int32(cmd.ErrorCode_InvalidParam)
	}
	err, errCode := h.checkQuestCondition(openQuest, objCfg)
	if errCode != cmd.ErrorCode_Success {
		return nil, err, int32(errCode)
	}

	// 提交材料类型校验
	if objCfg.QuestObjectType == QUEST_OBJECT_TYPE_2 {
		if len(objCfg.ItemSubmitGroupNum) == 0 {
			return nil, fmt.Errorf("quest object cfg err"), int32(cmd.ErrorCode_ConfigError)
		}
		// 材料校验
		if len(req.Costs) == 0 {
			return nil, fmt.Errorf("param error"), int32(cmd.ErrorCode_ParamError)
		}
		costs := myUtils.ConvertItem(req.Costs)
		if !myUtils.CompareSameMap(costs, objCfg.ItemSubmitGroupNum) {
			return nil, fmt.Errorf("param error"), int32(cmd.ErrorCode_ParamError)
		}

		// 扣除提交的材料
		if !GetConsumeMgr(h.actor).CheckMapEnough(costs) {
			return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
		}
		err = GetConsumeMgr(h.actor).ConsumeList(costs, h.actor.comData, common.CR_Quest_Object_Submit)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 处理逻辑
	dropChange, newQuest, completeId, mappointEvents, err, errCode := h.tryCompleteQuest(req.ObjectId, h.actor.comData)
	if err != nil {
		return nil, err, int32(errCode)
	}

	// 消息返回
	h.actor.comData.GetQuestData().OpenQuests = append(h.actor.comData.GetQuestData().OpenQuests, newQuest...)
	h.actor.comData.GetQuestData().CompleteQuests = append(h.actor.comData.GetQuestData().CompleteQuests, completeId)

	rsp := &cmd.LS2C_CompleteQuestObjectRes{
		ObjectId:   req.ObjectId,
		CompleteId: completeId,
		CommonData: h.actor.comData.FixDownComData(),
		IncrEventInfo: &cmd.IncrMappointEventInfo{
			NiwaId:         h.actor.ChapterHandler.GetCurrNiwaId(),
			MappointEvents: mappointEvents,
		},
		DropChange: dropChange,
	}

	// 下发当前关卡天气
	levelData, ok := h.actor.ChapterHandler.GetCurrLevelData()
	if ok {
		rsp.BigLevelData = levelData.BigLevelData
	}

	return rsp, nil, 0
}

// 检查物件交互的合法性
func (h *QuestHandler) checkQuestCondition(openQuest *cmd.PCommonQuestInfo, objCfg *excel.QuestObjectCfg) (error, cmd.ErrorCode) {
	tempSteps := make(map[int32]int32)
	for _, step := range openQuest.CompleteSteps {
		tempSteps[step] = step
	}

	tempObjs := make(map[int32]int32)
	for _, obj := range openQuest.CompleteObject {
		tempObjs[obj] = obj
	}

	// 出现步骤
	_, ok := tempSteps[objCfg.AppearStep]
	if openQuest.StepId != objCfg.AppearStep && !ok {
		return fmt.Errorf("quest Step Not Match appear %d", objCfg.AppearStep), cmd.ErrorCode_QuestStepNotMatch
	}
	// 消失步骤
	if _, ok = tempSteps[objCfg.DisStep]; ok {
		return fmt.Errorf("quest Step Not Match dismiss %d", objCfg.DisStep), cmd.ErrorCode_QuestStepNotMatch
	}

	// 出现条件
	for _, v := range objCfg.AppearCondition {
		if _, ok = tempObjs[v]; !ok {
			return fmt.Errorf("quest Condition Not Match AppearCondition"), cmd.ErrorCode_QuestConditionNotMatch
		}
	}
	// 消失条件
	if len(objCfg.DisCondition) > 0 {
		b := true
		for _, v := range objCfg.DisCondition {
			if _, ok = tempObjs[v]; !ok {
				b = false
			}
		}
		if b {
			return fmt.Errorf("quest Condition Not Match DisCondition"), cmd.ErrorCode_QuestConditionNotMatch
		}
	}

	// 出现flag
	if len(objCfg.AppearFlag) > 0 && !h.actor.StoryFlagHandler.checkExistFlags(objCfg.AppearFlag...) {
		return fmt.Errorf("quest flag not match appearfalg %v", objCfg.AppearFlag), cmd.ErrorCode_QuestConditionNotMatch
	}
	// 消失flag
	if len(objCfg.DisFlag) > 0 && h.actor.StoryFlagHandler.checkExistFlags(objCfg.DisFlag...) {
		return fmt.Errorf("quest flag not match disflag %v", objCfg.DisFlag), cmd.ErrorCode_QuestConditionNotMatch
	}

	return nil, cmd.ErrorCode_Success
}

// 尝试完成任务,返回任务数据,如果解锁新任务，将返回老任务id
func (h *QuestHandler) tryCompleteQuest(objectId int32, commonData *clidto.Comdata) (*cmd.DropChange, []*cmd.PCommonQuestInfo, int32, []*cmd.MappointEvent, error, cmd.ErrorCode) {
	var (
		dropChange = &cmd.DropChange{}
		openQuest  *cmd.PCommonQuestInfo
		openQuests = make([]*cmd.PCommonQuestInfo, 0)
		err        error
	)
	h.Infof("tryCompleteQuest objId: %d", objectId)
	if objectId <= 0 {
		return dropChange, nil, 0, nil, nil, cmd.ErrorCode_Success
	}

	questData := h.actor.GetQuestData()

	objectCfg := excel.GetQuestObjectMgr().GetById(objectId)
	if objectCfg == nil {
		return dropChange, nil, 0, nil, fmt.Errorf("objectCfg not found %d", objectId), cmd.ErrorCode_QuestObjectConfigNotFound
	}

	openQuest = questData.OpenQuests[objectCfg.QuestId]
	if openQuest == nil {
		return dropChange, nil, 0, nil, fmt.Errorf("quest not open %d", objectCfg.QuestId), cmd.ErrorCode_QuestNotOpen
	}
	if openQuest.CompleteSteps == nil {
		openQuest.CompleteSteps = make([]int32, 0)
	}
	if openQuest.CompleteObject == nil {
		openQuest.CompleteObject = make([]int32, 0)
	}
	openQuests = append(openQuests, openQuest)
	// 重复完成容错
	for _, v := range openQuest.CompleteObject {
		if v == objectId {
			h.Infof("duplicate objectId %d", objectId)
			return dropChange, openQuests, 0, nil, nil, cmd.ErrorCode_Success
		}
	}

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CompleteQuestObject{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CompleteQuestObject, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ObjectId:       objectId,
	//		StepId:         openQuest.StepId,
	//		QuestId:        openQuest.QuestId,
	//		CompleteObject: lilith.ConvertList2Str(openQuest.CompleteObject),
	//		CompleteStep:   lilith.ConvertList2Str(openQuest.CompleteSteps),
	//		CompleteQuest:  lilith.ConvertList2Str(questData.CompleteQuests),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CompleteQuestObject{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ObjectId:          objectId,
			StepId:            openQuest.StepId,
			QuestId:           openQuest.QuestId,
			CompleteObject:    taptap.ConvertList2Str(openQuest.CompleteObject),
			CompleteStep:      taptap.ConvertList2Str(openQuest.CompleteSteps),
			CompleteQuest:     taptap.ConvertList2Str(questData.CompleteQuests),
		}
		taptap.WriteDataLog(taptap.LogType_CompleteQuestObject, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 更新天气编号
	if err = h.actor.ChapterHandler.updateBigLevelDataWeatherIdx(false, h.actor.ChapterHandler.GetCurrLevelId(), objectCfg.WeatherId); err != nil {
		h.Debugf(err.Error())
	}

	// 不完成步骤
	if objectCfg.FinishStepId == 0 {
		openQuest.CompleteObject = append(openQuest.CompleteObject, objectId)
		if err = h.SaveDB(); err != nil {
			return dropChange, nil, 0, nil, err, cmd.ErrorCode_SaveDBError
		}

		h.saveFlag(objectCfg.SetFlag, commonData)
		h.Infof("not finish step %d", objectId)
		return dropChange, openQuests, 0, nil, nil, cmd.ErrorCode_Success
	}

	stepCfg := excel.GetQuestStepMgr().GetById(objectCfg.FinishStepId)
	if stepCfg == nil {
		return dropChange, nil, 0, nil, fmt.Errorf("stepCfg not found %d", objectCfg.FinishStepId), cmd.ErrorCode_QuestStepConfigNotFound
	}

	if stepCfg.Isfinalstep == 1 {
		return h.subCompleteQuest(questData, stepCfg.QuestId, objectCfg.SetFlag, commonData)
	} else {
		// 任务未完成
		openQuest.CompleteObject = append(openQuest.CompleteObject, objectCfg.Id)
		if len(stepCfg.NextStep) == 0 {
			return h.subCompleteSonStep(questData, stepCfg, objectCfg.NextStepNum, objectCfg.SetFlag, commonData)
		} else {
			// 直接完成主步骤，尝试开启下一个
			h.subCompleteStep(openQuest, stepCfg, objectCfg.NextStepNum, objectCfg.SetFlag, commonData)
			if err = h.SaveDB(); err != nil {
				return dropChange, openQuests, 0, nil, err, cmd.ErrorCode_SaveDBError
			}
		}
		return dropChange, openQuests, 0, nil, nil, cmd.ErrorCode_Success
	}
}

// 完成任务
func (h *QuestHandler) subCompleteQuest(questData *cmd.PQuestData, questId int32, flags []string,
	commonData *clidto.Comdata) (*cmd.DropChange, []*cmd.PCommonQuestInfo, int32, []*cmd.MappointEvent, error, cmd.ErrorCode) {
	var (
		dropChange = &cmd.DropChange{}
	)

	// 任务完成
	questCfg := excel.GetQuestMgr().GetById(questId)
	if questCfg == nil {
		return dropChange, nil, 0, nil, fmt.Errorf("questCfg not found %d", questId), cmd.ErrorCode_QuestConfigNotFound
	}

	// 完成老任务
	questData.CompleteQuests = append(questData.CompleteQuests, questId)
	delete(questData.OpenQuests, questId)

	// 给奖励
	reward := datahelper.ConvertItem3(questCfg.GetQuestReward())
	_, err := GetDropMgr(h.actor).DropList2(reward, true, nil, commonData, common.CR_Quest_Reward)
	if err != nil {
		return dropChange, nil, 0, nil, err, cmd.ErrorCode_InternalError
	}

	// 尝试解锁新任务
	openQuests, err := h.tryCreateQuest(questData)
	if err != nil {
		return dropChange, nil, 0, nil, err, cmd.ErrorCode_InternalError
	}

	if err = h.SaveDB(); err != nil {
		return dropChange, nil, 0, nil, err, cmd.ErrorCode_SaveDBError
	}
	h.Infof("完成剧情任务 %d", questId)

	// 事件发布
	h.Debugf("成就类型%d: 埋点level_id: %d", int32(cmd.AchieveConditionType_Quest_1), h.actor.ChapterHandler.GetCurrLevelId())
	h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_QUEST_COMPLETE, []int32{TASK_TYPE_521}, map[string]interface{}{
		"condition": int32(cmd.AchieveConditionType_Quest_1),
		"quest_id":  int(questId),
		"level_id":  h.actor.ChapterHandler.GetCurrLevelId(),
	}))

	// 完成任务新增地图中的事件
	mappointEvents := h.actor.ChapterHandler.IncrCurrNiwaMappointEvents(common.MAPPOINT_EVENT_UPDATE_TYPE_2, []string{strconv.Itoa(int(questId))})

	//判断关系有没有可以解锁
	h.actor.UserRelationHandler.OpenRelationLevel(questId, commonData)

	h.saveFlag(flags, commonData)
	return dropChange, openQuests, questId, mappointEvents, nil, cmd.ErrorCode_Success
}

// 完成主步骤
func (h *QuestHandler) subCompleteStep(openQuest *cmd.PCommonQuestInfo, stepCfg *excel.QuestStepCfg, index int32, flags []string, commonData *clidto.Comdata) {
	openQuest.StepId = stepCfg.NextStep[index]
	openQuest.CompleteSteps = append(openQuest.CompleteSteps, stepCfg.Id)
	openQuest.Progress = 0
	h.saveFlag(flags, commonData)
	h.Infof("完成主步骤 %d 当前任务数据 %+v", stepCfg.Id, openQuest)
}

// 完成子步骤
func (h *QuestHandler) subCompleteSonStep(questData *cmd.PQuestData, stepCfg *excel.QuestStepCfg, index int32, flags []string,
	commonData *clidto.Comdata) (*cmd.DropChange, []*cmd.PCommonQuestInfo, int32, []*cmd.MappointEvent, error, cmd.ErrorCode) {
	var (
		dropChange = &cmd.DropChange{}
		openQuest  = questData.OpenQuests[stepCfg.QuestId]
		openQuests = []*cmd.PCommonQuestInfo{openQuest}
	)

	// 完成子步骤，判定主步骤是否完成
	curStepCfg := excel.GetQuestStepMgr().GetById(openQuest.StepId)
	if curStepCfg == nil {
		return dropChange, nil, 0, nil, fmt.Errorf("curStepCfg not found %d", openQuest.StepId), cmd.ErrorCode_QuestStepConfigNotFound
	}
	if len(curStepCfg.ChildStep) == 0 {
		return dropChange, nil, 0, nil, fmt.Errorf("childStep not found %d", openQuest.StepId), cmd.ErrorCode_QuestSonStepConfigNotFound
	}

	openQuest.CompleteSteps = append(openQuest.CompleteSteps, stepCfg.Id)

	// 临时构造map
	tempSteps := make(map[int32]int32)
	for _, v := range openQuest.CompleteSteps {
		tempSteps[v] = v
	}

	var sumStep int
	for _, v := range curStepCfg.ChildStep {
		if _, ok := tempSteps[v]; ok {
			sumStep++
		}
	}
	openQuest.Progress = int32(sumStep)

	// 完成主步骤
	if sumStep == len(curStepCfg.ChildStep) {
		if curStepCfg.Isfinalstep == 1 {
			// 判定是否最后步骤
			return h.subCompleteQuest(questData, curStepCfg.QuestId, flags, commonData)
		} else if len(curStepCfg.NextStep) != 0 {
			// 判定是否后置任务
			h.subCompleteStep(openQuest, curStepCfg, index, flags, commonData)
			return dropChange, openQuests, 0, nil, nil, cmd.ErrorCode_Success
		} else {
			return dropChange, nil, 0, nil, fmt.Errorf("quest step config error"), cmd.ErrorCode_ConfigError
		}
	}

	h.saveFlag(flags, commonData)

	if err := h.SaveDB(); err != nil {
		return dropChange, nil, 0, nil, err, cmd.ErrorCode_SaveDBError
	}
	h.Infof("完成子步骤 %d 当前任务数据 %+v", stepCfg.Id, openQuest)
	return dropChange, openQuests, 0, nil, nil, cmd.ErrorCode_Success
}

// GetCompleteQuestIds 获取已完成的任务id集合
func (h *QuestHandler) GetCompleteQuestIds() []int32 {
	return h.actor.GetQuestData().CompleteQuests
}

// IsComplete 判断任务有没有完成
func (h *QuestHandler) IsComplete(id int32) bool {
	return myUtils.ArrayContain(h.actor.GetQuestData().CompleteQuests, id)
}

func (h *QuestHandler) saveFlag(flags []string, commonData *clidto.Comdata) {
	if len(flags) == 0 {
		return
	}
	err, _ := h.actor.StoryFlagHandler.saveStoryFlag(commonData, flags...)
	if err != nil {
		h.Warn("saveFlag: ", err)
		return
	}
	h.Debug("saveFlag success: ", flags)
}

func (h *QuestHandler) createQuest(questId int32) (*cmd.PCommonQuestInfo, error, cmd.ErrorCode) {
	questCfg := excel.GetQuestMgr().GetById(questId)
	if questCfg == nil {
		return nil, fmt.Errorf("questCfg not found %d", questId), cmd.ErrorCode_QuestConfigNotFound
	}

	quest := &cmd.PCommonQuestInfo{
		QuestId:        questCfg.Id,
		StepId:         questCfg.FirstStep,
		CompleteSteps:  make([]int32, 0),
		CompleteObject: make([]int32, 0),
		Progress:       0,
	}
	h.Infof("创建新剧情 %d", questId)
	return quest, nil, cmd.ErrorCode_Success
}

// CheckMapEventComplete
//
//	@Description: 检查给定地图id上的事件是否全部完成
//	@receiver h
//	@param mapId 地图id
//	@return error 判定结果，全部完成返回nil
//	@return cmd.ErrorCode 错误码
//func (h *QuestHandler) CheckMapEventComplete(mapId int32) (error, cmd.ErrorCode) {
//	var (
//		err  error
//		code = cmd.ErrorCode_Success
//	)
//	data := h.actor.GetQuestData()
//
//	excel.GetQuestObjectMgr().Foreach(func(cfg *excel.QuestObjectCfg) bool {
//		// 所属地图
//		if cfg.NiwaId == mapId && cfg.EventId > 0 {
//			eventCfg := excel.GetMappointEventMgr().GetById(cfg.EventId)
//			if eventCfg == nil {
//				err = fmt.Errorf("eventCfg not found %d", cfg.EventId)
//				code = cmd.ErrorCode_NotFoundConfig
//				return false // return
//			}
//
//			// 是否核心事件
//			if eventCfg.IsmainEvent != 1 {
//				return true // continue
//			}
//
//			b := true
//			questInfo := data.OpenQuests[cfg.QuestId]
//			if questInfo != nil {
//				if questInfo.CompleteObject == nil {
//					questInfo.CompleteObject = make([]int32, 0)
//				}
//				// 是否出现
//				_, errorCode := checkQuestCondition(questInfo, cfg)
//				if errorCode != cmd.ErrorCode_Success {
//					return true
//				}
//				// 是否完成
//				for _, v := range questInfo.CompleteObject {
//					if v == cfg.Id {
//						b = false
//						break
//					}
//				}
//			} else {
//				// 已完成剧情的容错
//				for _, quest := range data.CompleteQuests {
//					if quest == cfg.QuestId {
//						h.Debugf("CheckMapEventComplete quest is complete %d", cfg.QuestId)
//						return true
//					}
//				}
//				h.Warnf("questInfo is nil %v", data.OpenQuests)
//			}
//
//			// 未完成
//			if b {
//				err = fmt.Errorf("物件 %d 未完成", cfg.Id)
//				code = cmd.ErrorCode_Chapter_quest_event_undone
//				return false
//			}
//		}
//
//		return true
//	}, false)
//
//	return err, code
//}

func (h *QuestHandler) directCompleteObjectByGM(objectId, f int32, commonData *clidto.Comdata) error {
	if f == 0 {
		// 完成指定的物件
		_, _, _, _, err, _ := h.tryCompleteQuest(objectId, commonData)
		if err != nil {
			return err
		}
	} else if f > 0 {
		// 完成当前物件之前的所有物件
		curObjCfg := excel.GetQuestObjectMgr().GetById(objectId)
		if curObjCfg == nil {
			return fmt.Errorf("object config not found")
		}
		curQuestCfg := excel.GetQuestMgr().GetById(curObjCfg.QuestId)
		if curQuestCfg == nil {
			return fmt.Errorf("quest config not found")
		}

		questData := h.actor.GetQuestData()

		// 使用同一条任务线
		for _, quest := range questData.OpenQuests {
			cfg := excel.GetQuestMgr().GetById(quest.QuestId)
			if cfg == nil {
				continue
			}
			if cfg.ChapterId != curQuestCfg.ChapterId || cfg.QuestType != curQuestCfg.QuestType {
				continue
			}
			nextQuest := cfg.Id
			for curObjCfg.QuestId != nextQuest {
				// 直接完成该任务
				err := h.directCompleteQuestByGM(nextQuest, commonData)
				if err != nil {
					return err
				}

				questCfg := excel.GetQuestMgr().GetById(nextQuest)
				if questCfg == nil {
					return fmt.Errorf("quest config not found")
				}
				nextQuest = questCfg.NextQuest
			}
		}

		// 拿到该任务下的所有的物件配置
		objCfgs := make([]*excel.QuestObjectCfg, 0)
		excel.GetQuestObjectMgr().Foreach(func(cfg *excel.QuestObjectCfg) bool {
			if cfg.QuestId == curObjCfg.QuestId {
				objCfgs = append(objCfgs, cfg)
			}
			return true
		}, true)

		// 遍历完成物件
		total := len(objCfgs)
		var b bool
		for i := 0; i < total; i++ {
			for j := 0; j < len(objCfgs); j++ {
				cfg := objCfgs[j]
				// 物件是否出现了
				if err, _ := h.CheckQuestCondition(cfg.Id); err != nil {
					continue
				}
				// 完成该物件
				_, _, _, _, err, _ := h.tryCompleteQuest(cfg.Id, commonData)
				if err != nil {
					return err
				}
				// 到达目标物件
				if cfg.Id == objectId {
					b = true
					break
				}
				// 清除物件
				objCfgs = append(objCfgs[:j], objCfgs[j+1:]...)
				j--
			}
			// 跳出
			if b {
				break
			}
		}
	}

	commonData.Data.Quest = h.buildQuestInfo()
	return nil
}

func (h *QuestHandler) directCompleteQuestByGM(questId int32, commonData *clidto.Comdata) error {
	questData := h.actor.GetQuestData()

	// 已经完成了
	for _, quest := range questData.CompleteQuests {
		if quest == questId {
			return nil
		}
	}

	_, newQuest, _, _, err, _ := h.subCompleteQuest(questData, questId, []string{}, commonData)
	if err != nil {
		return err
	}

	// 记录完成物件的flag
	excel.GetQuestObjectMgr().Foreach(func(cfg *excel.QuestObjectCfg) bool {
		if cfg.QuestId == questId && len(cfg.SetFlag) > 0 {
			h.saveFlag(cfg.SetFlag, commonData)
		}
		return true
	}, true)

	commonData.Data.Quest = h.buildQuestInfo()

	h.Debugf("完成任务 %d 获得新任务 %+v", questId, newQuest)
	return nil
}

// checkQuestFinish
//
//	@Description: 检查给定questId任务是否完成
//	@receiver h
//	@param questId 任务id
//	@return bool 完成返回true，否则返回false
func (h *QuestHandler) checkQuestFinish(questIds ...int32) bool {
	data := h.actor.GetQuestData()

	finishQuestIdMap := make(map[int32]bool)
	for _, id := range data.CompleteQuests {
		finishQuestIdMap[id] = true
	}

	for _, questId := range questIds {
		if _, ok := finishQuestIdMap[questId]; !ok {
			h.Debugf("发现物件还未完成, questId=%d", questId)
			return false
		}
	}

	return true
}

// 验证物件出现的合法性
func (h *QuestHandler) CheckQuestCondition(objId int32) (error, cmd.ErrorCode) {
	objCfg := excel.GetQuestObjectMgr().GetById(objId)
	if objCfg == nil {
		return fmt.Errorf("config not found %d", objId), cmd.ErrorCode_NotFoundConfig
	}

	// check
	data := h.actor.GetQuestData()
	openQuest := data.OpenQuests[objCfg.QuestId]
	if openQuest == nil {
		return fmt.Errorf("quest not open %d", objCfg.QuestId), cmd.ErrorCode_QuestNotOpen
	}

	return h.checkQuestCondition(openQuest, objCfg)
}

// GetLastCompleteQuestId 获取最后一个完成的剧情任务id
// 主线是单任务线,不会有多个最后完成id,这里只考虑主线
func (h *QuestHandler) GetLastCompleteQuestId() int32 {
	data := h.actor.GetQuestData()
	tempMap := make(map[int32]int32)
	for _, id := range data.CompleteQuests {
		tempMap[id] = id
	}
	var target int32
	for id := range tempMap {
		cfg := excel.GetQuestMgr().GetById(id)
		if cfg == nil || cfg.QuestType == 2 {
			continue
		}
		if _, ok := tempMap[cfg.NextQuest]; ok {
			continue
		}
		target = id
		break
	}

	h.Infof("GetLastCompleteQuestId %d", target)
	return target
}

// 拦路事件触发条件【专用】
func (h *QuestHandler) GetBlockTriggerId(triggers map[int32]int32) int32 {
	var (
		target      int32 // 目标questId
		minStep     int32 // 当前的最短路径值
		max         int   // 最大循环次数
		data        = h.actor.GetQuestData()
		completeMap = make(map[int32]int32)
	)
	excel.GetQuestMgr().Foreach(func(cfg *excel.QuestCfg) bool {
		max++
		return true
	}, true)

	// 构建临时的完成map
	for _, id := range data.CompleteQuests {
		completeMap[id] = id
	}

	// 找其中最后完成的一个任务
	for questId := range triggers {
		// 未完成的任务
		if _, ok := completeMap[questId]; !ok {
			continue
		}
		// 不存在或者支线
		cfg := excel.GetQuestMgr().GetById(questId)
		if cfg == nil || cfg.QuestType == 2 {
			continue
		}

		var tempStep int32
		for i := 0; i < max; i++ {
			// 没有下一步
			cfg = excel.GetQuestMgr().GetById(cfg.NextQuest)
			if cfg == nil {
				break
			}
			// 未完成的
			if _, ok := completeMap[cfg.Id]; !ok {
				break
			}
			tempStep++
		}
		// 初始值
		if target == 0 {
			target = questId
			minStep = tempStep
		} else {
			if tempStep < minStep {
				target = questId
				minStep = tempStep
			}
		}
	}
	h.Infof("GetBlockTriggerId triggers: %v, completes: %v, target: %v", triggers, completeMap, target)
	return target
}

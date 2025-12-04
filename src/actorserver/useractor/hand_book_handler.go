package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type HandBookHandler struct {
	*UABaseHandler
}

func NewHandBookHandler(actor *UserActor) *HandBookHandler {
	h := &HandBookHandler{UABaseHandler: NewUABaseHandler(actor, "HandBookHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_InitHandBookInfoReq), h.HandBookInfoReq) // 初始化
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_HandBookRewardReq), h.HandBookRewardReq) // 领取对应的任务奖励
	return h
}

func (h *HandBookHandler) EnterGame() error {
	return nil
}

func (h *HandBookHandler) DailyRefresh() error {
	return nil
}

func (h *HandBookHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PHandbookInfo); ok {
		h.actor.Data.Handbooks = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *HandBookHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserHandBook(h.actor.ID()), h.actor.Data.Handbooks
}

// Init 初始化模块数据
func (h *HandBookHandler) Init() error {
	// 初始化
	h.actor.Data.Handbooks = &cmd.PHandbookInfo{
		Createtime:   time.Now().Unix(),
		HandBookItem: nil,
	}
	handBookItem := make(map[uint32]*cmd.ServerHandBookItem, len(h.actor.GetUserCardData().Card))
	for _, value := range h.actor.GetUserCardData().Card {
		// 判定是否开启图鉴
		cardCfg := excel.GetBeastarMgr().GetById(int32(value.BaseId))
		if cardCfg == nil || cardCfg.BookSwitch == 0 {
			return nil
		}
		temp := &cmd.ServerHandBookItem{
			CardId:     value.BaseId,
			CreateTime: value.CreateTimestamp,
			TaskInfo:   make(map[int32]*cmd.TaskInfoItem, 0),
		}
		if _, ok := handBookItem[value.BaseId]; ok {
			continue
		}
		h.tryRefreshTasks(temp, 0)
		handBookItem[value.BaseId] = temp
		h.Debug("图鉴新增卡牌：", value.BaseId)
	}
	h.actor.Data.Handbooks.HandBookItem = handBookItem

	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init handbook data success. player: %s", h.actor.ID())
	return nil
}

//////////////////////////////////////////////////////协议相关

// HandBookInfoReq 图鉴列表初始化
func (h *HandBookHandler) HandBookInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	bookData := h.actor.GetHandBookData()
	handBookItem := make([]*cmd.HandBookItem, 0, len(bookData.HandBookItem))
	for _, v := range bookData.HandBookItem {
		handBookItem = append(handBookItem, toClientItem(v))
	}

	return &cmd.LS2C_InitHandBookInfoRes{Items: handBookItem}, nil, 0
}

// HandBookRewardReq 领取对应的任务奖励
func (h *HandBookHandler) HandBookRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_HandBookRewardReq
	err := proto.Unmarshal(in.Data, &req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	bookData := h.actor.GetHandBookData()

	// 卡牌是否解锁
	bookItem := bookData.HandBookItem[uint32(req.CardId)]
	if bookItem == nil {
		return nil, fmt.Errorf("hand book config not found %d", req.CardId), int32(cmd.ErrorCode_ParamError)
	}

	// 任务是否可领取
	taskInfoItem := bookItem.TaskInfo[req.TaskId]
	if taskInfoItem == nil {
		return nil, fmt.Errorf("hand book task config not found %d", req.TaskId), int32(cmd.ErrorCode_ParamError)
	}
	if taskInfoItem.Status != TASK_STATUS_COMPLETE {
		return nil, fmt.Errorf("hand book task config not found %d", req.TaskId), int32(cmd.ErrorCode_TaskStatusNotComplete)
	}

	// 奖励入库
	reward := excel.GetHandbookTaskMgr().GetById(req.TaskId)
	dropChange, err := GetDropMgr(h.actor).DropList2(map[int32]int32{reward.Reward.Key: reward.Reward.Val}, true, nil, h.actor.comData, common.CR_Handbook_Single_Reward)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	taskInfoItem.Status = TASK_STATUS_RECEIVED
	// 尝试刷新任务
	h.tryRefreshTasks(bookItem, req.GetTaskId())
	h.actor.comData.Data.HandBookItem = append(h.actor.comData.Data.HandBookItem, toClientItem(bookItem))

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	h.Infof("图鉴领取奖励,taskId[%d],cardId[%d]", req.GetTaskId(), req.GetCardId())
	return &cmd.LS2C_HandBookRewardRes{CommonData: h.actor.comData.FixDownComData(), TaskId: req.TaskId, DropChange: dropChange}, nil, 0
}

// 尝试处理任务类型
func (h *HandBookHandler) handleTaskType(e event.IEvent) error {
	iterm := make([]*cmd.HandBookItem, 0)
	for _, t := range e.Type() {
		for _, taskList := range h.actor.GetHandBookData().HandBookItem {
			tmpHandBook := &cmd.HandBookItem{
				CardId:     taskList.CardId,
				CreateTime: taskList.CreateTime,
				TaskInfo:   make([]*cmd.TaskInfoItem, 0),
			}
			// 判断有没有完成
			for _, value := range taskList.TaskInfo {
				if value.CondId == t {
					if h.actor.TaskTypeMgr.CheckTaskConditionComplete(value, e, true) {
						tmpHandBook.TaskInfo = append(tmpHandBook.TaskInfo, value)
					}
				}
			}
			// 如果有变化，就推给客户端
			if len(tmpHandBook.TaskInfo) > 0 {
				iterm = append(iterm, tmpHandBook)
			}
		}
	}
	h.actor.comData.Data.HandBookItem = append(h.actor.comData.Data.HandBookItem, iterm...)

	if err := h.SaveDB(); err != nil {
		h.Error(err)
		return nil
	}
	h.Debugf("handleTaskType duty %+v", e)
	return nil
}

// 新增卡牌数据
func (h *HandBookHandler) handleAddNewCard(e event.IEvent) error {
	cardId, ok := e.Get("cardId").(int32)
	if !ok {
		return nil
	}

	// 判定是否开启图鉴
	cardCfg := excel.GetBeastarMgr().GetById(cardId)
	if cardCfg == nil || cardCfg.BookSwitch == 0 {
		return nil
	}

	bookData := h.actor.GetHandBookData()
	_, ok = bookData.HandBookItem[uint32(cardId)]
	if ok {
		return nil
	}

	temp := &cmd.ServerHandBookItem{
		CardId:     uint32(cardId),
		CreateTime: time.Now().Unix(),
		TaskInfo:   make(map[int32]*cmd.TaskInfoItem, 0),
	}
	h.tryRefreshTasks(temp, 0)
	bookData.HandBookItem[temp.CardId] = temp
	if err := h.SaveDB(); err != nil {
		h.Error(err)
		return nil
	}
	h.Debug("图鉴新增卡牌：", temp.CardId)
	return nil
}

//////////////////////////////////////////////////////内部方法调用

// 构建初始化数据
func (h *HandBookHandler) buildHandInfo() []*cmd.HandBookItem {
	bookData := h.actor.GetHandBookData()

	handBookItem := make([]*cmd.HandBookItem, len(bookData.HandBookItem))
	for _, v := range bookData.HandBookItem {
		handBookItem = append(handBookItem, toClientItem(v))
	}

	return handBookItem
}

// 尝试接取任务数据
func (h *HandBookHandler) tryRefreshTasks(bookItem *cmd.ServerHandBookItem, taskId int32) {
	// 获取卡牌的品质
	rarity, err := GetCardRarityById(int32(bookItem.CardId))
	if err != nil {
		return
	}

	var tasks = make([]*excel.HandbookTaskCfg, 0)
	if len(bookItem.TaskInfo) == 0 {
		tasks = getConfigByRareAndPre(rarity, 0)
	}

	//接取新的任务
	if newTask := GetNewTask(bookItem, taskId, rarity); len(newTask) > 0 {
		tasks = append(tasks, newTask...)
	}

	// 构建任务数据
	for _, v := range tasks {
		task := h.actor.TaskTypeMgr.CreateTaskInfoItem(v.Id, v.Condition, v.ParamNum, []int32{int32(bookItem.CardId)}, true)
		if task != nil {
			bookItem.TaskInfo[task.Id] = task
		}
	}

	h.Debugf("refreshTasks uid:%s, data:%+v", h.actor.ID(), bookItem)
}
func GetNewTask(bookItem *cmd.ServerHandBookItem, taskId, rarity int32) []*excel.HandbookTaskCfg {
	temp := getConfigByRareAndPre(rarity, taskId)
	// 老任务删掉

	if len(temp) > 0 {
		delete(bookItem.TaskInfo, taskId)
	}
	return temp
}

func toClientItem(item *cmd.ServerHandBookItem) *cmd.HandBookItem {
	items := make([]*cmd.TaskInfoItem, 0, len(item.TaskInfo))
	for _, v := range item.TaskInfo {
		items = append(items, v)
	}
	return &cmd.HandBookItem{
		CardId:     item.CardId,
		CreateTime: item.CreateTime,
		TaskInfo:   items,
	}
}

// 根据品质和前置条件查找任务
func getConfigByRareAndPre(rarity int32, pre int32) []*excel.HandbookTaskCfg {
	cfgs := make([]*excel.HandbookTaskCfg, 0)
	excel.GetHandbookTaskMgr().Foreach(func(cfg *excel.HandbookTaskCfg) bool {
		if cfg.Rare != rarity {
			return true
		}
		if cfg.Pre != pre {
			return true
		}

		cfgs = append(cfgs, cfg)
		return true
	}, true)
	return cfgs
}

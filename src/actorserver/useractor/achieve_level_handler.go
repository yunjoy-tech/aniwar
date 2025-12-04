package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"strconv"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"google.golang.org/protobuf/proto"
)

type AchieveLevelHandler struct {
	*UABaseHandler
}

func NewAchieveLevelHandler(actor *UserActor) *AchieveLevelHandler {
	h := &AchieveLevelHandler{UABaseHandler: NewUABaseHandler(actor, "AchieveLevelHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_InitAchieveInfoReq), h.InitAchieveInfoReq)    //初始化数据
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_AchieveSectionRewardReq), h.SectionRewardReq) //领取成就档位奖励
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_AchieveGroupRewardReq), h.GroupRewardReq)     // 领取成就集奖励

	return h
}

// Init 初始化模块数据
func (h *AchieveLevelHandler) Init() error {
	return nil
}

func (h *AchieveLevelHandler) EnterGame() error {
	return nil
}

func (h *AchieveLevelHandler) DailyRefresh() error {
	return nil
}

func (h *AchieveLevelHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *AchieveLevelHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

////////////////////////////////////////////////////协议相关

// InitAchieveInfoReq 初始化成就
func (h *AchieveLevelHandler) InitAchieveInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	ret := &cmd.LS2C_InitAchieveInfoRes{Achieve: h.buildAchieveLevelData()}
	return ret, nil, 0
}

// SectionRewardReq 领取成就档位奖励
func (h *AchieveLevelHandler) SectionRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_AchieveSectionRewardReq
	err := proto.Unmarshal(in.Data, &req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	cfg := excel.GetAchievementDataMgr().GetById(req.Id)
	if cfg == nil {
		return nil, fmt.Errorf("config not found %d", req.Id), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// 是否完成
	if h.getAchieveNum(cfg) < cfg.AchievementNum {
		return nil, fmt.Errorf("achieve not complete %d", req.Id), int32(cmd.ErrorCode_TaskStatusNotComplete)
	}

	// 是否已领取
	key := buildSectionKey(cfg.AchievementsId, cfg.FinishCondition)
	if h.actor.AchieveHandler.CheckSectionReceived(key, cfg.Id) {
		return nil, fmt.Errorf("reward had received %s", key), int32(cmd.ErrorCode_TaskStatusNotComplete)
	}

	// 处理领取
	err = h.actor.AchieveHandler.LogSectionReceive(key, cfg.Id)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	_, err = GetDropMgr(h.actor).DropList2(cfg.AchievementReward, true, nil, h.actor.comData, common.CR_ACHIEVE_REWARD)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	achieveInfo := h.NewPLevelAchieveInfo(cfg.AchievementsId)
	achieveInfo.Items = append(achieveInfo.Items, h.buildItems(cfg))
	res := &cmd.LS2C_AchieveSectionRewardRes{
		CommonData:  h.actor.comData.FixDownComData(),
		AchieveInfo: achieveInfo,
	}
	h.Infof("成功领取成就[%d]奖励", req.GetId())

	return res, nil, 0
}

// GroupRewardReq 领取成就集奖励
func (h *AchieveLevelHandler) GroupRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_AchieveGroupRewardReq
	err := proto.Unmarshal(in.Data, &req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	cfg := excel.GetAchievementsMgr().GetById(req.GroupId)
	if cfg == nil {
		return nil, fmt.Errorf("config not found %d", req.GroupId), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// 是否达成
	if !h.checkGroupComplete(req.GroupId) {
		return nil, fmt.Errorf("achieve not complete %d", req.GroupId), int32(cmd.ErrorCode_TaskStatusNotComplete)
	}

	// 是否领取
	key := strconv.Itoa(int(req.GroupId))
	if h.actor.AchieveHandler.CheckGroupReceived(key) {
		return nil, fmt.Errorf("reward had received %v", req.GroupId), int32(cmd.ErrorCode_NoTaskRewardToGet)
	}

	// 处理领取
	err = h.actor.AchieveHandler.LogGroupReceive(key)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 掉落奖励
	rewards := datahelper.GetRewardsByDropId(cfg.CollectionReward)
	_, err = GetDropMgr(h.actor).DropListByItems(rewards, true, nil, h.actor.comData, common.CR_ACHIEVE_REWARD)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	res := &cmd.LS2C_AchieveGroupRewardRes{
		CommonData:  h.actor.comData.FixDownComData(),
		AchieveInfo: &cmd.PLevelAchieveInfo{GroupId: req.GroupId, Receive: true},
	}
	h.Infof("成功领取成就集[%d]奖励", req.GetGroupId())
	return res, nil, 0
}

// 成就计数处理逻辑
func (h *AchieveLevelHandler) handleConditionCheck(e event.IEvent) error {
	// 参数解析
	condition, ok := e.Get("condition").(int32)
	if !ok {
		return nil
	}

	// 取当前章节id
	levelId, ok := e.Get("level_id").(int32)
	if !ok {
		return nil
	}
	if levelId == 0 {
		h.Warn("get level_id is 0", condition)
		return nil
	}
	chapterCfg := excel.GetLevelMgr().GetById(levelId)
	if chapterCfg == nil {
		h.Warn("achievement handleConditionCheck get  chapterId err")
		return nil
	}

	achieve := &cmd.PLevelAchieveInfo{
		Receive: false,
		GroupId: chapterCfg.ChapterId,
		Items:   make([]*cmd.AchieveItem, 0),
	}
	var typ, questId int
	achievementCfg := make([]*excel.AchievementDataCfg, 0)
	switch cmd.AchieveConditionType(condition) {
	case cmd.AchieveConditionType_Quest_1:
		if questId, ok = e.Get("quest_id").(int); !ok {
			return nil
		}
		achievementCfg = h.GetAchieveCfg(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Quest_1), strconv.Itoa(questId))
		if len(achievementCfg) <= 0 {
			return nil
		}
		key := buildAchieveKey(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Quest_1), strconv.Itoa(questId))
		h.Infof("成就计数 key= %s", key)
		h.actor.AchieveHandler.AddAchieveNum(key, 1)

	case cmd.AchieveConditionType_Level_2:
		var pointIds []int32
		if pointIds, ok = e.Get("point_id").([]int32); !ok {
			return nil
		}
		for _, v := range pointIds {
			key := buildAchieveKey(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Level_2), strconv.Itoa(int(v)))
			h.Infof("成就计数 key= %s", key)
			achievementCfg = h.GetAchieveCfg(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Level_2), strconv.Itoa(int(v))) // 这里目前就一条数据，后面有多条数据再优化
			h.actor.AchieveHandler.AddAchieveNum(key, 1)
		}
	case cmd.AchieveConditionType_Box_3:
		achievementCfg = h.GetAchieveCfg(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Box_3), "0")
		if len(achievementCfg) <= 0 {
			return nil
		}
		key := buildAchieveKey(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Box_3), "0")
		h.Infof("成就计数 key= %s", key)
		h.actor.AchieveHandler.AddAchieveNum(key, 1)

	case cmd.AchieveConditionType_Collect_4:
		if typ, ok = e.Get("resource_id").(int); !ok {
			return nil
		}
		achievementCfg = h.GetAchieveCfg(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Collect_4), strconv.Itoa(typ))
		if len(achievementCfg) <= 0 {
			return nil
		}
		key := buildAchieveKey(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Collect_4), strconv.Itoa(typ))
		h.Infof("成就计数 key= %s", key)
		h.actor.AchieveHandler.AddAchieveNum(key, 1)

	case cmd.AchieveConditionType_Battle_5:
		if typ, ok = e.Get("type").(int); !ok {
			return nil
		}
		achievementCfg = h.GetAchieveCfg(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Battle_5), strconv.Itoa(typ))
		if len(achievementCfg) <= 0 {
			return nil
		}
		key := buildAchieveKey(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Battle_5), strconv.Itoa(typ))
		h.Infof("成就计数 key= %s", key)
		h.actor.AchieveHandler.AddAchieveNum(key, 1)

	case cmd.AchieveConditionType_Level_6:
		achievementCfg = h.GetAchieveCfg(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Level_6), strconv.Itoa(int(levelId)))
		if len(achievementCfg) == 0 {
			h.Warnf("achievement handleConditionCheck  AchieveConditionType_Level_6 get GetAchieveCfg nil:%d %d", int32(cmd.AchieveConditionType_Level_6), int(levelId))
			return nil
		}
		key := buildAchieveKey(chapterCfg.ChapterId, int32(cmd.AchieveConditionType_Level_6), strconv.Itoa(int(levelId)))

		if !h.actor.AchieveHandler.AddAchieveNumOnce(key, 1) {
			return nil
		}
		h.Infof("成就计数 key= %s", key)
		//判断有没有完成
	}
	//判断有没有完成
	h.FinishAchievementEvent()
	// 要过滤掉已经完成的
	for _, v := range achievementCfg {
		key := buildSectionKey(v.AchievementsId, v.FinishCondition)
		achieve.Items = append(achieve.Items, &cmd.AchieveItem{
			Id:      v.Id,
			Cur:     h.getAchieveNum(v),
			Receive: h.actor.AchieveHandler.CheckSectionReceived(key, v.Id),
		})
	}

	//判断有没有完成
	if len(achieve.Items) > 0 {
		h.actor.comData.Data.Achieve = append(h.actor.comData.Data.Achieve, achieve)
	}

	return nil
}

//////////////////////////////////////////////////////内部方法调用

func (h *AchieveLevelHandler) FinishAchievementEvent() {
	count := h.actor.AchieveHandler.GetCompleteCount()
	if count <= 0 {
		return
	}
	h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_ACHIEVE_COMPLETE, []int32{}, map[string]interface{}{
		"complete_num": count,
	}))
}
func (h *AchieveLevelHandler) GetAllAchievementCfg() []*excel.AchievementDataCfg {
	cfgs := make([]*excel.AchievementDataCfg, 0)
	excel.GetAchievementDataMgr().Foreach(func(cfg *excel.AchievementDataCfg) bool {
		cfgs = append(cfgs, cfg)
		return true
	}, true)

	return cfgs
}

// 构建大地图成就数据
func (h *AchieveLevelHandler) buildAchieveLevelData() []*cmd.PLevelAchieveInfo {
	ret := make([]*cmd.PLevelAchieveInfo, 0)
	// 按组进行数据构建
	excel.GetAchievementsMgr().Foreach(func(cfg *excel.AchievementsCfg) bool {
		// 取组数据
		cfgs := getCfgByGroupId(cfg.Id)
		items := make([]*cmd.AchieveItem, 0)
		for _, v := range cfgs {
			items = append(items, h.buildItems(v))
		}
		// 构建组数据
		ret = append(ret, &cmd.PLevelAchieveInfo{
			Receive: h.actor.AchieveHandler.CheckGroupReceived(strconv.Itoa(int(cfg.Id))),
			GroupId: cfg.Id,
			Items:   items,
		})
		return true
	}, true)

	return ret
}

func (h *AchieveLevelHandler) NewPLevelAchieveInfo(groupId int32) *cmd.PLevelAchieveInfo {
	key := strconv.Itoa(int(groupId))
	h.actor.AchieveHandler.CheckGroupReceived(key)
	achieveInfo := &cmd.PLevelAchieveInfo{
		GroupId: groupId,
		Receive: h.actor.AchieveHandler.CheckGroupReceived(key),
	}
	return achieveInfo
}

func (h *AchieveLevelHandler) buildItems(cfg *excel.AchievementDataCfg) *cmd.AchieveItem {
	cur := h.getAchieveNum(cfg)
	received := h.actor.AchieveHandler.CheckSectionReceived(buildSectionKey(cfg.AchievementsId, cfg.FinishCondition), cfg.Id)
	return &cmd.AchieveItem{
		Id:      cfg.Id,
		Cur:     cur,
		Receive: received,
	}
}

func (h *AchieveLevelHandler) getAchieveNum(cfg *excel.AchievementDataCfg) int32 {
	var cur int32
	params := cfg.FinishParam
	if len(cfg.FinishParam) == 0 {
		params = []string{"0"}
	}

	for _, param := range params {
		key := buildAchieveKey(cfg.AchievementsId, cfg.FinishCondition, param)
		cur += h.actor.AchieveHandler.GetAchieveNum(key)
	}
	return cur
}

// 检查整个成就组是否完成
func (h *AchieveLevelHandler) checkGroupComplete(groupId int32) bool {
	var ret = true
	excel.GetAchievementDataMgr().Foreach(func(cfg *excel.AchievementDataCfg) bool {
		if cfg.AchievementsId != groupId {
			return true
		}

		cur := h.getAchieveNum(cfg)
		if cur < cfg.AchievementNum {
			ret = false
			return false
		}

		return true
	}, false)
	return ret
}

func getCfgByGroupId(groupId int32) []*excel.AchievementDataCfg {
	cfgs := make([]*excel.AchievementDataCfg, 0)
	excel.GetAchievementDataMgr().Foreach(func(cfg *excel.AchievementDataCfg) bool {
		if cfg.AchievementsId == groupId {
			cfgs = append(cfgs, cfg)
		}
		return true
	}, true)
	return cfgs
}

package useractor

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"time"

	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
)

const (
	Call_Story_Type_One   = 1 //好感度特殊剧情
	Call_Story_Type_Two   = 2 //日常普通剧情
	Call_Story_Type_Three = 3 //每日特殊剧情
)

// ValidMissedCall 判断通话是否有效
func (h *CallSysHandler) ValidMissedCall(callId int64) *cmd.CallInfo {
	callSysData := h.actor.GetCallSysData()
	for _, call := range callSysData.MissedCalls {
		if call.CallId == callId {
			return call
		}
	}
	return nil
}

// ValidCall 判断通话是否有效
func (h *CallSysHandler) ValidCall(callId int64) *cmd.CallInfo {
	callSysData := h.actor.GetCallSysData()
	for _, call := range callSysData.CallInfo {
		if call.CallId == callId {
			return call
		}
	}
	return nil
}

// SetTempCallInfo 临时通话记录
func (h *CallSysHandler) SetTempCallInfo(info *cmd.CallInfo) {
	h.TempCallInfo = info
}

func (h *CallSysHandler) GetTempCallInfo() *cmd.CallInfo {
	return h.TempCallInfo
}

// AddMissedCalls 加入到未接电话记录里
func (h *CallSysHandler) AddMissedCalls(info *cmd.CallInfo) {
	callSysData := h.actor.GetCallSysData()
	for _, v := range callSysData.MissedCalls {
		if v.CallId == info.CallId {
			return
		}
	}
	callSysData.MissedCalls = append(callSysData.MissedCalls, info)
}

// DelMissedCalls 从未接来电里面删除
func (h *CallSysHandler) DelMissedCalls(info *cmd.CallInfo) {
	callSysData := h.actor.GetCallSysData()
	index := 0
	for i := 0; i < len(callSysData.MissedCalls); i++ {
		if callSysData.MissedCalls[i].CallId == info.CallId {
			index = i
			break
		}
	}
	callSysData.MissedCalls = append(callSysData.MissedCalls[:index], callSysData.MissedCalls[index+1:]...)
}

// AddCallInfo 添加电话
func (h *CallSysHandler) AddCallInfo(info *cmd.CallInfo) {
	callSysData := h.actor.GetCallSysData()
	for _, v := range callSysData.CallInfo {
		if v.CallId == info.CallId {
			return
		}
	}
	callSysData.CallInfo = append(callSysData.CallInfo, info)
}

// DelCallInfo 删除第一个
func (h *CallSysHandler) DelCallInfo() {
	callSysData := h.actor.GetCallSysData()
	if len(callSysData.CallInfo) == 0 {
		return
	}
	callSysData.CallInfo = callSysData.CallInfo[1:]
}

// AddCallRecord 添加 通话记录
func (h *CallSysHandler) AddCallRecord(record *cmd.CallRecord) {
	callSysData := h.actor.GetCallSysData()
	for _, v := range callSysData.Record {
		if v.CallId == record.CallId {
			return
		}
	}
	callSysData.Record = append(callSysData.Record, record)
}

// ChangeRecordState 改变记录的状态
func (h *CallSysHandler) ChangeRecordState(callId int64, state cmd.CallRecordState) {
	callSysData := h.actor.GetCallSysData()
	for i := 0; i < len(callSysData.Record); i++ {
		if callSysData.Record[i].CallId == callId {
			callSysData.Record[i].State = state
		}
	}
}

// AddCardFavoriteReward 添加好感度记录
func (h *CallSysHandler) AddCardFavoriteReward(cardId, favoriteId int32) {
	callSysData := h.actor.GetCallSysData()
	signal, ok := callSysData.Signal[cardId]
	if !ok {
		signal = &cmd.CardSignal{}
	}
	signal.FavoriteReward = append(signal.FavoriteReward, &cmd.FavoriteReward{
		FavoriteId: favoriteId,
		TimesStamp: time.Now().Unix(),
	})
}

func (h *CallSysHandler) ProcessExcel() {
	h.SpecialTimes = excel.GetConfigMgr().GetCfg().CALL_SPECIAL_LIMIT
	h.CommonTimes = excel.GetConfigMgr().GetCfg().CALL_COMMON_LIMIT

	if h.SignalWeightMap == nil {
		h.SignalWeightMap = make(map[interface{}]int32, 0)
	}
	if h.CallStory == nil {
		h.CallStory = make(map[int32]map[int32][]*excel.CallStoryCfg, 0)
	}
	//初始化信号权重
	excel.GetSignalMgr().Foreach(func(cfg *excel.SignalCfg) bool {
		h.SignalWeightMap[cfg.Id] = cfg.Rate
		return true
	}, true)
	// 初始化剧情
	excel.GetCallStoryMgr().Foreach(func(cfg *excel.CallStoryCfg) bool {
		var item map[int32][]*excel.CallStoryCfg
		var ok bool
		if item, ok = h.CallStory[cfg.Hero]; !ok {
			item = make(map[int32][]*excel.CallStoryCfg, 0)
		}
		item[cfg.Type] = append(item[cfg.Type], cfg)
		h.CallStory[cfg.Hero] = item
		return true
	}, true)

}

// GetSignalCfgInfo 获取信号配置信息
func (h *CallSysHandler) GetSignalCfgInfo(cardId int32) *excel.SignalCfg {
	callSysData := h.actor.GetCallSysData()
	if level, ok := callSysData.Signal[cardId]; ok {
		return excel.GetSignalMgr().GetById(level.GetSignalLevel())
	}
	return nil
}

// GetCallWaitTime 获取通话等待时间
func (h *CallSysHandler) GetCallWaitTime(cfg *excel.SignalCfg) int32 {
	waitCfg := cfg.PosIdx
	if waitCfg == nil {
		return 0
	}
	wait, _ := myUtils.RandomInt(waitCfg.Key, waitCfg.Val+1)
	return wait
}

// SpecialCall 特殊通话，一定能打通
func (h *CallSysHandler) SpecialCall(cardId int32, cfg *excel.SignalCfg) (*cmd.CallInfo, int32) {
	cardInfo, _ := h.actor.CardHandler.GetCard(uint32(cardId))
	if cardInfo == nil {
		return nil, int32(cmd.ErrorCode_CardNotExist)
	}
	callInfo := &cmd.CallInfo{
		CallId:         int64(h.actor.Srv.GenGUID(guid.GUID_BUILDING)),
		FavorId:        0,
		CallResultType: cmd.CallResultType_CallResultType_Through,
		WaitTime:       0,
		CardId:         cardId,
		CallType:       cmd.CallType_CallType_To_Card,
	}
	//获取等待时间
	waitTime := h.GetCallWaitTime(cfg)
	if waitTime == 0 {
		h.Debug("SpecialCall get waitTime is err")
		return nil, int32(cmd.ErrorCode_ConfigError)
	}
	callInfo.WaitTime = waitTime

	//获取剧情Id
	favorId := h.RandomStory(cardId, Call_Story_Type_Three, int32(cardInfo.FavoriteLevel))
	if favorId == 0 {
		return nil, int32(cmd.ErrorCode_ConfigError)
	}
	callInfo.FavorId = favorId
	h.Infof("CallSysHandler SpecialCall 特殊通话[%d]打通剧情Id[%d]:", cardId, favorId)
	return callInfo, int32(cmd.ErrorCode_Success)
}

func (h *CallSysHandler) CommonCall(cardId int32, cfg *excel.SignalCfg) (*cmd.CallInfo, int32) {

	cardInfo, _ := h.actor.CardHandler.GetCard(uint32(cardId))
	if cardInfo == nil {
		return nil, int32(cmd.ErrorCode_CardNotExist)
	}
	callInfo := &cmd.CallInfo{
		CallId:         int64(h.actor.Srv.GenGUID(guid.GUID_BUILDING)),
		CardId:         cardId,
		WaitTime:       3,
		CallResultType: cmd.CallResultType_CallResultType_None,
		CallType:       cmd.CallType_CallType_To_Card,
	}

	//根据好感度等级判断等都打通
	if favorCfg, thro := h.IsThroughByFavor(cardInfo); !thro { //打不通
		h.Debug("CallSysHandler CommonCall 此次电话根据好感度判断为打不通:", cardId)
		//判断是否是拒接
		callInfo.CallResultType = cmd.CallResultType_CallResultType_No_Answer //无人接听
		if h.RandomTarget(favorCfg.GetRefuse()) {
			callInfo.CallResultType = cmd.CallResultType_CallResultType_Reject
			h.Debug("CallSysHandler CommonCall 此次电话根据好感度判断为拒接:", cardId)
		} else {
			h.Debug("CallSysHandler CommonCall 此次电话根据好感度判断为无人接听:", cardId)
		}
		return callInfo, int32(cmd.ErrorCode_Success)
	}

	//恩据信号强弱判断能否打通
	if h.IsThroughBySignal(cfg) {
		//信号能打通
		callInfo.WaitTime = h.GetCallWaitTime(cfg)
		callInfo.CallResultType = cmd.CallResultType_CallResultType_Through
		//获取剧情Id
		favorId := h.RandomStory(cardId, Call_Story_Type_Two, int32(cardInfo.FavoriteLevel))
		if favorId == 0 {
			return nil, int32(cmd.ErrorCode_ConfigError)
		}
		callInfo.FavorId = favorId
		h.Infof("CallSysHandler CommonCall 此次给[%d]电话根据信号判断为打通[%d]:", cardId, favorId)
	} else {
		callInfo.CallResultType = cmd.CallResultType_CallResultType_No_Answer
		h.Info("CallSysHandler CommonCall 此次电话根据信号判断为打不通:", cardId)
	}
	return callInfo, int32(cmd.ErrorCode_Success)
}

func (h *CallSysHandler) OtherCall(cardId int32) (*cmd.CallInfo, int32) {
	return &cmd.CallInfo{
		CallId:         int64(h.actor.Srv.GenGUID(guid.GUID_BUILDING)),
		CallResultType: cmd.CallResultType_CallResultType_No_Answer,
		CallType:       cmd.CallType_CallType_To_Card,
		WaitTime:       3,
		CardId:         cardId,
	}, int32(cmd.ErrorCode_Success)
}

// IsThroughBySignal 信号强度能否打通
func (h *CallSysHandler) IsThroughBySignal(cfg *excel.SignalCfg) bool {
	//判断是否能打通
	return h.RandomTarget(cfg.GetConnectPro())
}

// IsThroughByFavor 好感度判断能否打通
func (h *CallSysHandler) IsThroughByFavor(cardInfo *cmd.CardData) (*excel.IntimacyCfg, bool) {
	// 获取好感度配置
	favorCfg := h.GetFavorCfg(int32(cardInfo.FavoriteLevel))
	if favorCfg == nil {
		h.Debug("CallSysHandler IsThroughByFavor get favorCfg is err", cardInfo.FavoriteLevel)
		return nil, false
	}
	//判断是否能打通
	return favorCfg, h.RandomTarget(favorCfg.GetConnectPro())

}

func (h *CallSysHandler) GetFavorCfg(level int32) *excel.IntimacyCfg {
	return excel.GetIntimacyMgr().GetById(level)
}

func (h *CallSysHandler) RandomTarget(target int32) bool {
	r, _ := myUtils.RandomInt(0, 100)
	if int32(r) <= target {
		return true
	}
	return false
}

func (h *CallSysHandler) GetSpecialTime() int32 {
	callSysData := h.actor.GetCallSysData()
	if callSysData.CallTimes >= h.SpecialTimes {
		return h.SpecialTimes
	}
	return callSysData.CallTimes
}

func (h *CallSysHandler) GetCardSignal() []*cmd.CardSignal {
	callSysData := h.actor.GetCallSysData()
	//判断信号强度是否需要刷新
	if callSysData.SignalTime <= time.Now().Unix() { //需要刷新
		h.FlushSignalLevel()
	}
	return h.SignalLevel2Client(callSysData.Signal)
}

func (h *CallSysHandler) GetRecords() []*cmd.CallRecord {
	callSysData := h.actor.GetCallSysData()
	return callSysData.Record
}

func (h *CallSysHandler) FlushCallTimes() {
	callSysData := h.actor.GetCallSysData()
	refreshTime := time.Unix(callSysData.CallTime, 0)
	if common.IsSameDayByOffset(refreshTime, time.Now(), common.GAME_DAILY_REFRESH_HOUR) {
		return
	}
	callSysData.CallTimes = 0
	callSysData.CallTime = h.GetCallTimesFlushTime()

	if err := h.SaveDB(); err != nil {
		return
	}
}

func (h *CallSysHandler) FlushSignalLevel() {
	callSysData := h.actor.GetCallSysData()
	//获取所有的card
	cardIds := h.actor.CardHandler.GetAllCardIds()
	for _, id := range cardIds {
		cardSignal, ok := callSysData.Signal[id]
		if ok {
			cardSignal.SignalLevel = h.RandomSignalLevel()
		} else {
			cardSignal = h.InitCardSignal()
		}
		callSysData.Signal[id] = cardSignal
	}
	callSysData.SignalTime = h.GetSignalFlushTime()
	if err := h.SaveDB(); err != nil {
		return
	}
}

// AddCardInitSignalLevel 新增卡片时候初始化信号
func (h *CallSysHandler) AddCardInitSignalLevel(cardId int32) {
	callSysData := h.actor.GetCallSysData()
	cardSignal := h.InitCardSignal()
	callSysData.Signal[cardId] = cardSignal
	if err := h.SaveDB(); err != nil {
		return
	}
}

func (h *CallSysHandler) InitCardSignal() *cmd.CardSignal {
	return &cmd.CardSignal{
		SignalLevel:    h.RandomSignalLevel(),
		FavoriteReward: make([]*cmd.FavoriteReward, 0),
	}
}

func (h *CallSysHandler) SignalLevel2Client(signal map[int32]*cmd.CardSignal) []*cmd.CardSignal {
	cardSignal := make([]*cmd.CardSignal, 0)

	for k, v := range signal {
		cardSignal = append(cardSignal, &cmd.CardSignal{
			CardId:         k,
			SignalLevel:    v.SignalLevel,
			FavoriteReward: v.FavoriteReward,
		})
	}
	return cardSignal
}

// RandomSignalLevel 随机一个信号
func (h *CallSysHandler) RandomSignalLevel() int32 {
	return myUtils.RandomMap(h.SignalWeightMap).(int32)
}

// FavorUpTriggerCall 好感度升级触发通话
func (h *CallSysHandler) FavorUpTriggerCall(cardId, level int32) {
	favorCfg := excel.GetFavorMgr().GetById(cardId*100 + level)
	if favorCfg == nil {
		return
	}
	if favorCfg.FavorStory == 0 {
		return
	}
	callInfo := &cmd.CallInfo{
		CallId:         int64(h.actor.Srv.GenGUID(guid.GUID_BUILDING)),
		FavorId:        favorCfg.Id,
		CardId:         cardId,
		WaitTime:       5,
		CallResultType: cmd.CallResultType_CallResultType_Through,
		CallType:       cmd.CallType_CallType_To_Player,
	}
	//存入到队列，等客户端请求返回给客户端
	h.AddCallInfo(callInfo)
	_ = h.SaveDB()
}

// RandomStory 随机剧情Id
func (h *CallSysHandler) RandomStory(cardId, ty, favorLevel int32) int32 {
	//根据卡片Id,类型获取配置
	tmpMap, ok := h.CallStory[cardId]
	if !ok {
		h.Debug("根据cardId 获取剧情失败:", cardId)
		return 0
	}
	cfgs, ok := tmpMap[ty]
	if !ok {
		h.Debug("根据Type 获取剧情失败:", ty)
		return 0
	}
	weightMap := make(map[interface{}]int32, 0) // 信号权重
	for _, v := range cfgs {
		if v.Limit <= favorLevel {
			weightMap[v.Id] = v.Parameter
		}
	}
	if len(weightMap) == 0 {
		h.Debug("卡片[cardId]好感度[%d],没有随到剧情", cardId, favorLevel)
		return 0
	}
	return myUtils.RandomMap(weightMap).(int32)
}

func (h *CallSysHandler) GetSignalFlushTime() int64 {
	hour := excel.GetConfigMgr().GetCfg().CALL_FRESH_LIMIT
	return time.Now().Unix() + int64(hour*60)
}

func (h *CallSysHandler) GetCallTimesFlushTime() int64 {
	return time.Now().Unix()
}

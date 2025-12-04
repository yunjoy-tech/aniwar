package useractor

import (
	"context"
	"errors"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"time"
)

// CallSysHandler 通话系统
type CallSysHandler struct {
	*UABaseHandler
	TempCallInfo    *cmd.CallInfo
	SpecialTimes    int32
	CommonTimes     int32
	SignalWeightMap map[interface{}]int32 // 信号权重
	CallStory       map[int32]map[int32][]*excel.CallStoryCfg
}

func NewCallSysHandler(actor *UserActor) *CallSysHandler {
	h := &CallSysHandler{UABaseHandler: NewUABaseHandler(actor, "CallSysHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_GetCallSysInfoReq), h.GetCallSysInfoReq) //获取通讯数据
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CallReq), h.CallReq)                     //给角色打电话
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CallAfterReq), h.CallAfterReq)           //接通电话
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CallHangUpReq), h.CallHangUpReq)         //挂断电话
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CallBackReq), h.CallBackReq)             //回拨电话

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CardFavoriteStoryRewardReq), h.FavoriteStoryRewardReq) // 领取好感度剧情奖励
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CardFavoriteItemReq), h.FavoriteItemReq)               // 使用好感度经验道具
	return h
}

// Init 初始化模块数据
func (h *CallSysHandler) Init() error {
	// 初始化
	h.actor.Data.CallSysData = &cmd.PUserCallSysData{
		CallTimes:  0,
		Signal:     make(map[int32]*cmd.CardSignal, 0),
		SignalTime: 0,
		Record:     make([]*cmd.CallRecord, 0),
		CallInfo:   make([]*cmd.CallInfo, 0),
		CallTime:   h.GetCallTimesFlushTime(),
	}

	//h.FlushSignalLevel()
	//h.actor.Data.CallSysData.SignalTime = h.GetSignalFlushTime()
	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init achieve data success. player: %s", h.actor.ID())
	return nil
}

func (h *CallSysHandler) EnterGame() error {
	// 预处理配置表
	h.ProcessExcel()
	return nil
}

func (h *CallSysHandler) DailyRefresh() error {
	return nil
}

func (h *CallSysHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUserCallSysData); ok {
		h.actor.Data.CallSysData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *CallSysHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCallSys(h.actor.ID()), h.actor.Data.CallSysData
}

// GetCallSysInfoReq 获取通讯器信息
func (h *CallSysHandler) GetCallSysInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &cmd.C2LS_GetCallSysInfoReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	//判断每日次数要不要刷新
	h.FlushCallTimes()

	//返回数据
	res := &cmd.LS2C_GetCallSysInfoRes{
		CardSignal: h.GetCardSignal(),
		CallTimes:  h.GetSpecialTime(),
		Records:    h.GetRecords(),
	}

	return res, nil, int32(cmd.ErrorCode_Success)
}

// CallReq 给角色打电话
func (h *CallSysHandler) CallReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, codex := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(codex)
	}
	req := &cmd.C2LS_CallReq{}
	if err = in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	reqCardId := req.GetCardId()
	var callInfo *cmd.CallInfo
	var code int32
	// 获取卡片打给玩家的电话
	res := &cmd.LS2C_CallRes{}
	if reqCardId == 0 {
		callInfo = h.GetCallInfo()
		res.CallInfo = callInfo
		return res, nil, int32(cmd.ErrorCode_Success)
	}
	// 玩家给卡片打电话
	if reqCardId > 0 {
		if callInfo, code = h.UserCallCard(reqCardId); code != int32(cmd.ErrorCode_Success) {
			return nil, errors.New("UserCallCard is err"), code
		}
	}
	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	res.CallInfo = callInfo
	return res, nil, int32(cmd.ErrorCode_Success)
}

// CallAfterReq 接通电话
func (h *CallSysHandler) CallAfterReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &cmd.C2LS_CallAfterReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	// 玩家打给卡片
	if req.CallType == cmd.CallType_CallType_To_Card {
		if err, code := h.CallAfter2Card(); code != int32(cmd.ErrorCode_Success) {
			return nil, err, code
		}
	}
	// 卡片打给玩家
	if req.CallType == cmd.CallType_CallType_To_Player {
		if code := h.CallAfter2Player(req.GetCallId()); code != int32(cmd.ErrorCode_Success) {
			return nil, errors.New("invalid call"), code
		}
	}

	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	res := &cmd.LS2C_CallAfterRes{CommonData: h.actor.comData.FixDownComData()}
	return res, nil, int32(cmd.ErrorCode_Success)
}

// CallHangUpReq 挂断电话
func (h *CallSysHandler) CallHangUpReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &cmd.C2LS_CallHangUpReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	//  判断通话是否有效
	var callInfo *cmd.CallInfo
	if callInfo = h.ValidCall(req.CallId); callInfo == nil {
		return nil, errors.New("invalid callId "), int32(cmd.ErrorCode_ParamError)
	}

	//把通话加入到未接电话记录
	h.AddMissedCalls(callInfo)

	//加入到通话记录里
	record := &cmd.CallRecord{
		CardId:   callInfo.CardId,
		State:    cmd.CallRecordState_CallRecordState_No_Answer,
		CallTime: time.Now().Unix(),
		CallId:   callInfo.CallId,
	}
	// 添加到通话记录里
	h.AddCallRecord(record)
	//从队列中删除
	h.DelCallInfo()
	if err := h.SaveDB(); err != nil {
		h.Debug("CallHangUpReq SaveDB is err:", err)
		return nil, errors.New("SaveDB is err"), int32(cmd.ErrorCode_InternalError)
	}

	return &cmd.LS2C_CallHangUpRes{}, nil, int32(cmd.ErrorCode_Success)
}

// CallBackReq 回拨
func (h *CallSysHandler) CallBackReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &cmd.C2LS_CallBackReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	//  判断通话是否有效
	var callInfo *cmd.CallInfo
	if callInfo = h.ValidMissedCall(req.CallId); callInfo == nil {
		return nil, errors.New("invalid callId "), int32(cmd.ErrorCode_ParamError)
	}

	//// 从未接电话里面删除
	//h.DelMissedCalls(callInfo)
	//
	////改变通话记录的状态
	//h.ChangeRecordState(req.CallId, cmd.CallRecordState_CallRecordState_Answer)

	if err := h.SaveDB(); err != nil {
		h.Debug("CallBackReq SaveDB is err:", err)
		return nil, errors.New("SaveDB is err"), int32(cmd.ErrorCode_InternalError)
	}

	res := &cmd.LS2C_CallBackRes{
		CallInfo: callInfo,
	}
	return res, nil, int32(cmd.ErrorCode_Success)
}

// UserCallCard 玩家给卡片打电话
func (h *CallSysHandler) UserCallCard(reqCardId int32) (*cmd.CallInfo, int32) {
	var callInfo *cmd.CallInfo
	// 获得卡片的信号信息
	signalCfg := h.GetSignalCfgInfo(reqCardId)

	callSysData := h.actor.GetCallSysData()
	if signalCfg == nil {
		h.Debug("CallReq get signalCfg is err:", reqCardId)
		return nil, int32(cmd.ErrorCode_ConfigError)
	}

	var code int32
	//特殊通话必定打通
	if callSysData.CallTimes < h.SpecialTimes {
		callInfo, code = h.SpecialCall(reqCardId, signalCfg)
		h.Debug("CallSysHandler 每日特殊通话：", callSysData.CallTimes)
	} else if h.SpecialTimes <= callSysData.CallTimes && callSysData.CallTimes < h.SpecialTimes+h.CommonTimes { // 大于3次,小于日常8次 ，判断概率
		callInfo, code = h.CommonCall(reqCardId, signalCfg)
		h.Debug("CallSysHandler 日常通话：", callSysData.CallTimes)
	} else { // 大于8次 必定打不通
		callInfo, code = h.OtherCall(reqCardId)
		h.Debug("CallSysHandler 超过日常通话：", callSysData.CallTimes)
	}
	if code != int32(cmd.ErrorCode_Success) {
		return nil, code
	}
	//设置临时通话
	h.SetTempCallInfo(callInfo)

	return callInfo, int32(cmd.ErrorCode_Success)
}

// GetCallInfo 获取卡片打给玩家的通话
func (h *CallSysHandler) GetCallInfo() *cmd.CallInfo {
	callSysData := h.actor.GetCallSysData()
	var callInfo *cmd.CallInfo
	if len(callSysData.CallInfo) > 0 {
		callInfo = callSysData.CallInfo[0]
	}
	return callInfo
}

// CallAfter2Player 卡片打给玩家
func (h *CallSysHandler) CallAfter2Player(callId int64) int32 {

	var callInfo *cmd.CallInfo
	if callId == 0 { //直接接通
		callSysData := h.actor.GetCallSysData()
		if len(callSysData.CallInfo) == 0 {
			return int32(cmd.ErrorCode_Success)

		}
		callInfo = callSysData.CallInfo[0]
		// 从队列中删除通话
		h.DelCallInfo()
	} else { // 回拨后打通
		if callInfo = h.ValidMissedCall(callId); callInfo == nil {
			return int32(cmd.ErrorCode_ParamError)
		}
		// 从未接电话里面删除
		h.DelMissedCalls(callInfo)
		//改变通话记录的状态
		h.ChangeRecordState(callId, cmd.CallRecordState_CallRecordState_Answer)
	}
	//添加好感度记录
	h.AddCardFavoriteReward(callInfo.CardId, callInfo.FavorId)
	return int32(cmd.ErrorCode_Success)
}

// CallAfter2Card 玩家打给卡片
func (h *CallSysHandler) CallAfter2Card() (error, int32) {
	callSysData := h.actor.GetCallSysData()
	//判断上次通话记录
	tempCallInfo := h.GetTempCallInfo()
	if tempCallInfo == nil {
		h.Debug("CallSysHandler CallAfter2Card invalid call")
		return errors.New(" CallAfter2Card invalid call"), int32(cmd.ErrorCode_Call_Sys_invalid_call)
	}

	//增加打电话的次数
	callSysData.CallTimes++
	if callSysData.CallTimes <= h.SpecialTimes { //特殊通话,增加好感度经验
		var cardInfo *cmd.CardData
		cardInfo, _ = h.actor.CardHandler.GetCard(uint32(tempCallInfo.CardId))
		h.actor.CardHandler.AddFavoriteExp(cardInfo, uint32(excel.GetConfigMgr().GetCfg().CALL_SPECIAL_REWARD))
		h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.actor.CardHandler.ToClientData(cardInfo))
	}
	//电话加入到通话记录里
	record := &cmd.CallRecord{
		CardId:   tempCallInfo.CardId,
		State:    cmd.CallRecordState_CallRecordState_Answer,
		CallTime: time.Now().Unix(),
		CallId:   tempCallInfo.CallId,
	}
	h.AddCallRecord(record)

	return nil, int32(cmd.ErrorCode_Success)
}

func (h *CallSysHandler) FavoriteStoryRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &cmd.C2LS_CardFavoriteStoryRewardReq{}
	if err = in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 参数校验
	card, err := h.actor.CardHandler.GetCard(uint32(req.CardId))
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_CardNotExist)
	}
	for _, v := range card.FavoriteReward {
		if v == req.Section {
			return nil, fmt.Errorf("section reward had receive %d", v), int32(cmd.ErrorCode_ParamError)
		}
	}

	// 是否可以领取
	favorCfg := excel.GetFavorMgr().GetById(req.CardId*100 + req.Section)
	if favorCfg == nil {
		return nil, fmt.Errorf("cfg not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}
	if len(favorCfg.StoryReward) == 0 {
		return nil, fmt.Errorf("cfg not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	_, err = GetDropMgr(h.actor).DropList2(favorCfg.StoryReward, true, nil, h.actor.comData, common.CR_FAVOR_REWARD)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	card.FavoriteReward = append(card.FavoriteReward, req.Section)
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.actor.CardHandler.ToClientData(card))
	return &cmd.LS2C_CardFavoriteStoryRewardRes{CommonData: h.actor.comData.FixDownComData()}, nil, int32(cmd.ErrorCode_Success)
}

func (h *CallSysHandler) FavoriteItemReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CALL)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &cmd.C2LS_CardFavoriteItemReq{}
	if err = in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 道具校验
	costs := utils.ConvertItem(req.Items)
	if !GetConsumeMgr(h.actor).CheckMapItemNumAndType(costs, int32(cmd.ItemType_Material), int32(cmd.ItemMaterialType_ItemMaterialType_Gift)) { //10
		return nil, fmt.Errorf("param error"), int32(cmd.ErrorCode_ParamError)
	}

	// 取对应卡牌数据
	card, err := h.actor.CardHandler.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_CardNotExist)
	}

	// 满级了
	maxLv := getMaxLvTempalte(6)
	if card.FavoriteLevel >= maxLv {
		return nil, fmt.Errorf("favor level is max"), int32(cmd.ErrorCode_InvalidParam)
	}

	if err = GetConsumeMgr(h.actor).ConsumeList(costs, h.actor.comData, common.CR_FAVOR_ITEM_USE); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 计算经验值
	sum := calFavorExp(int32(req.CardId), costs)
	errorCode := h.actor.CardHandler.AddFavoriteExp(card, sum)
	if errorCode != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("add favor exp failed"), int32(errorCode)
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.actor.CardHandler.ToClientData(card))
	return &cmd.LS2C_CardFavoriteItemRes{CommonData: h.actor.comData.FixDownComData(), Items: req.Items}, nil, 0
}

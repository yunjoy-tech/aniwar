package useractor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/sensitive"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"gitee.com/aniwar2/musae/utils"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/service"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type UserChatHandler struct {
	*UABaseHandler
	FriendMessageLimit int32
	wChan              chan interface{} // 输出管道
}

type ChatMessage struct {
	FromRoleId uint64
	ToRoleId   uint64
	Message    *pb.BroadMessage
}

func NewChatMessage(fromRoleId, toRoleId uint64, message *pb.BroadMessage) *ChatMessage {
	return &ChatMessage{
		FromRoleId: fromRoleId,
		ToRoleId:   toRoleId,
		Message:    message,
	}
}

func NewUserChatHandler(actor *UserActor) *UserChatHandler {
	h := &UserChatHandler{UABaseHandler: NewUABaseHandler(actor, "UserChatHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_GetChatMessageReq), h.GetChatMessageReq)   // 获取聊天消息
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_SendChatMessageReq), h.SendChatMessageReq) // 发送聊天消息
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_HasReadMessageReq), h.HasReadMessageReq)   // 标记好友消息已读
	return h
}

// Init 初始化模块数据
func (h *UserChatHandler) Init() error {
	// 初始化
	h.actor.Data.ChatInfo = &pb.PUserChatInfo{
		LastSendTime: make(map[string]int64, 0),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init UserChat data success. player: %s", h.actor.ID())
	return nil
}

func (h *UserChatHandler) EnterGame() error {
	// 聊天消息异步存ES TODO 废弃es方案，避免引入es组件 方案一：redis缓存部分消息 方案二: 接入专业聊天sdk
	h.ProcessMessage()
	return nil
}

func (h *UserChatHandler) DailyRefresh() error {
	return nil
}

func (h *UserChatHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PUserChatInfo); ok {
		h.actor.Data.ChatInfo = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}
	return nil
}

func (h *UserChatHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserChatInfo(h.actor.ID()), h.actor.Data.ChatInfo
}

func (h *UserChatHandler) ProcessMessage() {

	h.wChan = make(chan interface{}, 1024)
	utils.GoSafeRun(func() {
		for {
			select {
			case tmpMessage, _ := <-h.wChan:
				if message, ok := tmpMessage.(*ChatMessage); ok {
					if err := h.SaveMessage2ES(message.FromRoleId, message.ToRoleId, message.Message); err != nil {
						h.Debug("聊天信息存储ES失败", err)
					}
				} else {
					h.Debugf("聊天数据类型输错")
				}
			}
		}
	}, nil)
}

func (h *UserChatHandler) MessageWrite2Channel(fromRoleId, toRoleId uint64, message *pb.BroadMessage) {
	chatMessage := NewChatMessage(fromRoleId, toRoleId, message)
	h.wChan <- chatMessage
	h.Debug("chat write message to channel:", fromRoleId, toRoleId, message)
}

// GetChatMessageReq 获取私人聊天消息
func (h *UserChatHandler) GetChatMessageReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	roleId := in.RoleId
	req := &pb.C2LS_GetChatMessageReq{}

	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	// 判断是否是好友或者同一联盟
	if code := h.IsFriendOrUnion(req.GetTarget(), req.GetChannelId()); code != int32(pb.ErrorCode_Success) {
		return nil, errors.New("不是好友或同一联盟"), code
	}
	res := &pb.LS2C_GetChatMessageRes{
		ChannelId: req.GetChannelId(),
	}
	message := make([]*pb.BroadMessage, 0)
	if req.GetChannelId() == pb.ChatChannel_Channel_private {
		// 设置没有未读消息
		h.SetHasMessage(roleId, false)
		// 从DB中获取数据
		message = h.GetMessageFromES(roleId, req.GetTarget(), time.Now().Unix(), req.GetFromSize(), req.GetSize())
	}

	// 获取联盟消息
	if req.GetChannelId() == pb.ChatChannel_Channel_alliance {
		reqMsg := &pb.S2S_GetAllianceMessageReq{
			FromSize: req.GetFromSize(),
			Size:     req.GetSize(),
			RoleId:   int64(in.RoleId),
		}
		rspData := &pb.S2S_GetAllianceMessageRes{}
		err, code := h.actor.UserAllianceHandler.AllianceInvoke(int64(req.GetTarget()), int32(pb.Protocols_PS2S_GetAllianceMessageReq), reqMsg, rspData, in.GetTopic())
		if err != nil {
			return nil, err, int32(code)
		}
		message = rspData.GetMessage()
	}

	res.Message = message

	return res, nil, int32(pb.ErrorCode_Success)
}

// SendChatMessageReq 好友、联盟成员发送消息
func (h *UserChatHandler) SendChatMessageReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	roleId := in.RoleId
	req := &pb.C2LS_SendChatMessageReq{}

	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, errors.New("发送空消息"), int32(pb.ErrorCode_Chat_message_empty)
	}
	// 暂时去掉判断CD时间
	if !h.IsCD(req.GetToObject(), req.GetChannelId()) {
		return nil, errors.New("CD 时间"), int32(pb.ErrorCode_Chat_message_CD)
	}

	// 判断长度
	if int32(utf8.RuneCountInString(req.GetMessage())) > h.FriendMessageLimit {
		return nil, errors.New("message too long"), int32(pb.ErrorCode_Chat_message_too_long)
	}
	// 敏感词过滤
	result, err := sensitive.CheckSensitiveWord(common.CHECK_TYPE_PLAYERNAME, req.GetMessage())
	if !result {
		return nil, err, int32(pb.ErrorCode_Chat_illegal_message)
	}
	// 判断是否是好友或者书联盟成员
	if code := h.IsFriendOrUnion(req.GetToObject(), req.GetChannelId()); code != int32(pb.ErrorCode_Success) {
		return nil, errors.New("不是好友或同一联盟"), code
	}
	if err := h.SaveDB(); err != nil {
		h.Debug("chat saveDB failed", err)
		return nil, errors.New("保存聊天消息失败"), int32(pb.ErrorCode_InternalError)
	}
	now := time.Now().Unix()
	message := &pb.BroadMessage{
		FromRoleId: roleId,
		Data:       []string{req.GetMessage()},
		TimeStamp:  now,
	}
	// 私人消息
	if req.GetChannelId() == pb.ChatChannel_Channel_private {
		message.MType = pb.MessageType_Message_Type_private
		h.Debugf("在[%d]时间,[%d]给[%d]发了消息：[%s]", now, roleId, req.GetToObject(), req.GetMessage())
		code := h.PushMessageToFriend(roleId, req.ToObject, message, pb.ChatChannel_Channel_private, true)
		if code != int32(pb.ErrorCode_Success) {
			return nil, errors.New("聊天消息推送好友失败"), code
		}
	}
	// 联盟消息
	if req.GetChannelId() == pb.ChatChannel_Channel_alliance {
		message.MType = pb.MessageType_Message_Type_alliance
		h.Debugf("在[%d]时间,[%d]向联盟[%d]发了消息：[%s]", now, roleId, req.GetToObject(), req.GetMessage())
		code := h.PushMessage2Alliance(int64(req.ToObject), message, pb.ChatChannel_Channel_alliance, in.GetTopic())
		if code != int32(pb.ErrorCode_Success) {
			return nil, errors.New("聊天消息推送好友失败"), code
		}
	}

	res := &pb.LS2C_SendChatMessageRes{
		Message: message,
	}
	return res, nil, int32(pb.ErrorCode_Success)
}

func (h *UserChatHandler) HasReadMessageReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &pb.C2LS_HasReadMessageReq{}

	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	h.SetHasMessage(req.GetFriendRoleId(), false)

	info, err := h.actor.getRoleBaseDataByRoleId(req.GetFriendRoleId())
	if err != nil {
		h.Debug("chat HasReadMessageReq getRoleBaseDataByRoleId err:", err)
		return nil, errors.New("获取好友信息失败"), int32(pb.ErrorCode_InternalError)
	}
	if h.actor.comData.Data.Friends == nil {
		h.actor.comData.Data.Friends = &pb.PClientFriendInfo{
			Friends: make([]*pb.PCommonRoleBaseInfo, 0),
		}
	}
	h.actor.comData.Data.Friends.Friends = append(h.actor.comData.Data.Friends.Friends, info.Common)
	res := &pb.LS2C_HasReadMessageRes{
		CommonData: h.actor.comData.FixDownComData(),
	}

	return res, nil, int32(pb.ErrorCode_Success)
}

// PushMessageToUserReq 处理好友推过来的聊天消息
func (h *UserChatHandler) PushMessageToUserReq(req *pb.S2S_PushMessageToUserReq, channel pb.ChatChannel) (error, int32) {
	h.Infof("处理好友发过来的消息:%v", req)
	// 判断是否是好友或同一联盟
	if code := h.IsFriendOrUnion(req.GetFromRoleId(), channel); code != int32(pb.ErrorCode_Success) {
		return errors.New("不是好友或同一联盟"), code
	}
	// 设置好友有未读消息
	h.SetHasMessage(req.GetFromRoleId(), true)
	if err := h.SaveDB(); err != nil {
		h.Debug("chat saveDB failed", err)
		return errors.New("保存聊天消息失败"), int32(pb.ErrorCode_InternalError)
	}
	h.Debug("chat --- 处理好友发送的消息:", req.GetToRoleId(), req.Message)
	// 通知自己客户端
	h.NotifyPrivateMessage(req.GetFromRoleId(), req.GetToRoleId(), req.Message, req.GetChannelId())
	return nil, int32(pb.ErrorCode_Success)
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////内部调用

func (h *UserChatHandler) PushMessage2Alliance(allianceId int64, message *pb.BroadMessage, channelId pb.ChatChannel, topic string) int32 {
	req := &pb.S2S_SendMessage2AllianceReq{
		Message: message,
	}
	res := &pb.S2S_SendMessage2AllianceRes{}
	// 获取玩家的联盟Id
	err, _ := h.actor.UserAllianceHandler.AllianceInvoke(int64(allianceId), int32(pb.Protocols_PS2S_SendMessage2AllianceReq), req, res, topic)
	if err != nil {
		h.Debugf("玩家[%d],向allianceActor,转发消息失败:[%v]", allianceId, err)
	}
	return 0
}

func (h *UserChatHandler) PushMessageToFriend(fromRoleId, toRoleId uint64, message *pb.BroadMessage, channelId pb.ChatChannel, save bool) int32 {

	// 存储es
	if save {
		// if err := h.SaveMessage2ES(fromRoleId, toRoleId, message); err != nil {
		//	h.Debug("聊天信息存储ES失败", err)
		//	return int32(pb.ErrorCode_InternalError)
		// }
		h.MessageWrite2Channel(fromRoleId, toRoleId, message)
	}
	// 推送给好友
	call, _ := proto.Marshal(&pb.S2S_PushMessageToUserReq{
		FromRoleId: h.actor.roleId,
		Message:    message,
		ToRoleId:   toRoleId,
		ChannelId:  channelId,
	})

	if err := h.PushMessage2Friend(toRoleId, int32(pb.Protocols_PS2S_PushMessageToUserReq), call); err != nil {
		return int32(pb.ErrorCode_Chat_message_push_friend_failed)
	}
	return int32(pb.ErrorCode_Success)
}

// IsCD 是否是CD 时间内 如果是联盟聊天，参数roleId  就是联盟Id
func (h *UserChatHandler) IsCD(roleId uint64, channel pb.ChatChannel) bool {
	var uaid string
	var err error

	uaid = strconv.Itoa(int(roleId))
	if channel == pb.ChatChannel_Channel_private {
		uaid, err = h.actor.Srv.GetUAIDByRoleId(roleId)
	}
	if err != nil {
		return false
	}
	nowTime, chatInfo := time.Now().Unix(), h.actor.GetChatData()
	// cd := excel.GetConfigMgr().GetCfg().CHAT_CD
	cd := 10
	lastSendTime, ok := int64(0), false

	if lastSendTime, ok = chatInfo.LastSendTime[uaid]; !ok {
		chatInfo.LastSendTime[uaid] = nowTime
		return true
	}
	if nowTime-lastSendTime <= int64(cd) {
		return false
	}

	chatInfo.LastSendTime[uaid] = nowTime
	return true
}

// IsFriendOrUnion 是否是好友或联盟
func (h *UserChatHandler) IsFriendOrUnion(roleId uint64, channel pb.ChatChannel) int32 {
	// 判断是否是好友
	if channel == pb.ChatChannel_Channel_private {
		if !h.actor.FriendHandler.IsFriend(roleId) {
			return int32(pb.ErrorCode_Chat_not_friend_ship)
		}
	}

	// 判断是否是同联盟
	if channel == pb.ChatChannel_Channel_alliance {
		if roleId != uint64(h.actor.UserAllianceHandler.getAllianceId()) {
			return int32(pb.ErrorCode_Not_Alliance_member)
		}
	}

	return int32(pb.ErrorCode_Success)
}

// BroadcastMessages 广播消息
func (h *UserChatHandler) BroadcastMessages(roleId uint64, channelId pb.ChatChannel, message []*pb.BroadMessage) error {
	uaid, err := h.actor.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return err
	}
	notify := &pb.LS2C_NotifyMessage{
		ChannelId: channelId,
		Message:   message,
	}

	// 系统消息，向联盟频道也发一遍
	if channelId == pb.ChatChannel_Channel_system {
		notify.ExtraValue = int32(h.actor.UserAllianceHandler.getAllianceId())
	}

	// 全服广播
	h.Debug("抽卡发送全服广播:", roleId)
	err = h.actor.Srv.Send2BC(uaid, notify)
	if err != nil {
		return err
	}
	return nil
}

// PushMessage2Friend 推送好友消息
func (h *UserChatHandler) PushMessage2Friend(roleId uint64, msgId int32, data []byte) error {
	uaid, err := h.actor.Srv.GetUAIDByRoleId(uint64(roleId))
	if err != nil {
		return err
	}
	in := &base.ProtoMsg{
		AppId:   h.actor.Srv.AppId,
		MsgId:   msgId,
		UserId:  uaid,
		RoleId:  0,
		UAID:    uaid,
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	}
	h.Infof("给好友[%d]推送消息[%v]", roleId, string(data))
	rsp, err := h.actor.Srv.UserEventInvoke(uaid, in)
	if rsp.ErrCode != 0 || err != nil {
		h.Error("chat PushMessage2Friend failed. errCode: %d, err: %v", rsp.ErrCode, err)
		return fmt.Errorf("chat PushMessage2Friend failed")
	}
	return nil
}

// NotifyPrivateMessage 推送私聊消息到客户端
func (h *UserChatHandler) NotifyPrivateMessage(fromRoleId, toRoleId uint64, message *pb.BroadMessage, channelId pb.ChatChannel) {

	baseInfo, err := h.actor.getRoleBaseDataByRoleId(fromRoleId)
	if err != nil {
		return
	}
	notify := &pb.LS2C_NotifyPrivateMessage{
		ChannelId: channelId,
		Message:   message,
		RoleInfo:  baseInfo.Common,
	}
	// 推送到客户端
	pid, ok := h.actor.CommonActor.UserMap[h.actor.GetUID()]
	if !ok {
		h.Debugf("chat --- notify client get pid from UserMap is err:%+v", h.actor.CommonActor.UserMap)
		return
	}
	h.Infof("chat --- notify client:%+v, gateTopic:%s", h.actor.CommonActor.UserMap, pid.GateId)
	err = h.actor.Srv.Send2Gate(strconv.Itoa(int(toRoleId)), pid, notify)
	if err != nil {
		return
	}
}

// GetMessageFromES 从ES中获取聊天记录
func (h *UserChatHandler) GetMessageFromES(myRoleId, roleId uint64, endTime int64, form, size int32) []*pb.BroadMessage {
	return nil
	// esIndex := h.GetChatIndex(myRoleId, roleId)
	// if esIndex == "" {
	// 	return nil
	// }
	// // hitSize := 30
	// infos := make([]*pb.BroadMessage, 0)
	// rangeMap := map[string]service.RangeItem{
	// 	"timeStamp": {
	// 		Min: float64(0),
	// 		Max: float64(endTime),
	// 	},
	// }
	//
	// err, hitData := h.actor.Srv.ESMultiSearchPage(esIndex, rangeMap, int(size), &sortorder.Desc, int(form))
	// if err != nil {
	// 	h.Warnf("es查询出错了: %v", err.Error())
	// 	return infos
	// }
	// for _, hit := range hitData.Hits {
	// 	temp := &pb.BroadMessage{}
	// 	if err = json.Unmarshal(hit.Source_, temp); err != nil {
	// 		continue
	// 	}
	// 	infos = append(infos, temp)
	// }
	// return infos
}

// SaveMessage2ES 聊天消息 存储到es
func (h *UserChatHandler) SaveMessage2ES(myRoleId, roleId uint64, message *pb.BroadMessage) error {
	// esIndex := h.GetChatIndex(myRoleId, roleId)
	// if esIndex == "" {
	// 	return errors.New("获取索引失败")
	// }
	// if err := h.actor.Srv.ESPutNoId(esIndex, message); err != nil {
	// 	h.Error(err)
	// 	return err
	// }
	return nil
}

// // GetChatIndex 按照roleId 大的在前面
// func (h *UserChatHandler) GetChatIndex(myRoleId, roleId uint64) string {
// 	if myRoleId > roleId {
// 		return fmt.Sprintf("%s_%s", strconv.Itoa(int(myRoleId)), strconv.Itoa(int(roleId)))
// 	}
// 	return fmt.Sprintf("%s_%s", strconv.Itoa(int(roleId)), strconv.Itoa(int(myRoleId)))
// }

// SetHasMessage 设计最后阅读时间
func (h *UserChatHandler) SetHasMessage(roleId uint64, value bool) error {
	uaid, err := h.actor.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return err
	}
	data := h.actor.GetChatData()
	data.HasMessage[uaid] = value
	if value == false {
		delete(data.HasMessage, uaid)
	}

	return h.SaveDB()
}
func (h *UserChatHandler) GetHasMessage(roleId uint64) bool {
	uaid, err := h.actor.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return false
	}
	data := h.actor.GetChatData()
	if hasMessage, ok := data.HasMessage[uaid]; ok {
		return hasMessage
	}
	return false
}

// HasMessage 是否有好友消息
func (h *UserChatHandler) HasMessage(roleId uint64) bool {
	return h.GetHasMessage(roleId)
}

// DeleteFriendChatMessage 删除好友聊天
func (h *UserChatHandler) DeleteFriendChatMessage(myRoleId, friendRoleId uint64) error {
	// esIndex := h.GetChatIndex(myRoleId, friendRoleId)
	// if esIndex == "" {
	// 	return nil
	// }
	// if err := h.actor.Srv.ESDelIndex(esIndex); err != nil {
	// 	h.Debug("delete friend chat message err:", myRoleId, friendRoleId, err)
	// 	return err
	// }
	return nil
}

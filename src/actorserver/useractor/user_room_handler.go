package useractor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/forgoer/openssl"

	"gitee.com/bychannel/musae/framework/global"
	"gitee.com/bychannel/musae/framework/utils"

	"gitee.com/bychannel/musae/framework/base"

	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/common/com_order"

	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/logger"
	"gitee.com/bychannel/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type UserRoomHandler struct {
	*UABaseHandler
}

func NewUserUserRoomHandler(actor *UserActor) *UserRoomHandler {
	h := &UserRoomHandler{UABaseHandler: NewUABaseHandler(actor, "UserRoomHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CreateRoomReq), h.CreateRoomReq)        // 创建房间 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_JoinRoomReq), h.JoinRoomReq)            // 加入房间 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_InviteIntoRoomReq), h.InviteIntoRoomReq) // 邀请加入房间S2S
	// actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_FetchUserInfoReq), h.FetchUserInfoReq) // 从userActor中获取玩家信息 S2S

	return h
}

// Init 初始化模块数据
func (h *UserRoomHandler) Init() error {
	// // 初始化
	// h.actor.OrderData = h.actor.GetOrderData()
	// // 保存
	// if err := h.SaveDB(true); err != nil {
	//	return err
	// }

	logger.Debug("init order data success. player: %s", h.actor.ID())
	return nil
}

func (h *UserRoomHandler) EnterGame() error {
	return nil
}

func (h *UserRoomHandler) DailyRefresh() error {
	return nil
}

func (h *UserRoomHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.OrderData); ok {
		h.actor.OrderData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *UserRoomHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	dbTable, dbKey := com_order.OrderDBTable(h.actor.GetUID())
	return dbTable, dbKey, h.actor.OrderData
	// return service.MongoDbType_MongoAccount, db.KeyUserOrderInfo(h.actor.GetUID()), h.actor.OrderDatas
}

func (h *UserRoomHandler) CreateRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	if h.actor.Srv.CheckInRoom(in.UserId) {
		return nil, fmt.Errorf("user room exist"), int32(pb.ErrorCode_Room_player_in_other_room)
	}
	var req pb.C2LS_CreateRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	// 分配roomActor
	reqMsg := &pb.S2S_CreateRoomReq{
		PlayType: req.PlayType,
		BaseInfo: toClientBaseInfo(h.actor.GetUserData()),
		Cards:    h.actor.getClientCardInfo(h.actor.roleId),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		logger.Debug("proto marshal err:", err)
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	roomIdSession := &pb.RoomID{
		RoomId:   fmt.Sprintf("room:%v:%s", req.PlayType, h.actor.ID()),
		PlayType: int32(req.PlayType),
		// CreateTime: time.Now().Unix(), // 房间需要复用,注释该值
	}
	logger.Debugf("roomIdSession: %+v\n", roomIdSession)
	roomIdData, err := json.Marshal(roomIdSession)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	logger.Debug("src:", len(data), data)
	roomIdData, err = openssl.AesECBEncrypt(roomIdData, []byte(common.RoomIdSecret), openssl.PKCS7_PADDING)
	if err != nil || len(roomIdData) == 0 {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	logger.Debug("aes:", len(roomIdData), roomIdData)
	roomId := base64.URLEncoding.EncodeToString(roomIdData)
	// roomId := strconv.Itoa(int(utils.GenIntUUID()))
	logger.Debug("code:", len(roomIdData), []byte(roomIdData))

	// var roomId = fmt.Sprintf("room:%v:%s", req.PlayType, h.actor.ID()) //strconv.Itoa(int(utils.GenIntUUID())) //"123" //utils.GenStrUUID()
	logger.Debugf("分配的roomId：%s", roomId)

	createRoomReqMsg := &base.ProtoMsg{
		MsgId:  int32(pb.Protocols_PS2S_CreateRoomReq),
		AppId:  global.ACTOR_SVC,
		UserId: h.actor.uid,
		RoleId: 0,
		UAID:   h.actor.Srv.UAID(h.actor.ID(), h.actor.roleId),
		Data:   data,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: utils.GenIntUUID(),
	}
	createRoomRspMsg, err := h.actor.Srv.ActorInvoke(global.RoomActorType, roomId, createRoomReqMsg)
	if err != nil {
		return nil, err, createRoomRspMsg.ErrCode
	}
	createRoomRsp := &pb.S2S_CreateRoomRes{}
	err = proto.Unmarshal(createRoomRspMsg.Data, createRoomRsp)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// // 保存RoomSession
	// roomSession := &pb.RoomSession{
	//	RoomId: roomId,
	//	Guid:   utils.GenStrUUID(),
	// }

	// 绑定玩家Id和roomId
	err = h.actor.Srv.SaveRoomBindingData(h.actor.uid, roomId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 持久化
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_CreateRoomRes{
		RoomSimple: createRoomRsp.RoomSimple,
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *UserRoomHandler) JoinRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	if h.actor.Srv.CheckInRoom(in.UserId) {
		return nil, fmt.Errorf("user room exist"), int32(pb.ErrorCode_Room_player_in_other_room)
	}

	var req pb.C2LS_JoinRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 分配roomActor
	reqMsg := &pb.S2S_JoinRoomReq{
		RoomId:     req.RoomId,
		RoomSecret: req.RoomSecret,
		BaseInfo:   toClientBaseInfo(h.actor.GetUserData()),
		Cards:      h.actor.getClientCardInfo(h.actor.roleId),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		logger.Debug("proto marshal err:", err)
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	joinRoomReqMsg := &base.ProtoMsg{
		MsgId:  int32(pb.Protocols_PS2S_JoinRoomReq),
		AppId:  global.ACTOR_SVC,
		UserId: h.actor.uid,
		RoleId: 0,
		UAID:   h.actor.Srv.UAID(h.actor.ID(), h.actor.roleId),
		Data:   data,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: utils.GenIntUUID(),
	}
	joinRoomRspMsg, err := h.actor.Srv.ActorInvoke(global.RoomActorType, req.RoomId, joinRoomReqMsg)
	if err != nil {
		h.Debugf("返回errCode, err:%+v", err.Error())
		return nil, err, joinRoomRspMsg.ErrCode
	}
	joinRoomRsp := &pb.S2S_JoinRoomRes{}
	err = proto.Unmarshal(joinRoomRspMsg.Data, joinRoomRsp)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 绑定玩家Id和roomId
	err = h.actor.Srv.SaveRoomBindingData(h.actor.uid, req.RoomId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_JoinRoomRes{
		RoomSimple: joinRoomRsp.RoomSimple,
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *UserRoomHandler) InviteIntoRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.S2S_InviteIntoRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	chatMsg := make([]string, 0)
	chatMsg = append(chatMsg, strconv.Itoa(int(h.actor.roleId)), req.RoomId, strconv.Itoa(int(req.PlayType)))
	message := &pb.BroadMessage{
		MType:      pb.MessageType_Message_Type_invited,
		FromRoleId: h.actor.roleId,
		Data:       chatMsg,
		TimeStamp:  time.Now().Unix(),
	}

	h.actor.UserChatHandler.PushMessageToFriend(h.actor.roleId, req.ToRoleId, message, pb.ChatChannel_Channel_private, false)

	resp := &pb.S2S_InviteIntoRoomRes{}

	return resp, nil, int32(pb.ErrorCode_Success)
}

// func (h *UserRoomHandler) FetchUserInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
//	var (
//		err error
//	)
//
//	var req pb.S2S_FetchUserInfoReq
//	err = in.UnmarshalData(&req)
//	if err != nil {
//		return nil, err, int32(pb.ErrorCode_DeSerializeError)
//	}
//
//	rsp := &pb.S2S_FetchUserInfoRes{
//		BaseInfo: toClientBaseInfo(h.actor.GetUserData()),
//		Cards:    h.actor.getClientCardInfo(h.actor.roleId),
//	}
//
//	return rsp, nil, int32(pb.ErrorCode_Success)
// }

// 尝试被动退出房间
func (h *UserRoomHandler) tryExitRoom() {
	data, err, _ := h.actor.Srv.GetRoomBindingData(h.actor.uid)
	if err != nil {
		h.Infof("房间绑定数据拉取失败 err: %v", err)
		return
	}

	if data.RoomId == "" {
		h.Debugf("没有老房间, 不做任何处理...")
		return
	}
	// 有老房间，尝试退出
	reqMsg := &pb.S2S_ExitRoomReq{}
	rspMsg := &pb.S2S_ExitRoomRes{}
	err, _ = h.RoomInvoke(data.RoomId, int32(pb.Protocols_PS2S_ExitRoomReq), reqMsg, rspMsg)
	if err != nil {
		h.Error(err)
		return
	}
	h.Debugf("退出老房间了")
}

// 封装房间actor调用方法
func (h *UserRoomHandler) RoomInvoke(roomId string, msgId int32, reqMsg proto.Message, rspData proto.Message) (error, pb.ErrorCode) {
	callData, err := proto.Marshal(reqMsg)
	if err != nil {
		return err, pb.ErrorCode_SerializeError
	}
	protoMsg := &base.ProtoMsg{
		MsgId:  msgId,
		AppId:  global.ACTOR_SVC,
		UserId: h.actor.uid,
		RoleId: h.actor.roleId,
		UAID:   h.actor.ID(),
		Data:   callData,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: utils.GenIntUUID(),
	}
	rspMsg, err := h.actor.Srv.ActorInvoke(global.RoomActorType, roomId, protoMsg)
	if err != nil {
		h.Error(err)
	}
	if rspMsg.ErrCode > 0 {
		return err, pb.ErrorCode(rspMsg.ErrCode)
	}

	// 返回数据解析
	if rspData != nil {
		err = proto.Unmarshal(rspMsg.Data, rspData)
		if err != nil {
			return err, pb.ErrorCode_DeSerializeError
		}
	}
	return nil, pb.ErrorCode_Success
}

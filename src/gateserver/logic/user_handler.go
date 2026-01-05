package logic

import (
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"time"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"google.golang.org/protobuf/proto"
)

func (u *User) HandleUserActor(reqMessageID int32, reqData []byte, reqIdx uint32) ([]byte, pb.Protocols, pb.ErrorCode) {
	msg, err := u.s.UserInvoke(u.uaid, &base.ProtoMsg{
		AppId:   u.s.AppId,
		MsgId:   reqMessageID,
		UserId:  u.uid,
		RoleId:  u.roleId,
		UAID:    u.uaid,
		Data:    reqData,
		ErrCode: 0,
		ReqIdx:  reqIdx,
		Topic:   u.s.PrivateTopicID(),
	})
	logger.Debugf("OnNetMessage, UserInvoke end, msg: %+v, err:%+v", msg.Str(), err)

	respMessageID, respData := msg.MsgId, msg.Data
	if respMessageID > 0 {
		if respMessageID == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
			if msg.ErrCode == int32(pb.ErrorCode_RepeatMsg) {
				logger.Debugf("OnNetMessage, 防重放中获取到数据返回, actor:%s, %s, %v, %d, %d", stub.RoomActorType, u.String(), pb.Protocols(respMessageID), respMessageID, len(respData))

				lastRespData, lastDownId := u.s.reqRepeated(nil, reqMessageID, reqIdx, u.session)
				return lastRespData, pb.Protocols(lastDownId), pb.ErrorCode_Success
			}

			rsp := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(msg.ErrCode), Param: []string{string(respData)}}
			// err = u.Reply(int32(pb.Protocols_PS2C_ErrorCodeNtf), rsp)
			b, err := proto.Marshal(rsp)
			if err != nil {
				logger.Debug("OnNetMessage, UserInvoke end, proto.Marshal got error:",
					u.String(), pb.Protocols(respMessageID), respMessageID, len(respData), err.Error())
				return nil, pb.Protocols_Protocols_None, pb.ErrorCode(msg.ErrCode)
			}

			logger.Debug("OnNetMessage, UserInvoke end got errorCode:",
				u.String(), pb.Protocols(respMessageID), respMessageID, pb.ErrorCode(msg.ErrCode), string(respData))
			return b, pb.Protocols_PS2C_ErrorCodeNtf, pb.ErrorCode(msg.ErrCode)
			// err = u.ReplyWithBody(int32(pb.Protocols_PS2C_ErrorCodeNtf), b)
			// if err != nil {
			//	logger.Warn("OnNetMessage, UserInvoke, reply error:", u.String(), pb.Protocols(respMessageID), respMessageID, errorx.Wrap(err).Error())
			// }

		} else {
			// err = u.ReplyWithBody(respMessageID, respData)
			// if err != nil {
			//	logger.Warn("OnNetMessage, UserInvoke ReplyWithBody err: ", errorx.Wrap(err).Error())
			// }
			logger.Debug("OnNetMessage, UserInvoke end:", u.String(), pb.Protocols(respMessageID), respMessageID, len(respData))
			return respData, pb.Protocols(respMessageID), pb.ErrorCode(msg.ErrCode)
		}
	}
	return nil, pb.Protocols_Protocols_None, pb.ErrorCode(msg.ErrCode)
}

func (u *User) HandleRoomActor(reqMessageID int32, reqData []byte, reqIdx uint32) ([]byte, pb.Protocols, pb.ErrorCode) {
	// 根据玩家的uid获取所在roomId
	binding, err, errCode := u.s.GetRoomBindingData(u.uid)
	if err != nil {
		return nil, pb.Protocols_PS2C_ErrorCodeNtf, errCode
	}

	msg, err := u.s.ActorInvoke(stub.RoomActorType, binding.RoomId, &base.ProtoMsg{
		AppId:   u.s.AppId,
		MsgId:   reqMessageID,
		UserId:  u.uid,
		RoleId:  u.roleId,
		UAID:    u.uaid,
		Data:    reqData,
		ErrCode: 0,
		Topic:   u.s.PrivateTopicID(),
		ReqIdx:  reqIdx,
	})
	if pb.ErrorCode(msg.ErrCode) == pb.ErrorCode_RepeatMsg {
		logger.Debugf("OnNetMessage, 防重放中获取到数据返回, actor:%s, %s, %v, %d, %d", stub.RoomActorType, u.String(), pb.Protocols(reqMessageID), reqMessageID, len(msg.Data))
		lastRespData, lastDownId := u.s.reqRepeated(nil, reqMessageID, reqIdx, u.session)
		// return lastRespData, pb.Protocols(lastDownId), pb.ErrorCode_Success
		msg.ErrCode = int32(pb.ErrorCode_Success)
		msg.Data = lastRespData
		msg.MsgId = lastDownId

	} else if err != nil {
		logger.Debugf("HandleRoomActor ActorInvoke got error, actor:%s, msg: %+v, err:%+v", stub.RoomActorType, msg.Str(), err)
		return nil, pb.Protocols_PS2C_ErrorCodeNtf, pb.ErrorCode(msg.ErrCode)

	}
	logger.Debugf("OnNetMessage, ActorInvoke end, actor:%s, msg: %+v, err:%+v", stub.RoomActorType, msg.Str(), err)

	respMessageID, respData := msg.MsgId, msg.Data
	if respMessageID > 0 {
		if respMessageID == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
			//
			// if msg.ErrCode == int32(pb.ErrorCode_RepeatMsg) {
			//	logger.Debugf("OnNetMessage, 防重放中获取到数据返回, actor:%s, :", global.RoomActorType, u.String(), pb.Protocols(respMessageID), respMessageID, len(respData))
			//
			//	rspData, lastDownId := u.s.reqRepeated(u.ctx, reqMessageID, reqIdx, u.session)
			//	return rspData, pb.Protocols(lastDownId), pb.ErrorCode(msg.ErrCode)
			// }

			rsp := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(msg.ErrCode), Param: []string{string(respData)}}
			// err = u.Reply(int32(pb.Protocols_PS2C_ErrorCodeNtf), rsp)
			b, err := proto.Marshal(rsp)
			if err != nil {
				logger.Debug("OnNetMessage, ActorInvoke end, actor:%s, proto.Marshal got error:",
					stub.RoomActorType, u.String(), pb.Protocols(respMessageID), respMessageID, len(respData), err.Error())
				return nil, pb.Protocols_Protocols_None, pb.ErrorCode(msg.ErrCode)
			}

			logger.Debug("OnNetMessage, ActorInvoke end actor:%s, got errorCode:",
				stub.RoomActorType, u.String(), pb.Protocols(respMessageID), respMessageID, pb.ErrorCode(msg.ErrCode), string(respData))
			return b, pb.Protocols_PS2C_ErrorCodeNtf, pb.ErrorCode(msg.ErrCode)
			// err = u.ReplyWithBody(int32(pb.Protocols_PS2C_ErrorCodeNtf), b)
			// if err != nil {
			//	logger.Warn("OnNetMessage, UserInvoke, reply error:", u.String(), pb.Protocols(respMessageID), respMessageID, errorx.Wrap(err).Error())
			// }

		} else {
			// err = u.ReplyWithBody(respMessageID, respData)
			// if err != nil {
			//	logger.Warn("OnNetMessage, UserInvoke ReplyWithBody err: ", errorx.Wrap(err).Error())
			// }
			logger.Debug("OnNetMessage, ActorInvoke end actor:%s, :", stub.RoomActorType, u.String(), pb.Protocols(respMessageID), respMessageID, len(respData))
			return respData, pb.Protocols(respMessageID), pb.ErrorCode(msg.ErrCode)
		}
	}
	return nil, pb.Protocols_Protocols_None, pb.ErrorCode(msg.ErrCode)
}

// func (u *User) HandleHeartbeat(msgId int32) ([]byte, pb.Protocols) {
//	u.lastHeartbeatTs = time.Now().Unix()
//	res := &pb.C2LS_HeartBeatRes{}
//	b, err := proto.Marshal(res)
//	if err != nil {
//		return nil, pb.Protocols_Protocols_None
//	}
//	//err = u.ReplyWithBody(int32(pb.Protocols_PC2LS_HeartBeatRes), b)
//	//if err != nil {
//	//	logger.Warn("OnNetMessage, invoke lobby, reply error:", u.String(), pb.Protocols(msgId), msgId, errorx.Wrap(err).Error())
//	//}
//	return b, pb.Protocols_PC2LS_HeartBeatRes
// }

/*func (u *User) UserInvoke(msgId int32, data []byte) (*base.ProtoMsg, error) {
	in := &base.ProtoMsg{}
	in.MsgId = msgId
	in.Data = data
	in.UserId = u.uid
	in.RoleId = u.roleId
	in.UAID = u.uaid
	switch conf.Base().Actor2GateType {
	case base.Actor2GateOnRpc:
		in.AppId = u.s.AppId
	case base.Actor2GateOnCh:
		in.AppId = u.s.PrivateTopic
	}
	in.GUID = utils.GenIntUUID()
	ctx := context.Background()
	md := metadata.Pairs("msg-id", fmt.Sprintf("%v", pb.Protocols(in.MsgId)))
	ctx = metadata.NewOutgoingContext(ctx, md)
	logger.Debugf("OnNetMessage UserInvoke Begin, msgId:%v, %v, %v", pb.Protocols(msgId), u.String(), in.String())
	return u.actor.UserInvoke(ctx, in)
}

func (u *User) UserInvokeByMsg(msgId int32, msg proto.Message) (*base.ProtoMsg, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	in := &base.ProtoMsg{}
	in.MsgId = msgId
	in.Data = data
	in.UserId = u.uid
	switch conf.Base().Actor2GateType {
	case base.Actor2GateOnRpc:
		in.AppId = u.s.AppId
	case base.Actor2GateOnCh:
		in.AppId = u.s.PrivateTopic
	}
	in.GUID = utils.GenIntUUID()
	ctx := context.Background()
	md := metadata.Pairs("msg-id", fmt.Sprintf("%v", pb.Protocols(in.MsgId)))
	ctx = metadata.NewOutgoingContext(ctx, md)
	logger.Debugf("OnNetMessage UserInvokeByMsg Begin, msgId:%v, %v, %v", pb.Protocols(msgId), u.String(), in.String())
	return u.actor.UserInvoke(ctx, in)
}*/

func (u *User) Logout(account string, reason string) error {
	logger.Infof("[logout] user %s account,reason %s", account, reason)
	u.offlineTs = time.Now().Unix()
	return u.ctx.CloseConn()
}

func (u *User) HandleDeprecatedMsg(messageID int32) ([]byte, pb.Protocols) {
	logger.Debugf("OnNetMessage, HandleDeprecatedMsg, msgId: %d", messageID)

	rsp := &pb.S2C_ErrorCodeNtf{
		ErrorCode: uint32(pb.ErrorCode_DeprecatedMsgError),
		Param:     []string{"功能临时关闭"},
	}
	// return rsp, int32(pb.Protocols_PS2C_ErrorCodeNtf)
	b, err := proto.Marshal(rsp)
	if err != nil {
		logger.Warn(err)
		return nil, pb.Protocols_Protocols_None
	}
	return b, pb.Protocols_PS2C_ErrorCodeNtf
	// err = u.ReplyWithBody(int32(pb.Protocols_PS2C_ErrorCodeNtf), b)
	// if err != nil {
	//	logger.Warn("OnNetMessage, HandleDeprecatedMsg, reply error:", u.String(), pb.Protocols(messageID), messageID, errorx.Wrap(err).Error())
	// }
	//
	// logger.Debug("OnNetMessage, HandleDeprecatedMsg End:", u.String(), pb.Protocols(messageID), messageID)
}

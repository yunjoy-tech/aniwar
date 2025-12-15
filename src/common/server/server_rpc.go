package server

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/aniwar2/musae/framework/gamelib/guid"
	"net/http"
	"strconv"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/http/request"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/metrics"
	svc "gitee.com/aniwar2/musae/framework/service"
	"gitee.com/aniwar2/musae/framework/utils"
	dapr "github.com/dapr/go-sdk/client"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// 全服广播
func (s *Server) Send2BC(aid string, msg proto.Message) error {
	for _, gate := range global.GateServices {
		_ = s.PubTopicEvent(svc.EVENT_PRIVATE, gate, aid, nil, msg)
		logger.Debugf("aid:%s, gate:%s, msg:%s", aid, utils.PrettyJson(msg))
	}
	return nil
}

func (s *Server) Send2Gates(aid string, pids map[string]*pb.ActorUserInfo, msg proto.Message) error {
	pidMap := map[string][]string{}
	for _, pid := range pids {
		if _, ok := pidMap[pid.GateId]; !ok {
			pidMap[pid.GateId] = make([]string, 0)
		}
		pidMap[pid.GateId] = append(pidMap[pid.GateId], pid.Uid)
	}

	for k, v := range pidMap {
		_ = s.PubTopicEvent(svc.EVENT_PRIVATE, k, aid, v, msg)
	}

	return nil
}

func (s *Server) Send2Gate(aid string, pid *pb.ActorUserInfo, msg proto.Message) error {
	if pid == nil {
		return nil
	}
	return s.PubTopicEvent(svc.EVENT_PRIVATE, pid.GateId, aid, []string{pid.Uid}, msg)
}

// PubTopicEvent
//
//	@Description: 发布 主题事件消息
//	@receiver s
//	@param eType ：EVENT_PRIVATE,EVENT_AppId,EVENT_Global
//	@param topicName: EVENT_PRIVATE:s.PrivateTopic, EVENT_AppId:s.AppId, EVENT_Global:"global"
//	@param aid : actor id
//	@param uids : user id list
//	@param msg ： proto message
//	@return error ： 返回错误
func (s *Server) PubTopicEvent(eType svc.EVENT_TYPE, topicName, aid string, uids []string, msg proto.Message) error {
	logger.Debugf("PubTopicEvent type:%v uids:%v topic:%v aid:%v msg:%+v", eType, uids, topicName, aid, utils.PrettyJson(msg))

	ctx := context.Background()
	var data []byte
	var msgId int32
	var md metadata.MD
	var ok bool
	var err error
	if (eType == svc.EVENT_GLOBAL && topicName != svc.GLOBAL_TOPIC) ||
		(eType == svc.EVENT_APPID && !IsValidAppId(topicName)) {
		err = fmt.Errorf("service publish topic event, topic name error")
		goto OnFail
	}

	data, err = proto.Marshal(msg)
	if err != nil {
		goto OnFail
	}

	msgId, ok = pb.Protocols_value[string("P"+msg.ProtoReflect().Descriptor().Name())]
	if !ok {
		err = fmt.Errorf("invalid msgId")
		goto OnFail
	}
	data, err = base.PackProtoMsg(msgId, "", 0, aid, data, s.AppId, uids)
	if err != nil {
		goto OnFail
	}

	logger.Debugf("PubTopicEvent ProtoData msgId:%v len:%v data:%+v", pb.Protocols(msgId), len(data), data)

	md = metadata.Pairs("msg-id", fmt.Sprintf("%v", pb.Protocols(msgId)))
	ctx = metadata.NewOutgoingContext(ctx, md)
	if err = s.Daprc.PublishEvent(ctx, string(eType), topicName, data,
		dapr.PublishEventWithMetadata(map[string]string{svc.REDIS_TTL_NAME: strconv.Itoa(svc.PUBSUB_TTL_SEC)}),
		// dapr.PublishEventWithContentType("application/json"),
		dapr.PublishEventWithContentType("application/octet-stream"), // application/octet-stream
	); err != nil {
		goto OnFail
	}
	metrics.GaugeInc(metrics.MsgPubCount)
	return nil

OnFail:
	logger.Warnf("service publish topic event error:%v type:%v uids:%v topic:%v aid:%v msgId:%v msg:%+v", err, eType, uids, aid, topicName, msgId, utils.PrettyJson(msg))
	return err
}

// SvcInvoke
//
//	@Description: rpc call between services
//	@receiver s
//	@param appId
//	@param uid  账号ID
//	@param playerId UserActor id
//	@param msg
//	@return out
//	@return err
func (s *Server) SvcInvoke(appId, uid string, roleId uint64, uaid string, msg proto.Message) (out []byte, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("[SvcInvoke] recover, err: ", err)
		}
	}()

	if appId == "" {
		logger.Warnf("[server]:SvcInvoke invalid msg: uid:%s appId:%v uaid:%s", uid, appId, uaid)
		return nil, fmt.Errorf("appid error")
	}

	name := msg.ProtoReflect().Descriptor().Name()
	msgId, ok := pb.Protocols_value[string("P"+name)]
	if !ok {
		logger.Warnf("[server]:SvcInvoke invalid msg: %v %v %+v", uid, appId, msg)
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		logger.Debug("proto marshal err:", err)
		return nil, err
	}
	return s.SvcInvokeByData(appId, uid, roleId, uaid, msgId, data)
}

func (s *Server) SvcInvokeByData(appId, uid string, roleId uint64, uaid string, msgId int32, data []byte) (out []byte, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("[SvcInvokeByData] recover err: ", err)
		}
	}()

	msg := base.ProtoMsg{
		MsgId:  msgId,
		AppId:  appId,
		UserId: uid,
		RoleId: roleId,
		UAID:   uaid,
		Data:   data,
	}
	data, err = msg.Marshal()
	if err != nil {
		logger.Debug("SvcInvokeByData msg marshal err :", appId, pb.Protocols(msgId), uid, err)
		return nil, err
	}
	logger.Debugf("SvcInvokeByData Begin  appId:%s msgId:%v uid:%s len:%v", appId, pb.Protocols(msgId), uid, len(data))

	ctx := context.Background()
	startTime := time.Now()
	md := metadata.Pairs("msg-id", fmt.Sprintf("%v", pb.Protocols(msgId)))
	ctx = metadata.NewOutgoingContext(ctx, md)
	out, err = s.Daprc.InvokeMethodWithContent(ctx, appId, "RpcCall", "post",
		&dapr.DataContent{Data: data, ContentType: "text/plain"}) // text/plain
	metrics.HistogramPut(metrics.SrvInvokeDelayHist, time.Since(startTime).Milliseconds(), metrics.Invoke)
	metrics.GaugeInc(metrics.InvokePubCount)
	logger.Debugf("SvcInvokeByData End appId:%s msgId:%v uid:%s err:%v", appId, pb.Protocols(msgId), uid, err)
	return out, err
}

func (s *Server) UserInvoke(uaid string, msg *base.ProtoMsg) (*base.ProtoMsg, error) {
	return s.userInvoke(uaid, msg, false)
}

func (s *Server) UserEventInvoke(uaid string, msg *base.ProtoMsg) (*base.ProtoMsg, error) {
	return s.userInvoke(uaid, msg, true)
}

func (s *Server) userInvoke(uaid string, msg *base.ProtoMsg, isEvent bool) (*base.ProtoMsg, error) {
	startTime := time.Now()
	if msg.ReqIdx == 0 {
		msg.ReqIdx = guid.GenIntUuid()
	}

	invokeName := "UserInvoke"
	if isEvent {
		invokeName = "UserInvokeEvent"
	}
	logger.Debugf("===>>>%s-Begin Msg:%v UAID:%s MSG-REQ:%s,", invokeName, pb.Protocols(msg.MsgId), uaid, msg.Str())
	userStub := stub.NewUserStub(uaid)
	s.ImpActorStub(userStub)
	ctx := context.Background()
	md := metadata.Pairs("msg-id", fmt.Sprintf("%v", pb.Protocols(msg.MsgId)))
	ctx = metadata.NewOutgoingContext(ctx, md)
	var ret *base.ProtoMsg
	var err error
	if isEvent {
		ret, err = userStub.EventInvoke(ctx, msg)
	} else {
		ret, err = userStub.UserInvoke(ctx, msg)
	}

	delayTime := time.Since(startTime).Milliseconds()
	metrics.HistogramPut(metrics.UserInvokeDelayHist, delayTime, metrics.Invoke)
	var errStr string
	if err != nil {
		errStr = err.Error()
	} else if ret != nil && ret.MsgId == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
		errStr = string(ret.Data)
	}
	logger.Debugf("===>>>%s-End Delay:%d Msg:%v UAID:%s MSG-RET:%s Err:%v", invokeName, delayTime, pb.Protocols(msg.MsgId), uaid, ret.Str(), errStr)
	logger.WarnDelayf(delayTime, "")

	return ret, err
}

func (s *Server) ActorInvoke(actorType, actorId string, msg *base.ProtoMsg) (*base.ProtoMsg, error) {
	startTime := time.Now()
	if msg.ReqIdx == 0 {
		msg.ReqIdx = guid.GenIntUuid()
	}
	ctx := context.Background()
	md := metadata.Pairs("msg-id", fmt.Sprintf("%v", pb.Protocols(msg.MsgId)))
	ctx = metadata.NewOutgoingContext(ctx, md)
	if pb.Protocols(msg.MsgId) != pb.Protocols_PS2S_SvcStatusReq {
		logger.Debugf("===>>>ActorInvoke-Begin Actor:%v Msg:%v AID:%s MSG-REQ:%s", actorType, pb.Protocols(msg.MsgId), actorId, msg.Str())
	}
	var ret *base.ProtoMsg
	var invokeErr error
	switch actorType {
	case global.RoomActorType:
		stub := stub.NewRoomStub(actorId)
		s.ImpActorStub(stub)
		ret, invokeErr = stub.Invoke(ctx, msg)
	case global.AllianceActorType:
		stub := stub.NewAllianceStub(actorId)
		s.ImpActorStub(stub)
		ret, invokeErr = stub.Invoke(ctx, msg)
	case global.CenterActorType:
		stub := stub.NewCenterStub(actorId)
		s.ImpActorStub(stub)
		ret, invokeErr = stub.Invoke(ctx, msg)
	case global.MailActorType:
		stub := stub.NewMailStub(actorId)
		s.ImpActorStub(stub)
		ret, invokeErr = stub.Invoke(ctx, msg)
	}
	delayTime := time.Since(startTime).Milliseconds()
	metrics.HistogramPut(metrics.RoomInvokeDelayHist, delayTime, metrics.Invoke)

	var err error
	if invokeErr != nil {
		err = invokeErr
		// ret.ErrCode = int32(pb.ErrorCode_RpcInvokeError) // 返给客户端errCode
	} else if ret != nil && ret.MsgId == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
		err = errors.New(string(ret.Data))
	}
	if pb.Protocols(msg.MsgId) != pb.Protocols_PS2S_SvcStatusReq {
		logger.Debugf("===>>>ActorInvoke-End Actor:%v Delay:%d Msg:%v AID:%s MSG-RET:%s Err:%+v", actorType, delayTime, pb.Protocols(msg.MsgId), actorId, ret.Str(), err)
	}
	logger.WarnDelayf(delayTime, "ActorInvoke Actor:%s", actorType)
	return ret, err
}

func (s *Server) DeleteActor(actorType, actorId string) error {
	url := fmt.Sprintf("http://localhost%s/actors/%s/%s", s.InAddr, actorType, actorId)
	logger.Infof("===>>> request url: %v", url)

	resp, err := request.New().Method(http.MethodDelete).JSONBytesBody(nil).Send(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 返回值200成功
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete actor failed. status code is %v", resp.StatusCode)
	}

	logger.Infof("===>>> delete actor success. actorType: %v, actorId: %v", actorType, actorId)
	return nil
}

// UserActor调用,isEvent为true使用离线模式
func (s *Server) CallUserActor(isEvent bool, roleId uint64, msgId int32, reqMsg proto.Message, rspMsg proto.Message) (error, pb.ErrorCode) {
	callData, err := proto.Marshal(reqMsg)
	if err != nil {
		return err, pb.ErrorCode_SerializeError
	}
	uaid, err := s.GetUAIDByRoleId(roleId)
	if err != nil {
		return err, pb.ErrorCode_NotFoundPlayer
	}
	in := &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   msgId,
		UserId:  "",
		RoleId:  roleId,
		UAID:    uaid,
		Data:    callData,
		ErrCode: 0,
		ReqIdx:  guid.GenIntUuid(),
		Topic:   "",
	}
	var rsp *base.ProtoMsg
	if isEvent {
		rsp, err = s.UserEventInvoke(uaid, in)
	} else {
		rsp, err = s.UserInvoke(uaid, in)
	}
	if rsp.ErrCode != 0 || err != nil {
		logger.Error("UserEventInvoke handle failed. errCode: %d, err: %v", rsp.ErrCode, err)
		return fmt.Errorf("UserEventInvoke handle failed"), pb.ErrorCode_RpcInvokeError
	}
	// 解析数据
	if rspMsg != nil {
		err = proto.Unmarshal(rsp.Data, rspMsg)
		if err != nil {
			return err, pb.ErrorCode_DeSerializeError
		}
	}
	return nil, pb.ErrorCode_Success
}

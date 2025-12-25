package logic

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"strings"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/conf"

	"gitee.com/aniwar2/aniwar/src/common/auth"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"github.com/dapr/go-sdk/service/common"
)

func (s *GateServer) OnHttp(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnGate failed, err: ", err)
		}
	}()

	curTime := time.Now()
	upMsgLen := len(in.Data)

	if in == nil {
		return nil, fmt.Errorf("nil parameter")
	}
	// logger.Debugf("[GateServer] [LoginStep] OnLogin ContentType:%s, Verb:%s, QueryString:%s, len:%d", in.ContentType, in.Verb, in.QueryString, upMsgLen)

	if upMsgLen > conf.Base().GateMsgMaxSize {
		return nil, fmt.Errorf("invalid error")
	}

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}
	// check session
	authToken := in.Request.Header.Get("auth-token")
	// authToken := in.Request.Header.Get("cookie")
	clientVersion := in.Request.Header.Get("client-version")
	channel := in.Request.Header.Get("channel")
	platform := in.Request.Header.Get("platform")
	deviceId := in.Request.Header.Get("device-id")
	// 客户端版本验证
	if conf.Base().VersionCheck {
		logger.Infof("VersionCheck clientVersion:%s", clientVersion)
		err = s.VersionCheckExt(platform, clientVersion)
		if err != nil {
			logger.Warnf("VersionCheck error:%s", err.Error())
			out.Data = s.ErrorPack(pb.ErrorCode_VersionLimit)
			return out, nil
		}
	}

	session, errCode := s.GetSession(authToken)
	if errCode != pb.ErrorCode_Success || session == nil {
		out.Data = s.ErrorPack(errCode)
		return out, nil
	}
	logger.Debugf("clientVersion:%s, channel:%s, deviceId:%s", clientVersion, channel, deviceId)

	if err, errCode = s.CheckToken(session, authToken); err != nil {
		out.Data, err = s.Pack(pb.Protocols_PS2C_ErrorCodeNtf, errCode, &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(errCode)}, "")
		if err != nil {
			out.Data = []byte(err.Error())
			logger.Warnf("PackWithBody err:%s", string(out.Data))
		}
		return out, err
	}

	// handle gate msg
	data, msgId, errCode := s.HandlerGate(in, session)
	if errCode != pb.ErrorCode_Success || msgId == pb.Protocols_Protocols_None {
		out.Data = s.ErrorPack(errCode)
		logger.Errorf("HandlerGate got fail, msgId=%d, errCode:%d", msgId, errCode)
		return out, nil
	}

	metrics.GaugeInc(metrics.GateMsgCount)
	metrics.GaugeAdd(metrics.GateUpMsgSize, int64(upMsgLen))
	metrics.GaugeAdd(metrics.GateDownMsgSize, int64(len(data)))
	out.Data, err = s.PackWithBody(msgId, errCode, data, session.CryptKey)
	if err != nil {
		out.Data = []byte(err.Error())
		logger.Warnf("PackWithBody err:%s", string(out.Data))
	}

	delayTime := time.Since(curTime).Milliseconds()
	var idx uint32
	if session.LastRspData != nil {
		idx = session.LastRspData.ReqIdx
	}
	metrics.HistogramPut(metrics.GateDelayHist, delayTime, metrics.Delay)
	logger.Debugf("===>>>OnGateDelay msg:%v idx:%v uid:%s Delay:%d len:%v", msgId, idx, session.Uid, delayTime, len(out.Data))
	logger.WarnDelayf(delayTime, "")
	taptap.GateDelayComm(session.Uid, nil, nil, int32(msgId), delayTime)

	return out, nil
}

func (s *GateServer) HandlerGate(in *common.InvocationEvent, session *pb.UserSession) ([]byte, pb.Protocols, pb.ErrorCode) {
	//
	// if now > session.LimitTs {
	//	return nil, pb.Protocols_Protocols_None, pb.ErrorCode_TokenTimeout
	// }
	if conf.Base().IsDebug {
		logger.Debugf("user session:%+v", session)
	}

	data, e := tcpx.Decrypt(in.Data, session.CryptKey)
	if e != nil {
		logger.Warn(e)
		return nil, pb.Protocols_Protocols_None, pb.ErrorCode_Crypt
	}
	msgId, e := tcpx.MessageIDOf(data)
	if e != nil {
		logger.Warn(errorx.Wrap(e, "").Error())
		return nil, pb.Protocols_Protocols_None, pb.ErrorCode_UnKnownMsg
	}
	logger.Debug("OnMessage: ", in.Request.RemoteAddr, msgId, len(data))

	reqIdx, err := tcpx.ReqIndexOf(data)
	if err != nil {
		logger.Warn("OnNetMessage ReqIndexOf", errorx.Wrap(err, "").Error())
		return nil, pb.Protocols_Protocols_None, pb.ErrorCode_UnKnownMsg
	}
	// logger.Infof(" c.GetReqIndex() : %d", reqIdx)

	// 防重放
	rspData, downMsgId := s.reqRepeated(nil, msgId, reqIdx, session)
	if rspData != nil && downMsgId != int32(pb.Protocols_Protocols_None) {
		logger.Debugf("OnNetMessage ReqRepeated msgId:%d reqIdx:%d downMsgId:%d uid:%v",
			msgId, reqIdx, downMsgId, session.Uid)
		return rspData, pb.Protocols(downMsgId), pb.ErrorCode_Success
	}

	data, e = tcpx.BodyBytesOf(data)
	if e != nil {
		logger.Warn("OnNetMessage BodyBytesOf", e.Error())
		return nil, pb.Protocols_Protocols_None, pb.ErrorCode_UnKnownMsg
	}
	// 包体大小限制
	if len(data) > conf.Base().GateMsgMaxSize {
		logger.Warn("OnNetMessage msg body too big")
		return nil, pb.Protocols_Protocols_None, pb.ErrorCode_MsgBodyLimit
	}

	var messageId pb.Protocols
	var errCode = pb.ErrorCode_Success
	switch pb.Protocols(msgId) {
	/*case pb.Protocols_PC2LS_RsaClientRandomReq:
	data, session.CryptKey, e = s.HandleRsa(nil, msgId, data)
	messageId = pb.Protocols_PLS2C_RsaServerRandomRes*/
	case pb.Protocols_PC2G_LoginGateReq:
		data, e = s.HandleLoginGate(nil, msgId, data, reqIdx)
		messageId = pb.Protocols_PG2C_LoginGateRes
	case pb.Protocols_PC2G_LoginGameReq:
		// 判断秘钥是否生成
		if conf.Base().UseEncrypt == 1 && session.CryptKey == "" { // 默认秘钥为空
			logger.Warnf("服务器配置:%d, 秘钥没有重新生成, 当前秘钥:%s", conf.Base().UseEncrypt, session.CryptKey)
			metrics.GaugeInc(metrics.EnterFailedCount)
			return nil, pb.Protocols_Protocols_None, pb.ErrorCode_ReLogin
		}

		var err *base.RpcError
		data, err = s.HandleLoginGame(nil, session, msgId, data, reqIdx)
		if err != nil {
			logger.Errorf("HandleLoginGame error, %v %v", in.Request.RemoteAddr, err.Error())
			metrics.GaugeInc(metrics.EnterFailedCount)
			return nil, pb.Protocols_Protocols_None, pb.ErrorCode(err.Code)
		}
		metrics.GaugeInc(metrics.EnterSucceedCount)
		messageId = pb.Protocols_PG2C_LoginGameRes
	default:
		// 判断秘钥是否生成
		if conf.Base().UseEncrypt == 1 && session.CryptKey == "" { // 默认秘钥为空
			logger.Warnf("服务器配置:%d, 秘钥没有重新生成, 当前秘钥:%s", conf.Base().UseEncrypt, session.CryptKey)
			metrics.GaugeInc(metrics.GateAuthFailCount)
			return nil, pb.Protocols_Protocols_None, pb.ErrorCode_ReLogin
		}
		user := NewUserExt(session, s)
		data, messageId, errCode = user.HandleClientMsg(msgId, data, reqIdx)
		if data != nil && errCode == pb.ErrorCode_Success {
			session.LastRspData = &pb.LastRspData{
				ReqIdx:  reqIdx,
				UpCmd:   msgId,
				DownCmd: int32(messageId),
				RspData: data,
			}
		}
	}
	session.LastHeartbeatTs = time.Now().Unix()
	s.SaveUserSession(session)

	return data, messageId, errCode
}

func (s *GateServer) GetSession(authToken string) (*pb.UserSession, pb.ErrorCode) {
	// logger.Debugf("user token:%v", []byte(authToken))
	if authToken == "" {
		return nil, pb.ErrorCode_ReLogin
	}

	token, err := auth.DecodeAuthToken([]byte(authToken))
	if err != nil || token == nil {
		return nil, pb.ErrorCode_ReLogin
	}

	session, err, errCode := s.GetUserSession(token.Uid)
	if session == nil {
		return nil, pb.ErrorCode_ReLogin
	} else if err != nil {
		return nil, errCode
	}

	return session, pb.ErrorCode_Success
}

func (s *GateServer) HttpLoginGame(pendingUser *PendingUser) ([]byte, *base.RpcError) {

	if pendingUser.session.Uaid == "" || pendingUser.session.PlayerId == 0 {
		var err error
		pendingUser.session.PlayerId, err = s.GetPlayerId(pendingUser.uid)
		if pendingUser.session.PlayerId == 0 && err != nil {
			return nil, &base.RpcError{Err: err, Code: int32(pb.ErrorCode_NotFoundPlayer)}
		}
		pendingUser.session.Uaid = s.UAID(pendingUser.uid, pendingUser.session.PlayerId)
	}

	now := time.Now()
	msg, err := s.UserInvoke(pendingUser.session.Uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PC2G_LoginGameReq),
		ReqIdx:  pendingUser.reqIdx,
		UserId:  pendingUser.session.Uid,
		RoleId:  pendingUser.session.PlayerId,
		UAID:    pendingUser.session.Uaid,
		Data:    pendingUser.data,
		ErrCode: 0,
		// Topic:   "",
		Topic: s.PrivateTopicID(),
	})

	if err != nil {
		return nil, &base.RpcError{Err: err, Code: int32(pb.ErrorCode_RpcInvokeError)}
	}

	messageID, data := msg.MsgId, msg.Data
	if messageID > 0 {
		if messageID == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
			return nil, &base.RpcError{Err: fmt.Errorf(string(msg.Data)), Code: msg.ErrCode}
		} else {
			if msg.RoleId > 0 && pendingUser.session.PlayerId != msg.RoleId {
				pendingUser.session.PlayerId = msg.RoleId
			}
			if len(msg.UAID) > 0 && strings.Compare(pendingUser.session.Uaid, msg.UAID) != 0 {
				pendingUser.session.Uaid = msg.UAID
			}
			s.SaveUserSession(pendingUser.session)
		}
		logger.Debug("GateServer:ExecuteLoginGame, UserInvoke End:", pendingUser.session.Uaid, pb.Protocols(messageID), messageID, len(data))
	}
	metrics.HistogramPut(metrics.EnterDelayHist, time.Since(now).Milliseconds(), metrics.Delay)
	return data, nil
}

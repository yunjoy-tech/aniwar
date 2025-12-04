package logic

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"strings"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"

	"github.com/dapr/go-sdk/service/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/auth"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"
	"gitlab.musadisca-games.com/wangxw/musae/framework/errorx"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/metrics"
	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"
)

func (s *GateServer) OnHttp(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("OnGate failed, err: ", err)
		}
	}()

	curTime := time.Now()
	upMsgLen := len(in.Data)

	if in == nil {
		return nil, fmt.Errorf("nil parameter")
	}
	//logger.Debugf("[GateServer] [LoginStep] OnLogin ContentType:%s, Verb:%s, QueryString:%s, len:%d", in.ContentType, in.Verb, in.QueryString, upMsgLen)

	if upMsgLen > baseconf.GetBaseConf().GateMsgMaxSize {
		return nil, fmt.Errorf("invalid error")
	}

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}
	// check session
	authToken := in.Request.Header.Get("auth-token")
	//authToken := in.Request.Header.Get("cookie")
	clientVersion := in.Request.Header.Get("client-version")
	channel := in.Request.Header.Get("channel")
	platform := in.Request.Header.Get("platform")
	deviceId := in.Request.Header.Get("device-id")
	//客户端版本验证
	if conf.GConf().Base.VersionCheck {
		logger.Infof("VersionCheck clientVersion:%s", clientVersion)
		err = s.VersionCheckExt(platform, clientVersion)
		if err != nil {
			logger.Warnf("VersionCheck error:%s", err.Error())
			out.Data = s.ErrorPack(cmd.ErrorCode_VersionLimit)
			return out, nil
		}
	}

	session, errCode := s.GetSession(authToken)
	if errCode != cmd.ErrorCode_Success || session == nil {
		out.Data = s.ErrorPack(errCode)
		return out, nil
	}
	logger.Debugf("clientVersion:%s, channel:%s, deviceId:%s", clientVersion, channel, deviceId)

	if err, errCode = s.CheckToken(session, authToken); err != nil {
		out.Data, err = s.Pack(cmd.Protocols_PS2C_ErrorCodeNtf, errCode, &cmd.S2C_ErrorCodeNtf{ErrorCode: uint32(errCode)}, "")
		if err != nil {
			out.Data = []byte(err.Error())
			logger.Warnf("PackWithBody err:%s", string(out.Data))
		}
		return out, err
	}

	// handle gate msg
	data, msgId, errCode := s.HandlerGate(in, session)
	if errCode != cmd.ErrorCode_Success || msgId == cmd.Protocols_Protocols_None {
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

func (s *GateServer) HandlerGate(in *common.InvocationEvent, session *cmd.UserSession) ([]byte, cmd.Protocols, cmd.ErrorCode) {
	//
	//if now > session.LimitTs {
	//	return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_TokenTimeout
	//}
	if conf.Base().IsDebug {
		logger.Debugf("user session:%+v", session)
	}

	data, e := tcpx.Decrypt(in.Data, session.CryptKey)
	if e != nil {
		logger.Warn(e)
		return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_Crypt
	}
	msgId, e := tcpx.MessageIDOf(data)
	if e != nil {
		logger.Warn(errorx.Wrap(e).Error())
		return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_UnKnownMsg
	}
	logger.Debug("OnMessage: ", in.Request.RemoteAddr, msgId, len(data))

	reqIdx, err := tcpx.ReqIndexOf(data)
	if err != nil {
		logger.Warn("OnNetMessage ReqIndexOf", errorx.Wrap(err).Error())
		return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_UnKnownMsg
	}
	//logger.Infof(" c.GetReqIndex() : %d", reqIdx)

	//防重放
	rspData, downMsgId := s.reqRepeated(nil, msgId, reqIdx, session)
	if rspData != nil && downMsgId != int32(cmd.Protocols_Protocols_None) {
		logger.Debugf("OnNetMessage ReqRepeated msgId:%d reqIdx:%d downMsgId:%d uid:%v",
			msgId, reqIdx, downMsgId, session.Uid)
		return rspData, cmd.Protocols(downMsgId), cmd.ErrorCode_Success
	}

	data, e = tcpx.BodyBytesOf(data)
	if e != nil {
		logger.Warn("OnNetMessage BodyBytesOf", e.Error())
		return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_UnKnownMsg
	}
	// 包体大小限制
	if len(data) > baseconf.GetBaseConf().GateMsgMaxSize {
		logger.Warn("OnNetMessage msg body too big")
		return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_MsgBodyLimit
	}

	var messageId cmd.Protocols
	var errCode = cmd.ErrorCode_Success
	switch cmd.Protocols(msgId) {
	/*case cmd.Protocols_PC2LS_RsaClientRandomReq:
	data, session.CryptKey, e = s.HandleRsa(nil, msgId, data)
	messageId = cmd.Protocols_PLS2C_RsaServerRandomRes*/
	case cmd.Protocols_PC2G_LoginGateReq:
		data, e = s.HandleLoginGate(nil, msgId, data, reqIdx)
		messageId = cmd.Protocols_PG2C_LoginGateRes
	case cmd.Protocols_PC2G_LoginGameReq:
		// 判断秘钥是否生成
		if baseconf.GetBaseConf().UseEncrypt == 1 && session.CryptKey == "" { // 默认秘钥为空
			logger.Warnf("服务器配置:%d, 秘钥没有重新生成, 当前秘钥:%s", baseconf.GetBaseConf().UseEncrypt, session.CryptKey)
			metrics.GaugeInc(metrics.EnterFailedCount)
			return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_ReLogin
		}

		var err *base.RpcError
		data, err = s.HandleLoginGame(nil, session, msgId, data, reqIdx)
		if err != nil {
			logger.Errorf("HandleLoginGame error, %v %v", in.Request.RemoteAddr, err.Error())
			metrics.GaugeInc(metrics.EnterFailedCount)
			return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode(err.Code)
		}
		metrics.GaugeInc(metrics.EnterSucceedCount)
		messageId = cmd.Protocols_PG2C_LoginGameRes
	default:
		// 判断秘钥是否生成
		if baseconf.GetBaseConf().UseEncrypt == 1 && session.CryptKey == "" { // 默认秘钥为空
			logger.Warnf("服务器配置:%d, 秘钥没有重新生成, 当前秘钥:%s", baseconf.GetBaseConf().UseEncrypt, session.CryptKey)
			metrics.GaugeInc(metrics.GateAuthFailCount)
			return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_ReLogin
		}
		user := NewUserExt(session, s)
		data, messageId, errCode = user.HandleClientMsg(msgId, data, reqIdx)
		if data != nil && errCode == cmd.ErrorCode_Success {
			session.LastRspData = &cmd.LastRspData{
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

func (s *GateServer) GetSession(authToken string) (*cmd.UserSession, cmd.ErrorCode) {
	//logger.Debugf("user token:%v", []byte(authToken))
	if authToken == "" {
		return nil, cmd.ErrorCode_ReLogin
	}

	token, err := auth.DecodeAuthToken([]byte(authToken))
	if err != nil || token == nil {
		return nil, cmd.ErrorCode_ReLogin
	}

	session, err, errCode := s.GetUserSession(token.Uid)
	if session == nil {
		return nil, cmd.ErrorCode_ReLogin
	} else if err != nil {
		return nil, errCode
	}

	return session, cmd.ErrorCode_Success
}

func (s *GateServer) HttpLoginGame(pendingUser *PendingUser) ([]byte, *base.RpcError) {

	if pendingUser.session.Uaid == "" || pendingUser.session.PlayerId == 0 {
		var err error
		pendingUser.session.PlayerId, err = s.GetPlayerId(pendingUser.uid)
		if pendingUser.session.PlayerId == 0 && err != nil {
			return nil, &base.RpcError{Err: err, Code: int32(cmd.ErrorCode_NotFoundPlayer)}
		}
		pendingUser.session.Uaid = s.UAID(pendingUser.uid, pendingUser.session.PlayerId)
	}

	now := time.Now()
	msg, err := s.UserInvoke(pendingUser.session.Uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(cmd.Protocols_PC2G_LoginGameReq),
		ReqIdx:  pendingUser.reqIdx,
		UserId:  pendingUser.session.Uid,
		RoleId:  pendingUser.session.PlayerId,
		UAID:    pendingUser.session.Uaid,
		Data:    pendingUser.data,
		ErrCode: 0,
		//Topic:   "",
		Topic: s.PrivateTopicID(),
	})

	if err != nil {
		return nil, &base.RpcError{Err: err, Code: int32(cmd.ErrorCode_RpcInvokeError)}
	}

	messageID, data := msg.MsgId, msg.Data
	if messageID > 0 {
		if messageID == int32(cmd.Protocols_PS2C_ErrorCodeNtf) {
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
		logger.Debug("GateServer:ExecuteLoginGame, UserInvoke End:", pendingUser.session.Uaid, cmd.Protocols(messageID), messageID, len(data))
	}
	metrics.HistogramPut(metrics.EnterDelayHist, time.Since(now).Milliseconds(), metrics.Delay)
	return data, nil
}

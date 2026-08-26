package logic

import (
	"context"
	"fmt"
	"github.com/yunjoy-tech/aniwar/src/common"
	"github.com/yunjoy-tech/aniwar/src/common/actor/stub"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/db"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/metrics"
	"github.com/yunjoy-tech/musae/tcpx"
	"google.golang.org/protobuf/proto"
	"time"
)

func (s *GateServer) OnTcp(c *tcpx.Context) {

	var session *pb.UserSession
	var err error
	accountId := c.GetAccountId()

	if accountId != "" {
		session, err, _ = s.GetUserSession(accountId)
		if err != nil {
			logger.Warn("OnNetMessage GetUserSession", fmt.Errorf(": %w", err).Error())
			return
		}
		user := s.userMgr.GetUser(accountId)
		if user == nil {
			logger.Warn("OnTcp GetUser", fmt.Errorf(": %w", err).Error())
			return
		}

		err = user.SetSession(session)
		if err != nil {
			err = fmt.Errorf("user.SetSession got error: %w", err)
			logger.Warn(err.Error())
			return
		}
	}

	// 判断协议秘钥
	if conf.Base().UseEncrypt == 1 && accountId != "" {
		// 赋值协议秘钥
		if c.GetEncryptKey() == "" {
			c.SetEncryptKey(session.CryptKey)
		}

		// 判断秘钥是否生成
		if c.GetEncryptKey() == "" {
			logger.Warnf("服务器配置:%d, 协议秘钥为nil", conf.Base().UseEncrypt)
			c.CloseConn()
			metrics.GaugeInc(metrics.GateAuthFailCount)
			return
		}
	}

	allData, e := tcpx.Decrypt(c.Stream, c.GetEncryptKey())
	if e != nil {
		logger.Warn(e)
		return
	}

	messageID, err := tcpx.MessageIDOf(allData)
	if err != nil {
		logger.Warn("OnNetMessage MessageIDOf", fmt.Errorf(": %w", err).Error())
		return
	}

	reqIdx, err := tcpx.ReqIndexOf(allData)
	if err != nil {
		logger.Warn("OnNetMessage ReqIndexOf", fmt.Errorf(": %w", err).Error())
		return
	}
	logger.Infof(" c.GetReqIndex() : %d", c.GetReqIndex())

	// 防重放
	rspData, downMsgId := s.reqRepeated(nil, messageID, reqIdx, session)
	if rspData != nil && downMsgId != int32(pb.Protocols_Protocols_None) {
		user := s.userMgr.GetUser(accountId)
		if user != nil {
			err = user.ReplyWithBody(downMsgId, reqIdx, pb.ErrorCode_Success, rspData)
			if err != nil {
				logger.Debugf("OnNetMessage ReqRepeated err:%v ", err)
				return
			}
			// 重复请求, 直接答复上次保存的数据
			metrics.GaugeInc(metrics.ReplayReqCount)
			return
		}
	}

	data, err := tcpx.BodyBytesOf(allData)
	if err != nil {
		logger.Warn("OnNetMessage BodyBytesOf", fmt.Errorf(": %w", err).Error())
		return
	}

	dataLen := len(data)
	// 包体大小限制
	if dataLen > conf.Base().GateMsgMaxSize {
		logger.Warn("OnNetMessage BodyBytesOf", fmt.Errorf(": %w", err).Error())

		// TODO 处理踢人操作
		return
	}

	metrics.GaugeInc(metrics.GateMsgCount)
	metrics.GaugeAdd(metrics.GateUpMsgSize, int64(dataLen))

	// ddos check
	if conf.DDos().TimeInterval > 0 && accountId != "" {
		user := s.userMgr.GetUser(accountId)
		if user.DDosCheck(uint32(dataLen)) {
			// TODO 处理踢人操作
			metrics.GaugeInc(metrics.DDosCount)
			return
		}
	}

	// gm命令处理
	handleMsgStatistics(messageID, int64(dataLen), true)
	logger.Debug("OnNetMessage: ", c.ClientIP(), c.Network(), pb.Protocols(messageID), len(data))

	switch pb.Protocols(messageID) {
	/*case pb.Protocols_PC2LS_RsaClientRandomReq:
	s.HandleRsa(c, messageID, data)*/
	case pb.Protocols_PC2G_LoginGateReq:
		_, err = s.HandleLoginGate(c, messageID, data, reqIdx)
		if err != nil {
			logger.Debugf("OnNetMessage Auth error, %v %v %v", c.ClientIP(), c.Network(), err)
			c.CloseConn()
			metrics.GaugeInc(metrics.GateAuthFailCount)
			return
		}
	case pb.Protocols_PC2G_LoginGameReq: // 快速登陆
		_, err = s.HandleLoginGame(c, nil, messageID, data, reqIdx)
		if err != nil {
			logger.Debugf("OnNetMessage Auth error, %v %v %v", c.ClientIP(), c.Network(), err)
			// TODO 返回错误码提示
			// s.HandleAuthFail(c)
			c.CloseConn()
			metrics.GaugeInc(metrics.GateAuthFailCount)
			return
		}

	default:
		user := s.userMgr.GetUser(accountId)
		if user != nil {
			block := make([]byte, dataLen)
			copy(block, data)
			user.PushClientMsg(&Msg{
				msgId:  messageID,
				reqIdx: reqIdx,
				Data:   block,
			})
		}
	}
}

// func (s *GateServer) Send2ClientErrorCode(uid string, errCode pb.ErrorCode) error {
//	user := s.userMgr.GetUser(uid)
//	if user == nil {
//		return nil
//	}
//
//	rsp := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(errCode)}
//	b, err := proto.Marshal(rsp)
//	if err != nil {
//		logger.Errorf(err.Error())
//		return err
//	}
//
//	err = user.ReplyWithBody(int32(pb.Protocols_PS2C_ErrorCodeNtf), 0, errCode, b)
//	if err != nil {
//		logger.Warn("GateServer:Send2ClientErrorCode got error, uid:%s, errCode:%+v", uid, errCode)
//		return err
//	}
//
//	return nil
// }

func (s *GateServer) Send2Client(msg *base.ProtoMsg) {
	var err error
	logger.Debugf("Send2Gate, msg:%+v", msg)
	// 判断是否是4开头，然后全服广播
	if common.IsBC(pb.Protocols(msg.MsgId)) {
		s.userMgr.BroadcastMsg(msg.MsgId, msg.AppId, msg.Data)
		return
	}
	for _, uid := range msg.Uids {
		user := s.userMgr.GetUser(uid)
		if user == nil {
			continue
		}
		if user.IsOnline() {
			err = user.ReplyWithBody(msg.MsgId, msg.ReqIdx, pb.ErrorCode(msg.ErrCode), msg.Data)
		} else {
			var vals []interface{}
			b, e := proto.Marshal(&pb.MsgData{Id: msg.MsgId, Data: msg.Data})
			if e != nil {
				continue
			}
			vals = append(vals, b)
			err = s.SAdd(context.Background(), db.KeyOfflineMsg(uid), int(conf.Base().HeartbeatTimout), vals...)
		}
		if err != nil {
			logger.Warnf("ReplyWithBody uid:%s, msg:%+v, err: %+v", uid, pb.Protocols(msg.MsgId), err)
		}
	}
}

func (s *GateServer) TcpLoginGame(pendingUser *PendingUser) ([]byte, *base.RpcError) {
	playerId, err := s.GetPlayerId(pendingUser.uid)
	if playerId == 0 && err != nil {
		return nil, &base.RpcError{Err: err, Code: int32(pb.ErrorCode_NotFoundPlayer)}
	}
	now := time.Now()
	uaid := s.UAID(pendingUser.uid, playerId)
	msg, err := s.UserInvoke(uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PC2G_LoginGameReq),
		ReqIdx:  pendingUser.reqIdx,
		UserId:  pendingUser.uid,
		RoleId:  playerId,
		UAID:    uaid,
		Data:    pendingUser.data,
		ErrCode: 0,
	})
	respMessageID, respData := msg.MsgId, msg.Data
	if respMessageID > 0 {
		if respMessageID == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
			rsp := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(msg.ErrCode), Param: []string{string(respData)}}
			b, err := proto.Marshal(rsp)
			if err != nil {
				logger.Warn("GateServer:ExecuteLoginGame, UserInvoke, reply error:", uaid, pb.Protocols(respMessageID), respMessageID, fmt.Errorf(": %w", err).Error())
			}
			err = pendingUser.ctx.ReplyWithBody(int32(pb.Protocols_PS2C_ErrorCodeNtf), msg.ErrCode, b)
			if err != nil {
				logger.Warn("GateServer:ExecuteLoginGame, UserInvoke, reply error:", uaid, pb.Protocols(respMessageID), respMessageID, fmt.Errorf(": %w", err).Error())
			}
			metrics.GaugeInc(metrics.EnterFailedCount)
		} else if msg.ErrCode == int32(pb.ErrorCode_RepeatMsg) {
			logger.Debugf("OnNetMessage, 防重放中获取到数据返回, :", stub.RoomActorType, pb.Protocols(respMessageID), respMessageID, len(respData))

			lastRspData, _ := s.reqRepeated(nil, pendingUser.msgId, pendingUser.reqIdx, pendingUser.session)
			err = pendingUser.ctx.ReplyWithBody(respMessageID, int32(pb.ErrorCode_Success), lastRspData)
			if err != nil {
				logger.Warn("GateServer:ExecuteLoginGame, UserInvoke, reply error:", uaid, pb.Protocols(respMessageID), respMessageID, fmt.Errorf(": %w", err).Error())
			}
		} else {
			// deprecated
			user := s.userMgr.AddUser(pendingUser.uid, playerId, pendingUser.ctx, pendingUser.session)
			err = user.ReplyWithBody(respMessageID, pendingUser.reqIdx, pb.ErrorCode_Success, respData)
			if err != nil {
				logger.Warn("GateServer:ExecuteLoginGame, UserInvoke ReplyWithBody err: ", fmt.Errorf(": %w", err).Error())
			}
			metrics.HistogramPut(metrics.EnterDelayHist, time.Since(now).Milliseconds(), metrics.Delay)
			metrics.GaugeInc(metrics.EnterSucceedCount)
		}
		logger.Debug("GateServer:ExecuteLoginGame, UserInvoke End:", uaid, pb.Protocols(respMessageID), respMessageID, len(respData))
	}
	return nil, nil
}

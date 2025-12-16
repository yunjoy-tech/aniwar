package logic

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/common/rsa"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/errorx"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/metrics"
	"gitee.com/aniwar2/musae/framework/tcpx"
	"google.golang.org/protobuf/proto"
)

// Deprecated: Use HandleLoginGame instead.
/*
func (s *GateServer) HandleAuth(c *tcpx.Context, messageID int32, data []byte) error {
	var req pb.C2DB_SessionAuthReq
	err := base.UnmarshalData(data, &req)
	if err != nil {
		return errors.New("proto.Unmarshal error")
	}

	accountId := req.AccountId
	logger.Info("GateServer:HandleAuth begin,", accountId, c.ClientIP())

	var table *state.KvTable
	table, err = s.GetGlobalRedis(db.KeyAccountToken(accountId), nil)
	if err != nil || table == nil {
		return fmt.Errorf("get account[%s] AccountToken error, err:%v", req.AccountId, err)
	}

	if string(table.Data) != strconv.FormatInt(int64(req.SessionId), 10) {
		return fmt.Errorf("account[%s] AccountToken error, err:%v, table.Data=%s, req.SessionId=%d", accountId, err, table.Data, req.SessionId)
	}

	// 判断队列是否达到排队上限
	//if s.pendingUserMgr.IsQueuingPossible() {
	if true {
		c.SetAccountId(accountId)
		pendingUser := &PendingUser{
			msg:     &req,
			ctx:     c,
			startTs: time.Now().Unix(),
		}
		s.pendingUserMgr.Push(pendingUser)
		return nil
	}

	return fmt.Errorf("peeding user num limit,now peeding num")
}
*/

// func (s *GateServer) ExecuteAuthLogic(req *PendingUser) error {
//
//	msg := req.msg.(*pb.C2DB_SessionAuthReq)
//	user := s.AddUser(msg.AccountId, req.ctx)
//	//
//	//// 通知拉起actor
//	//data, err := proto.Marshal(&pb.S2S_AccountAuthSuccess{})
//	//if err != nil {
//	//	logger.Error("AccountAuth Marshal failed", req.ctx.ClientIP(), req.ctx.Network())
//	//}
//	////notify user actor
//	//_, _ = user.UserInvoke(int32(pb.Protocols_PS2S_AccountAuthSuccess), data)
//
//	rsp := &pb.DB2C_SessionAuthRes{AccountId: msg.AccountId, SessionType: msg.SessionType, ErrorCode: uint32(0)}
//
//	//err = c.Reply(int32(pb.Protocols_PDB2C_SessionAuthRes), rsp)
//	b, err := proto.Marshal(rsp)
//	if err != nil {
//		return fmt.Errorf("OnNetMessage Reply, error:%v", errorx.Wrap(err,"").Error())
//	}
//	err = req.ctx.ReplyWithBody(int32(pb.Protocols_PDB2C_SessionAuthRes), b)
//	if err != nil {
//		return fmt.Errorf("OnNetMessage Reply, error:%v", errorx.Wrap(err,"").Error())
//	}
//
//	metrics.GaugeInc(metrics.GateAuthCount)
//	logger.Info("AccountAuth Success", req.ctx.ClientIP(), req.ctx.Network(), user.String())
//	return nil
// }

// func (s *GateServer) HandleAuthFail(ctx *tcpx.Context) error {
//
//
//	rsp := &pb.DB2C_SessionAuthRes{ErrorCode: uint32(pb.ErrorCode_InternalError)}
//
//	//err = c.Reply(int32(pb.Protocols_PDB2C_SessionAuthRes), rsp)
//	b, err := proto.Marshal(rsp)
//	if err != nil {
//		return fmt.Errorf("OnNetMessage Reply, error:%v", errorx.Wrap(err,"").Error())
//	}
//	err = ctx.ReplyWithBody(int32(pb.Protocols_PDB2C_SessionAuthRes), int32(pb.ErrorCode_Success), b)
//	if err != nil {
//		return fmt.Errorf("OnNetMessage Reply, error:%v", errorx.Wrap(err,"").Error())
//	}
//
//	metrics.GaugeInc(metrics.GateAuthFailCount)
//	logger.Info("AccountAuth Fail")
//	return nil
// }

// 在线玩家个人消息
func (s *GateServer) privateMsg(msg *base.ProtoMsg) error {
	uid, messageID, data := msg.UserId, msg.MsgId, msg.Data

	if messageID > 0 {
		user := s.userMgr.GetUser(uid)
		if user == nil {
			return fmt.Errorf("user %v invalid ", uid)
		}
		err := user.ReplyWithBody(messageID, msg.ReqIdx, pb.ErrorCode_Success, data)
		if err != nil {
			return fmt.Errorf("privateMsg ReplyWithBody %v", err)
		}

		// 如果是踢人通知
		switch pb.Protocols(messageID) {
		case pb.Protocols_PGWS2C_KickOutNtf:
			s.userMgr.Logout(uid, "kickout")
			if err != nil {
				return err
			}
			logger.Debug("玩家被踢下线: ", uid, string(data))
		}

	} else {
		return fmt.Errorf("privateMsg invalid msgId")
	}
	logger.Debugf("privateMsg, MsgId: %v AppId:%v len: %v", msg.MsgId, msg.AppId, len(msg.Data))
	return nil
}

// 在线玩家广播消息
func (s *GateServer) broadcastMsg(msg *base.ProtoMsg) error {
	messageID, appid, data := msg.MsgId, msg.AppId, msg.Data
	if messageID > 0 {
		s.userMgr.BroadcastMsg(messageID, appid, data)
	} else {
		return fmt.Errorf("[broadcastMsg] msg, %v %v", messageID, appid)
	}
	logger.Debugf("broadcastMsg, MsgId: %s AppId:%s len: %d", msg.MsgId, msg.AppId, len(msg.Data))
	return nil
}

// RegisterDeprecatedMsg 注册废弃消息
func (s *GateServer) RegisterDeprecatedMsg() {
	// 初始化
	DeprecatedMsgId = sync.Map{}

	// 静态配置
	// ids := excel.GetConfigMgr().GetCfg().DEPRECATED_MSG_ID
	// for _, id := range ids {
	//	DeprecatedMsgId.Store(id, id)
	//	logger.Infof("Register Deprecated Msg %d", id)
	// }

	// 动态配置
	data, err := s.GetFromConfigCenter(db.KeyCfgGlobalDeprecatedMsg)
	if err != nil {
		return
	}
	if data != "" {
		idList := strings.Split(data, "|")
		for _, v := range idList {
			id, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			DeprecatedMsgId.Store(int32(id), id)
			logger.Infof("Register Deprecated Msg %d", id)
		}
	}
}

// 跑马灯推送
func (s *GateServer) HandlePushRollingNotice(content string) {
	// 构建消息
	ntf := &pb.LS2C_NotifyMessage{
		ChannelId: pb.ChatChannel_Channel_system,
		Message: []*pb.BroadMessage{{
			MType:      pb.MessageType_Message_Type_ServerMsg,
			FromRoleId: 0,
			Data:       []string{content},
			TimeStamp:  time.Now().Unix(),
		}},
	}
	data, err := proto.Marshal(ntf)
	if err != nil {
		logger.Error(err)
		return
	}
	msg := &base.ProtoMsg{
		MsgId:  int32(pb.Protocols_PLS2C_NotifyMessage),
		UserId: "",
		RoleId: 0,
		UAID:   "",
		Data:   data,
		AppId:  s.AppId,
		Uids:   nil,
	}

	// 广播
	s.Send2Client(msg)
}

func (s *GateServer) PushSrvMsg(msg *base.ProtoMsg) (bool, error) {
	select {
	case s.ch <- msg:
	default:
		metrics.GaugeInc(metrics.DropDownMsgCount)
		logger.Debugf("GateServer msg chan full, drop msg: %d", msg.MsgId)
		return true, errors.New("msg chan full")
	}
	return false, nil
}

func (s *GateServer) HandlerSrvMsg() {
	var err error
	for {
		msg := <-s.ch
		if IsBroadcastCmd(msg.MsgId) {
			err = s.broadcastMsg(msg)
		} else {
			err = s.privateMsg(msg)
		}
		if err != nil {
			logger.Warn("HandlerSrvMsg process msg error,err: ", err)
		}
		logger.Debug("HandlerSrvMsg:", msg.String())
	}
}

func (s *GateServer) HandleRsa(c *tcpx.Context, messageID int32, data []byte) ([]byte, string, error) {
	var (
		err error
		// cliKey string
		req pb.C2LS_RsaClientRandomReq
	)
	err = base.UnmarshalData(data, &req)
	if err != nil {
		logger.Warn(errorx.Wrap(err, "").Error())
		return nil, "", err
	}
	logger.Debugf("HandleRsa req: %v", &req)

	_, baseStr, rsaKey := rsa.CreateSrvRsaKey(c, req.CliRandomSeed)

	res := &pb.LS2C_RsaServerRandomRes{
		SrvRandomSeed: baseStr,
	}
	// logger.Debugf("===>>> RSA\n客户端随机码:%s, 密文:%s\n 服务器随机码:%s, 密文:%s",
	//	cliKey, req.CliRandomSeed, srvKey, res.SrvRandomSeed)
	//
	// // 最终密码规则: MD5(客户端随机码+服务器随机码)
	// md5Text := tls.RsaVal(cliKey, srvKey)

	// md5Val := md5.Sum([]byte(md5Text))
	// secretKey := fmt.Sprintf("%X", md5Val)
	// logger.Debug("HandleRsa md5Key: ", secretKey)
	b, err := proto.Marshal(res)
	if err != nil {
		logger.Debugf(err.Error())
		return nil, "", fmt.Errorf("OnNetMessage Reply proto.Marshal, error:%v", errorx.Wrap(err, "").Error())
	}
	logger.Debugf("OnNetMessage HandleRsa %v %v %v", pb.Protocols(messageID), &req, res)
	if c != nil {

		err = c.ReplyWithBody(int32(pb.Protocols_PLS2C_RsaServerRandomRes), int32(pb.ErrorCode_Success), b)
		if err != nil {
			logger.Debugf(err.Error())
			return nil, "", fmt.Errorf("OnNetMessage Reply ReplyWithBody, error:%v", errorx.Wrap(err, "").Error())
		}

		defer func() {
			// 该接口返回不做对称加密返回给客户端数据
			logger.Debug("最终秘钥 HandleRsa md5Key: ", rsaKey)
			c.SetEncryptKey(rsaKey)
		}()
	} else {
		return b, rsaKey, err
	}
	return nil, "", nil
}

func (s *GateServer) HandleLoginGate(c *tcpx.Context, messageID int32, data []byte, reqIdx uint32) ([]byte, error) {
	var (
		err error
		// cliKey string
		req pb.C2G_LoginGateReq
	)
	err = base.UnmarshalData(data, &req)
	if err != nil {
		logger.Warn(errorx.Wrap(err, "").Error())
		return nil, err
	}
	logger.Debugf("LoginGateReq: %v", &req)

	// 校验token有效性
	session, errCode := s.GetSession(req.Token)
	if errCode != pb.ErrorCode_Success || session == nil {
		return nil, fmt.Errorf("HandleLoginGate GetSession, errCode:%v", errCode)
	}
	// token, err := s.GetToken(session.Uid)
	// if err != nil {
	//	logger.Errorf("GetToken err:%+v", err)
	// }
	// if token == "" || token != req.Token {
	//	return nil, fmt.Errorf("HandleLoginGate GetSession, errCode:%v", pb.ErrorCode_TokenInvalid)
	// }
	if err, errCode = s.CheckToken(session, req.Token); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("HandleLoginGate CheckToken, errCode:%v", errCode))
		return nil, err
	}

	res := &pb.G2C_LoginGateRes{
		Err_Code: int32(pb.ErrorCode_Success),
	}
	b, err := proto.Marshal(res)
	if err != nil {
		logger.Debugf(err.Error())
		return nil, fmt.Errorf("reply proto.Marshal, error:%v", errorx.Wrap(err, "").Error())
	}
	logger.Debugf("HandleLoginGate %v %v %v", pb.Protocols(messageID), &req, res)

	// 绑定链接
	user := s.userMgr.AddUser(session.Uid, session.PlayerId, c, session)

	s.SaveHeartBeat(session.Uid, s.PrivateTopicID())

	// 通知userActor, 广播消息到所有actors, 更新topic
	err = s.NotifyActorGateTopic(session.Uid, session.Uaid, pb.GateTopicOperator_GTO_bind)
	if err != nil {
		logger.Errorf(err.Error())
	}

	if c != nil {
		c.SetAccountId(req.AccountId)
		err = user.ReplyWithBody(int32(pb.Protocols_PG2C_LoginGateRes), 0, pb.ErrorCode_Success, b)
		if err != nil {
			logger.Debugf(err.Error())
			return nil, fmt.Errorf("reply ReplyWithBody, error:%v", errorx.Wrap(err, "").Error())
		}
		if account, err := s.GetAccount(db.KeyAccountInfo(req.AccountId)); err == nil {
			taptap.TcpEnterComm(req.AccountId, account.TapUserInfo, account.CliDeviceInfo)
		}
	} else {
		return b, err
	}
	return nil, nil
}

package logic

import (
	"context"
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"github.com/pkg/errors"
	myCommon "github.com/yunjoy-tech/aniwar/src/common"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/datalog/taptap"
	comn "github.com/yunjoy-tech/aniwar/src/common/server"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/gamelib/guid"
	"github.com/yunjoy-tech/musae/global"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/metrics"
	svc "github.com/yunjoy-tech/musae/service"
	"github.com/yunjoy-tech/musae/tcpx"
	"github.com/yunjoy-tech/musae/utils"
	"google.golang.org/protobuf/proto"
	"os"
	"sync"
	"time"
)

type GateServer struct {
	comn.Server
	userMgr *UserMgr
	// pendingUserMgr *PendingUserMgr
	ch chan *base.ProtoMsg
}

// todo 暂时可以废弃，先做功能开启控制，后面再看需求是否需要增加协议Id控制
var DeprecatedMsgId sync.Map // 废弃消息id

func NewGateServer() base.IServer {
	srv := &GateServer{}
	srv.HasPriTopic = true // 开启私有频道订阅
	srv.OnPreInit = srv.OnPreInitHandler
	srv.OnPostInit = srv.OnPostInitHandler
	srv.OnNetConnect = srv.OnNetConnectHandler
	srv.OnNetMessage = srv.OnNetMessageHandler
	srv.OnNetClose = srv.OnNetCloseHandler
	srv.OnDaprTopicEvent = srv.OnDaprTopicEventHandler
	srv.OnDaprSvcInvoke = srv.OnDaprSvcInvokeHandler
	srv.OnDaprBindInvoke = srv.OnDaprBindInvokeHandler
	srv.OnRegisterMetric = srv.RegisterMetrics
	srv.OnCfgCenterCB = srv.HandlerConfEvent
	srv.userMgr = NewUserMgr(srv)
	// srv.pendingUserMgr = NewPendingUserMgr(srv)
	return srv
}

func (s *GateServer) RegisterMetrics() {
	metrics.RegisterGauge(metrics.UserConn, false)
	metrics.RegisterGauge(metrics.GateConnCount, true)

	metrics.RegisterGauge(metrics.EnterSucceedCount, false)
	metrics.RegisterGauge(metrics.EnterFailedCount, false)
	metrics.RegisterGauge(metrics.EnterDropCount, false)

	metrics.RegisterGauge(metrics.GateAuthCount, false)
	metrics.RegisterGauge(metrics.GateAuthFailCount, false)

	metrics.RegisterGauge(metrics.PendingUserCount, false)
	metrics.RegisterGauge(metrics.QueueUserCount, false)
	metrics.RegisterGauge(metrics.GateMsgCount, true)
	metrics.RegisterGauge(metrics.GateUpMsgSize, true)
	metrics.RegisterGauge(metrics.GateDownMsgSize, true)
	metrics.RegisterGauge(metrics.DropUpMsgCount, false)
	metrics.RegisterGauge(metrics.DropDownMsgCount, false)
	metrics.RegisterGauge(metrics.DDosCount, false)
	metrics.RegisterGauge(metrics.ReplayReqCount, false)

	metrics.RegisterHistogram(metrics.GateDelayHist, nil, metrics.Delay)
	metrics.RegisterHistogram(metrics.EnterDelayHist, nil, metrics.Delay)
}

func (s *GateServer) OnPreInitHandler() error {
	return nil
}

func (s *GateServer) OnPostInitHandler() error {

	// 注册login接口
	s.RegisterDaprSvcInvokeHandler("/api", s.OnHttp)

	interval := time.Duration(conf.Base().HeartbeatInterval)
	// 定时器模块

	s.AddTimer(true, time.Second*interval, s.userMgr.HeartbeatCheck)
	// 每分钟定时器
	s.AddTimer(true, time.Minute, s.userMgr.ReportDataMinute)

	s.RegisterDeprecatedMsg()

	// 消息包异步处理
	s.ch = make(chan *base.ProtoMsg, 10000)

	// 排队处理
	/*for i := int32(0); i < conf.Base()Conf().GateLoginThreadNum; i++ {
		threading.GoSafe(s.pendingUserMgr.Execute)
	}*/
	utils.GoSafeRun(s.HandlerSrvMsg, nil)
	// threading.GoSafe(func() {
	//	t := time.NewTicker(time.Second * 1)
	//	defer t.Stop()
	//	for {
	//		select {
	//		case <-t.C:
	//			utils.SafeRunNoError(s.pendingUserMgr.GrantLoginToken)
	//		}
	//	}
	// })

	s.LiveTime = time.Now().Unix() // 创建server时间戳
	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "gateserver")

	return nil
}

// func (s *GateServer) GetUser(userId string, roleId uint64, c *tcpx.Context) *User {
//	user := s.userMgr.GetUser(userId)
//	if user == nil || user.roleId != roleId {
//		return s.userMgr.AddUser(userId, roleId, c)
//	}
//	user.ctx = c
//	return user
// }

func (s *GateServer) OnNetConnectHandler(c *tcpx.Context) {
	if conf.Base().IsDebug {
		logger.Debug("OnNetConnect from remote host: ", c.ClientIP(), c.Network())
	}
	metrics.GaugeInc(metrics.GateConnCount)
}

func (s *GateServer) OnNetMessageHandler(c *tcpx.Context) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnNetMessage recover, err: ", err)
		}
	}()

	// logger.Debug("OnNetMessage from remote host: ", c.ClientIP(), c.Network())
	s.OnTcp(c)
}

func (s *GateServer) OnNetCloseHandler(c *tcpx.Context) {
	if c.GetAccountId() != "" {
		logger.Infof("[gate]OnNetClose : %s, %s, %s.", c.ClientIP(), c.Network(), c.GetAccountId())
	}
	accountId, ok := c.AccountId()
	if ok && accountId != "" {
		err := s.CleanHeartBeat(accountId)
		if err != nil {
			err = errors.Wrap(err, "长链接断开, 删除心跳报错")
			logger.Error(err.Error())
		}

		s.userMgr.Logout(accountId, "close")
	}
	metrics.GaugeDec(metrics.GateConnCount)
}

// // KnockOff 被踢下线
// func (s *GateServer) KnockOff(uid string) {
//	if isOnline, session := s.PlayerIsOnline(uid); isOnline {
//		if session != nil {
//			heartBeat := s.GetHeartBeat(session.Uaid)
//
//			if heartBeat != nil {
//				ret := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(pb.ErrorCode_KnockedOff)}
//				logger.Debugf("顶号, 踢人下线, %+v", ret)
//
//				err := s.Send2ClientErrorCode(uid, pb.ErrorCode_KnockedOff)
//				if err != nil {
//					logger.Errorf("GateServer KnockOff got err:%+v", err)
//				}
//			}
//		}
//	}
// }

func (s *GateServer) OnDaprTopicEventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("recover, err: ", err)
		}
	}()

	if e == nil {
		return false, fmt.Errorf("nil topic event")
	}

	msg, err := base.UnPackProtoMsg(e.RawData)
	if err != nil {
		logger.Debugf("UnPackProtoMsg, error: %+v", err)
		return false, err
	}
	metrics.GaugeInc(metrics.MsgSubCount)
	handleMsgStatistics(msg.MsgId, int64(len(msg.Data)), false)
	logger.Debugf("PubSubName:%s Topic:%s ID:%s DataLen:%v Msg:%s", e.PubsubName, e.Topic, e.ID, len(e.RawData), msg.Str())

	if msg.MsgId == int32(pb.Protocols_PS2C_ErrorCodeNtf) {
		errCode := &pb.S2C_ErrorCodeNtf{}
		err = proto.Unmarshal(msg.Data, errCode)
		if err != nil {
			logger.Debugf("Unmarshal, error: %+v", err)
			return false, err
		}

		msg.ErrCode = int32(errCode.ErrorCode)
	}

	err = s.HandlerSubEvent(msg)
	if err != nil {
		logger.Debugf("HandlerSubEvent  error:%+v", err)
	}
	return false, nil
}

func (s *GateServer) OnDaprSvcInvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnDaprSvcInvokeHandler recover, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
		return nil, err
	}

	metrics.GaugeInc(metrics.InvokeSubCount)
	out = &common.Content{
		Data:        []byte{byte(1)},
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	msg, err := base.UnPackProtoMsg(in.Data)
	if err != nil {
		logger.Warn("OnDaprSvcInvokeHandler UnPackProtoMsg, err: ", err)
		return nil, err
	}
	// messageID, uid, data := msg.MsgId, msg.UserId, msg.Data
	logger.Debugf("Gate OnDaprSvcInvokeHandler: msgId:%v, %s", pb.Protocols(msg.MsgId), msg.String())

	msg.Topic = string(svc.EVENT_PRIVATE)
	s.PushSrvMsg(msg)
	return out, nil
}

func (s *GateServer) OnDaprBindInvokeHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnDaprBindInvokeHandler recover, err: ", err)
		}
	}()

	logger.Debugf("Binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *GateServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("OnHeartBeat", c.ClientIP())
}

func (s *GateServer) Exit() {

	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "gateserver", time.Now().Unix()-s.LiveTime)
	s.printMsgStatistics()
}

func (s *GateServer) Reload() error {
	if err := s.LoadConf(); err != nil {
		logger.Warn("gate server reload err:", err)
		return err
	}

	s.RegisterDeprecatedMsg()

	logger.Info("gate server reload success")
	return nil
}

func (s *GateServer) Main() {
	for {
		select {
		case timeEvt := <-s.TimeCh:
			s.OnTimerEventCB(timeEvt)
		case <-s.ExitCh:
			if s.Daprc != nil {
				s.Daprc.Close()
				s.Daprc.Shutdown(context.Background())
			}
			logger.Infof("server %s exit success", global.AppID)
			logger.Flush()
			os.Exit(0)
		}
	}
}

// 防重放
func (s *GateServer) reqRepeated(c *tcpx.Context, messageID int32, reqIdx uint32, session *pb.UserSession) ([]byte, int32) {
	if conf.Base() == nil || conf.Base().UseReqIdx == 0 {
		// 未开启
		return nil, int32(pb.Protocols_Protocols_None)
	}
	// if c != nil {
	//	if user := s.userMgr.GetUser(c.GetAccountId()); user != nil {
	//
	//		//lastRsp := user.GetLastRsp()
	//		//logger.Debugf(fmt.Sprintf("上次保存的下发数据lastRsp: %+v, 本次请求的编号 reqIdx=%d", lastRsp, reqIdx))
	//		//
	//		//if lastRsp != nil && reqIdx == lastRsp.ReqIdx && messageID == int32(lastRsp.Up) {
	//		//	logger.Debugf("重复请求, 直接答复上次保存的数据, 本次请求编号reqIdx=%d, 保存的上次接口数据:%v", reqIdx, user.lastRspData)
	//		//	return lastRsp.RspData, int32(lastRsp.Down)
	//		//}
	//
	//	} else {
	//		logger.Warnf("OnNetMessage GetUser got nil")
	//	}
	// } else

	if session != nil {
		if session.LastRspData != nil && session.LastRspData.ReqIdx == reqIdx && messageID == session.LastRspData.UpCmd {
			return session.LastRspData.RspData, session.LastRspData.DownCmd
		}
	}

	return nil, int32(pb.Protocols_Protocols_None)
}

func (s *GateServer) setLastRespData(uid string, downMsgId int32, reqIdx uint32, body []byte) error {
	if !myCommon.IsDown(pb.Protocols(downMsgId)) {
		return nil
	}

	err, upMsgId := myCommon.DownNumber2UpNumber(downMsgId)
	if err != nil {
		return err
	}

	// 保存最后次接口回包数据
	session, err, _ := s.GetUserSession(uid)
	if err != nil {
		return err
	}
	session.LastRspData = &pb.LastRspData{
		ReqIdx:  reqIdx,
		UpCmd:   upMsgId,
		DownCmd: downMsgId,
		RspData: body,
	}
	err = s.SaveUserSession(session)
	if err != nil {
		return err
	}

	return nil
}

// NotifyActorGateTopic 通知userActor, 广播gate的topic
func (s *GateServer) NotifyActorGateTopic(uid string, uaid string, operator pb.GateTopicOperator) error {
	gateTopicData, err := proto.Marshal(&pb.S2S_TcpGateTopicReq1{
		Opt:    operator,
		Uid:    uid,
		GateId: s.PrivateTopicID(),
	})
	if err != nil {
		return err
	}

	_, playerId := s.ConvUAID(uaid)
	_, err = s.UserInvoke(uaid, &base.ProtoMsg{
		AppId:        s.AppId,
		MsgId:        int32(pb.Protocols_PS2S_TcpGateTopicReq1),
		ServerReqIdx: guid.GenIntUuid(),
		UserId:       uid,
		RoleId:       playerId,
		UAID:         uaid,
		Data:         gateTopicData,
		ErrCode:      0,
	})
	if err != nil {
		return err
	}

	return nil
}

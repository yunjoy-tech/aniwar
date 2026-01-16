package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/aniwar/src/common/db"
	comn "gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"gitee.com/aniwar2/musae/utils"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
	"time"
)

type Msg struct {
	msgId    int32
	Data     []byte
	ctx      *tcpx.Context
	ClientIp string
}

func (m *Msg) String() string {
	return fmt.Sprintf("msgId:%v, ip:%s", m.msgId, m.ctx.ClientIP())
}

type LoginServer struct {
	comn.Server
	ch           chan *Msg
	ticket       chan struct{}
	ticketAddSum int
	ticketDecSum int
}

func NewLoginServer() base.IServer {
	srv := &LoginServer{}
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
	return srv
}

func (s *LoginServer) RegisterMetrics() {
	metrics.RegisterGauge(metrics.LoginSucceedCount, false)
	metrics.RegisterGauge(metrics.LoginFailedCount, false)
	metrics.RegisterHistogram(metrics.LoginDelayHist, nil, metrics.Delay) // todo

}

func (s *LoginServer) OnPreInitHandler() error {
	return nil
}

func (s *LoginServer) OnPostInitHandler() error {
	// 注册login接口
	s.RegisterDaprSvcInvokeHandler("/api", s.OnHttp)
	//
	if conf.Login().LoginReqRate > 0 && conf.Login().LoginReqQueue > 0 {
		s.ch = make(chan *Msg, conf.Login().LoginReqQueue)
		utils.GoSafeRun(s.doHandleMsg, nil)
		s.ticket = make(chan struct{}, conf.Login().LoginReqRate)
		utils.GoSafeRun(func() {
			t := time.NewTicker(time.Second)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					utils.SafeRunNoError(s.GrantGateTicket)
				}
			}
		}, nil)
	}

	s.LiveTime = time.Now().Unix() // 创建server时间戳

	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "loginserver")

	return nil
}

func (s *LoginServer) OnNetConnectHandler(c *tcpx.Context) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnNetConnect failed, err: ", err)
		}
	}()

	logger.Debug("OnNetConnect from remote host:", c.ClientIP(), c.Network())
}

func (s *LoginServer) OnNetMessageHandler(c *tcpx.Context) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnNetMessage failed, err: ", err)
		}
	}()

	s.OnTcp(c)
}

func (s *LoginServer) OnNetCloseHandler(c *tcpx.Context) {

	logger.Debug("OnNetClose from remote host: ", c.ClientIP(), c.Network())
}

func (s *LoginServer) OnDaprTopicEventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnDaprTopicEvent failed, err: ", err)
		}
	}()

	msg, err := base.UnPackProtoMsg(e.RawData)
	if err != nil {
		logger.Debugf("UnPackProtoMsg, error: %+v", err)
		return false, err
	}
	logger.Debugf("event - PubsubName:%s, Topic:%s, ID:%s, Data: %s", e.PubsubName, e.Topic, e.ID, e.Data)

	err = s.HandlerSubEvent(msg)
	if err != nil {
		logger.Debugf("HandlerSubEvent  error:%+v", err)
	}
	return false, nil
}

func (s *LoginServer) OnDaprSvcInvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnDaprSvcInvokeHandler failed, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
		return
	}
	logger.Debugf("OnDaprSvcInvokeHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))
	metrics.GaugeInc(metrics.InvokeSubCount)
	out = &common.Content{
		Data:        in.Data,
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	req := &pb.RpcCallReq{}
	if err := json.Unmarshal(in.Data, req); err != nil {
		logger.Debug("C2SMsg - Unmarshal error")
	}
	logger.Debug("RpcCallReq : %+v", req)

	return out, nil
}

func (s *LoginServer) OnDaprBindInvokeHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	logger.Debug("binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *LoginServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("OnHeartBeat", c.ClientIP())
}

func (s *LoginServer) Exit() {

	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "loginserver", time.Now().Unix()-s.LiveTime)

	logger.Info("Server Exit", s.AppId)
}

func (s *LoginServer) Reload() error {
	s.LoadConf()
	return nil
}

func (s *LoginServer) Main() {
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

func (s *LoginServer) handleKickOut(uid string) error {
	ntf := &pb.S2S_KickoutPlayerNtf{
		Reason: "account_multi_login",
	}
	data, err := proto.Marshal(ntf)
	if err != nil {
		logger.Debug("proto.Marshal error:", err)
		return err
	}

	// 调用userInvoke
	in := &base.ProtoMsg{}
	in.MsgId = int32(pb.Protocols_PS2S_KickoutPlayerNtf)
	in.Data = data
	in.UserId = uid
	in.AppId = s.AppId
	// in.GUID = utils.GenIntUUID()
	in.ReqIdx = 0
	userStub := stub.NewUserStub(uid)
	s.ImpActorStub(userStub)
	_, err = userStub.UserInvoke(context.Background(), in)
	if err != nil {
		logger.Warn("login invoke actor got err:", err)
		return err
	}

	logger.Debug("handleKickOut: ", uid)
	return nil
}

func (s *LoginServer) checkAccountBanned(accountId string) int64 {
	// 查询db中的数据
	kvTable, err := s.GetMongoAccount(db.KeyAccountInfo(accountId), nil)
	if err != nil {
		return 0
	}

	if kvTable == nil {
		return 0
	}
	account := &pb.UserData{}
	err = proto.Unmarshal(kvTable.Data, account)
	if err != nil {
		logger.Warn("proto unmarshal err: ", err)
		return 0
	}

	if account.Account == nil {
		return 0
	}
	return account.Account.BannedTs
}

func (s *LoginServer) checkWhitelist(accountId string) bool {
	if str, err := s.GetConfigKeyForStr(db.KeyCfgWhiteList); err == nil {
		ids := strings.Split(str, ",")
		for _, k := range ids {
			if len(k) > 0 && strings.Compare(k, accountId) == 0 {
				return true
			}
		}
	}

	return false
}

// checkBlacklist 验证是否在黑名单
func (s *LoginServer) checkBlacklist(accountId string) bool {
	if str, err := s.GetConfigKeyForStr(db.KeyCfgBlackList); err == nil {
		ids := strings.Split(str, ",")
		for _, k := range ids {
			if len(k) > 0 && strings.Compare(k, accountId) == 0 {
				return true
			}
		}
	}
	return false
}

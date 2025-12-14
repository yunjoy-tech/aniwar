package logic

import (
	"context"
	"errors"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/metrics"
	"gitee.com/aniwar2/musae/framework/tcpx"
	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"
	comn "gitee.com/bychannel/aniwar/src/common/server"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"github.com/dapr/go-sdk/service/common"
	"os"
	"time"
)

type GuideServer struct {
	comn.Server
}

func NewGuideServer() base.IServer {
	srv := &GuideServer{}
	srv.AppId = "guide"
	srv.InAddr = ":20001"
	srv.GRPCPort = "50001"
	srv.HasPriTopic = true // 开启私有频道订阅
	srv.OnPreInit = srv.PreInit
	srv.OnServerInit = srv.ServerInit
	srv.OnEventHandler = srv.EventHandler
	srv.OnInvokeHandler = srv.InvokeHandler
	srv.OnBindHandler = srv.BindingHandler
	srv.OnRegisterMetric = srv.RegisterMetrics
	srv.OnCfgCenterCB = srv.HandlerConfEvent
	return srv
}

func (s *GuideServer) RegisterMetrics() {
	metrics.RegisterGauge(metrics.GuideSucceedCount, false)
	metrics.RegisterGauge(metrics.GuideFailedCount, false)
	metrics.RegisterHistogram(metrics.GuideDelayHist, nil, metrics.Delay)
}

func (s *GuideServer) PreInit() error {
	// client version
	s.RegisterRpcHandler("/api/version", s.Version)
	s.RegisterRpcHandler("/api/notice", s.Notice)

	// test
	// s.RegisterBindingInvocationHandler("/api/_test", s.test)
	return nil
}

func (s *GuideServer) ServerInit() error {

	s.LiveTime = time.Now().Unix() // 创建server时间戳
	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "GuideServer")
	return nil
}

func (s *GuideServer) EventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("EventHandler failed, err: ", err)
		}
	}()

	msg, err := base.UnPackProtoMsg(e.RawData)
	if err != nil {
		logger.Debugf("UnPackProtoMsg, error: %+v", err)
		return false, err
	}
	logger.Debug("event - PubsubName:%s, Topic:%s, ID:%s, DataLen: %v", e.PubsubName, e.Topic, e.ID, len(e.RawData))

	err = s.HandlerSubEvent(msg)
	if err != nil {
		logger.Debugf("HandlerSubEvent  error:%+v", err)
	}
	return false, nil
}

func (s *GuideServer) InvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("InvokeHandler failed, err: ", err)
		}
	}()

	if in == nil {
		return nil, errors.New("nil invocation parameter")
	}
	metrics.GaugeInc(metrics.InvokeSubCount)

	msg, err := base.UnPackProtoMsg(in.Data)
	if err != nil {
		logger.Warn("InvokeHandler UnPackProtoMsg, err: ", err)
		return nil, err
	}

	messageID, uid := msg.MsgId, msg.UserId
	logger.Debug("guide.server ===>>> ", uid, pb.Protocols(messageID), messageID)
	logger.Debug("InvokeHandler: ", in.ContentType, in.Verb, in.QueryString, pb.Protocols(messageID), msg.String())

	return out, nil
}

func (s *GuideServer) BindingHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	logger.Debug("binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *GuideServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("OnHeartBeat", c.ClientIP())
}

func (s *GuideServer) OnNetConnect(c *tcpx.Context) {
	logger.Debug("GuideServer:OnNetMessage, implement me")
}

func (s *GuideServer) OnNetMessage(c *tcpx.Context) {
	logger.Debug("GuideServer:OnNetMessage, implement me")
}

func (s *GuideServer) OnNetClose(c *tcpx.Context) {
	logger.Debug("GuideServer:OnNetMessage, implement me")
}

func (s *GuideServer) Exit() {

	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "GuideServer", time.Now().Unix()-s.LiveTime)
	logger.Info("Server Exit", s.AppId)
}

func (s *GuideServer) Reload() error {
	logger.Info("GuideServer Reload ===>>>")
	err := s.LoadConf()
	if err != nil {
		logger.Errorf("reload --> LoadConf got err:%+v", err)
		return err
	}
	return nil
}

func (s *GuideServer) Main() {
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

package logic

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	comn "gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/musae/global"
	"os"
	"time"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"github.com/dapr/go-sdk/service/common"
	_ "google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
)

type LobbyServer struct {
	comn.Server
}

func NewLobbyServer() base.IServer {
	srv := &LobbyServer{}
	srv.AppId = "lobby"
	srv.InAddr = ":23001"
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

func (s *LobbyServer) RegisterMetrics() {

}

func (s *LobbyServer) PreInit() error {
	return nil
}

func (s *LobbyServer) ServerInit() error {

	s.LiveTime = time.Now().Unix() // 创建server时间戳
	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "lobbyserver")
	return nil
}

func (s *LobbyServer) EventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("EventHandler recover, err: ", err)
		}
	}()

	logger.Debug("event - PubsubName:%s, Topic:%s, ID:%s, Data: %s", e.PubsubName, e.Topic, e.ID, e.Data)
	return false, nil
}

func (s *LobbyServer) InvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {

	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("InvokeHandler exception, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
		logger.Warn("InvokeHandler failed, err: ", err)
		return nil, err
	}
	metrics.GaugeInc(metrics.InvokeSubCount)
	msg, err := base.UnPackProtoMsg(in.Data)
	if err != nil {
		logger.Warn("InvokeHandler UnPackProtoMsg, err: ", err)
		return nil, err
	}
	messageID := msg.MsgId

	logger.Info("lobby InvokeHandler begin: ", in.ContentType, in.Verb, in.QueryString, pb.Protocols(messageID), messageID, msg.String())

	out = &common.Content{}
	if messageID == int32(pb.Protocols_PAS2LS_CheckSystemMailReq) {
		out, err = s.CheckSystemMailReq(msg)
	} else if messageID == int32(pb.Protocols_PS2S_SendGMAddMailReq) {
		out, err = s.AddSystemMail(msg)
	} else {
		out, err = nil, fmt.Errorf("unknown message")
	}
	logger.Info("lobby InvokeHandler begin: ", in.ContentType, in.Verb, in.QueryString, pb.Protocols(messageID), messageID, msg.String())
	return out, err
}

func (s *LobbyServer) BindingHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("BindingHandler failed, err: ", err)
		}
	}()

	logger.Debug("binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *LobbyServer) OnNetConnect(c *tcpx.Context) {
	logger.Debug("LobbyServer,implement me")
}

func (s *LobbyServer) OnNetMessage(c *tcpx.Context) {
	logger.Debug("LobbyServer,implement me")
}

func (s *LobbyServer) OnNetClose(c *tcpx.Context) {
	logger.Debug("LobbyServer,implement me")
}

func (s *LobbyServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("LobbyServer,implement me")
}

func (s *LobbyServer) sendPacket(uid string, roleId uint64, uaid string, msgId int32, res proto.Message) (*common.Content, error) {
	data, err := proto.Marshal(res)
	if err != nil {
		logger.Debug("proto.Marshal error:", res)
	}
	b, err := base.PackProtoMsg(msgId, uid, roleId, uaid, data, s.AppId, nil)
	if err != nil {
		logger.Debug("PackProtoMsg err: ", err)
		return nil, err
	}

	return &common.Content{
		Data:        b,
		ContentType: "text/plain",
		DataTypeURL: "",
	}, nil
}

func (s *LobbyServer) Exit() {
	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "lobbyserver", time.Now().Unix()-s.LiveTime)

	logger.Info("Server Exit", s.AppId)
}

func (s *LobbyServer) Reload() error {
	s.LoadConf()
	return nil
}

func (s *LobbyServer) Main() {
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

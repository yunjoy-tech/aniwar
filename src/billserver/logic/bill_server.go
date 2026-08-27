package logic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/yunjoy-tech/aniwar/src/common/datalog/taptap"

	comn "github.com/yunjoy-tech/aniwar/src/common/server"
	"github.com/yunjoy-tech/musae/global"

	"github.com/dapr/go-sdk/service/common"
	"github.com/gin-gonic/gin"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/metrics"
	"github.com/yunjoy-tech/musae/tcpx"
)

type BillServer struct {
	comn.Server
}

func NewBillServer() base.IServer {
	srv := &BillServer{}
	srv.HasPriTopic = true // 开启私有频道订阅
	srv.OnPreInit = srv.OnPreInitHandler
	srv.OnPostInit = srv.OnPostInitHandler
	srv.OnDaprTopicEvent = srv.OnDaprTopicEventHandler
	srv.OnDaprSvcInvoke = srv.OnDaprSvcInvokeHandler
	srv.OnDaprBindInvoke = srv.OnDaprBindInvokeHandler
	srv.OnRegisterMetric = srv.RegisterMetrics
	srv.OnCfgCenterCB = srv.HandlerConfEvent
	return srv
}

func (s *BillServer) RegisterMetrics() {

}

func (s *BillServer) OnPreInitHandler() error {
	// 充值三方回调
	s.RegisterDaprSvcInvokeHandler("/api/pay", s.PayHandler)
	s.RegisterDaprSvcInvokeHandler("/api/refund", s.RefundHandler)

	s.RegisterDaprSvcInvokeHandler("healthz", func(ctx context.Context, in *common.InvocationEvent) (*common.Content, error) {
		out := &common.Content{
			Data:        []byte(time.Now().String()),
			ContentType: "text/plain",
		}
		return out, nil
	})
	return nil
}

func (s *BillServer) OnPostInitHandler() error {

	s.LiveTime = time.Now().Unix() // 创建server时间戳
	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "billserver")
	// if err := s.Server.LoadExcel(); err != nil {
	//	return err
	// }
	// if err := excel.GetShopGiftMgr().LoadByFileName(s.MetaDir, excel.GetShopGiftMgr().GetDataFileName()); err != nil {
	//	return err
	// }
	return nil
}

func (s *BillServer) test(c *gin.Context) {
	c.String(http.StatusOK, "hello world")
}

func (s *BillServer) OnDaprTopicEventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
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
	logger.Debug("[OnDaprTopicEvent] PubsubName:%s, Topic:%s, ID:%s, DataLen: %v", e.PubsubName, e.Topic, e.ID, len(e.RawData))

	err = s.HandlerSubEvent(msg)
	if err != nil {
		logger.Debugf("HandlerSubEvent  error:%+v", err)
	}
	return false, nil
}

func (s *BillServer) OnDaprSvcInvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("[OnDaprSvcInvokeHandler] failed, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
		return
	}
	logger.Debug("[bill] OnDaprSvcInvokeHandler ContentType:%s, Verb:%s, QueryString:%s, %v", in.ContentType, in.Verb, in.QueryString, len(in.Data))
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
	return out, nil
}

func (s *BillServer) OnDaprBindInvokeHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	logger.Debug("binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *BillServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("OnHeartBeat", c.ClientIP())
}

func (s *BillServer) OnNetConnect(c *tcpx.Context) {
	logger.Debug("BillServer:OnNetMessage, implement me")
}

func (s *BillServer) OnNetMessage(c *tcpx.Context) {
	logger.Debug("BillServer:OnNetMessage, implement me")
}

func (s *BillServer) OnNetClose(c *tcpx.Context) {
	logger.Debug("BillServer:OnNetMessage, implement me")
}

func (s *BillServer) Exit() {
	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "billserver", time.Now().Unix()-s.LiveTime)

	logger.Info("Server Exit", s.AppId)
}

func (s *BillServer) Reload() error {
	s.LoadConf()
	return nil
}

func (s *BillServer) Main() {
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

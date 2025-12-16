package logic

import (
	"context"
	"encoding/json"
	"errors"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"net/http"
	"os"
	"time"

	comn "gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/musae/global"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"github.com/dapr/go-sdk/service/common"
	"github.com/gin-gonic/gin"
)

type BillServer struct {
	comn.Server
}

func NewBillServer() base.IServer {
	srv := &BillServer{}
	srv.AppId = "bill"
	srv.InAddr = ":28001"
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

func (s *BillServer) RegisterMetrics() {

}

func (s *BillServer) PreInit() error {
	// 充值三方回调
	s.RegisterRpcHandler("/api/pay", s.PayHandler)
	s.RegisterRpcHandler("/api/refund", s.RefundHandler)

	s.RegisterRpcHandler("healthz", func(ctx context.Context, in *common.InvocationEvent) (*common.Content, error) {
		out := &common.Content{
			Data:        []byte(time.Now().String()),
			ContentType: "text/plain",
		}
		return out, nil
	})
	return nil
}

func (s *BillServer) ServerInit() error {

	s.LiveTime = time.Now().Unix() // 创建server时间戳
	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "billserver")
	// if err := s.Server.LoadExcel(); err != nil {
	//	return err
	// }
	// if err := excel.GetShopGiftMgr().LoadByFileName(s.DataDir, excel.GetShopGiftMgr().GetDataFileName()); err != nil {
	//	return err
	// }
	return nil
}

func (s *BillServer) test(c *gin.Context) {
	c.String(http.StatusOK, "hello world")
}

func (s *BillServer) EventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
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
	logger.Debug("[EventHandler] PubsubName:%s, Topic:%s, ID:%s, DataLen: %v", e.PubsubName, e.Topic, e.ID, len(e.RawData))

	err = s.HandlerSubEvent(msg)
	if err != nil {
		logger.Debugf("HandlerSubEvent  error:%+v", err)
	}
	return false, nil
}

func (s *BillServer) InvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("[InvokeHandler] failed, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
	}
	logger.Debug("[bill] InvokeHandler ContentType:%s, Verb:%s, QueryString:%s, %v", in.ContentType, in.Verb, in.QueryString, len(in.Data))
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

func (s *BillServer) BindingHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
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

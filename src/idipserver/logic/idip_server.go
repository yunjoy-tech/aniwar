package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	comn "gitlab.musadisca-games.com/wangxw/aniwar/src/common/server"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/metrics"
	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"
	"google.golang.org/protobuf/proto"
	"os"
	"time"
)

type IDIPServer struct {
	comn.Server
}

func NewIDIPServer() base.IServer {
	srv := &IDIPServer{}
	srv.AppId = "idip"
	srv.InAddr = ":29001"
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

func (s *IDIPServer) RegisterMetrics() {

}

func (s *IDIPServer) PreInit() error {
	s.InitMap()
	s.InitCmdHandler()
	// lilith gmt数据接口
	s.RegisterRpcHandler("/api/gmt", s.GMTHandler)

	// lilith 问卷系统奖励发放接口
	s.RegisterRpcHandler("/api/survey/reward", s.QuestionRewardHandler)

	// lilith ParkSDK账号删除接口
	s.RegisterRpcHandler("/api/account/delete", s.DelAccountHandler)

	// 内部gmt调用
	s.InitInsideGmtHandlerMap()
	s.RegisterRpcHandler("/api/insideGMT", s.InsideGMT)

	// http reload api
	if baseconf.GetBaseConf().IsDebug {
		s.RegisterRpcHandler("/api/hotReload", s.HotReload)
	}
	return nil
}

func (s *IDIPServer) ServerInit() error {
	s.LiveTime = time.Now().Unix() // 创建server时间戳
	s.NeedExcel = map[string]int{  // 需要加载的策划表 TODO 后台导入字典元数据
		excel.GetPackageMgr().GetDataFileName(): 0,
		excel.GetItemMgr().GetDataFileName():    0,
		// excel.GetBeastarMgr().GetDataFileName():   0,
		excel.GetEquipmentMgr().GetDataFileName(): 0,
		// excel.GetSkinMgr().GetDataFileName():      0,
	}

	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "idipserver")

	// 策划表配置加载
	if err := s.LoadNeedExcel(nil); err != nil {
		return err
	}
	// 国际化文本加载
	if err := s.LoadLocalizedStr(); err != nil {
		return err
	}
	return nil
}

/*func (s *IDIPServer) test(c *gin.Context) {
	c.String(http.StatusOK, "hello world")
}*/

func (s *IDIPServer) EventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("recover failed, err: ", err)
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
	logger.Debug("PubSubName:%s Topic:%s ID:%s DataLen:%v Msg:%s", e.PubsubName, e.Topic, e.ID, len(e.RawData), msg.Str())
	err = s.HandlerSubEvent(msg)
	if err != nil {
		logger.Debugf("HandlerSubEvent  error:%+v", err)
	}
	logger.Debug("event - PubsubName:%s, Topic:%s, ID:%s, DataLen: %v", e.PubsubName, e.Topic, e.ID, len(e.RawData))
	return false, nil
}

func (s *IDIPServer) InvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
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
	logger.Debug("idip.server ===>>> ", uid, pb.Protocols(messageID), messageID)

	logger.Debug("InvokeHandler: ", in.ContentType, in.Verb, in.QueryString, pb.Protocols(messageID), msg.String())

	if messageID == int32(pb.Protocols_PC2LS_HeartBeatReq) { // 测试useractor 调用 svcinvoke 到 idipserver
		logger.Debug("idip ===>>> test ugc gm command...")
	} else {
		logger.Warn("idip ===>>> 未处理的handle, check it!, messageID=", pb.Protocols(messageID), string(messageID))
		return nil, fmt.Errorf("unrealized message id %d", messageID)
	}

	if err != nil {
		logger.Warn("err: ", err)
		return nil, err
	}

	return out, nil
}

func (s *IDIPServer) BindingHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	logger.Debug("binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *IDIPServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("OnHeartBeat", c.ClientIP())
}

func (s *IDIPServer) OnNetConnect(c *tcpx.Context) {
	logger.Debug("IDIPServer:OnNetMessage, implement me")
}

func (s *IDIPServer) OnNetMessage(c *tcpx.Context) {
	logger.Debug("IDIPServer:OnNetMessage, implement me")
}

func (s *IDIPServer) OnNetClose(c *tcpx.Context) {
	logger.Debug("IDIPServer:OnNetMessage, implement me")
}

func (s *IDIPServer) Exit() {

	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "idipserver", time.Now().Unix()-s.LiveTime)
	logger.Info("Server Exit", s.AppId)
}

func (s *IDIPServer) Reload() error {
	logger.Info("IDIPServer Reload ===>>>")
	err := s.LoadConf()
	if err != nil {
		logger.Errorf("reload --> LoadConf got err:%+v", err)
		return err
	}

	err = s.LoadNeedExcel(nil)
	if err != nil {
		logger.Errorf("reload --> LoadExcel got err:%+v", err)
		return err
	}
	return nil
}

func (s *IDIPServer) Main() {
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

func (s *IDIPServer) sendPacket(uid string, roleId uint64, uaid string, msgId int32, res proto.Message) (*common.Content, error) {
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

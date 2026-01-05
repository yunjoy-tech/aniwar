package frame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	myCommon "gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/aniwar/src/common/db"
	comn "gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/aniwar/src/idipserver/logic"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"github.com/dapr/go-sdk/actor/runtime"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var GSrv *ActorServer

type CmdType int

const (
	CMD_TYPE_PERSON CmdType = 1
	CMD_TYPE_SERVER CmdType = 2
)

type CmdInfo struct {
	Name string  // 指令名
	Desc string  // 功能描述
	Help string  // 指令用法
	Type CmdType // 指令类型 1=个人指令，2=全服指令
}

type ActorServer struct {
	comn.Server
	CmdLogicHandlerMap map[string]*CmdInfo
	CloseFuncMap       sync.Map
	SysMailMgr         *SysMailMgr
}

func NewActorServer() base.IServer {
	srv := &ActorServer{}
	srv.AppId = "actor"
	srv.InAddr = ":24001"
	srv.GRPCPort = "50001"
	srv.HasPriTopic = true // 开启私有频道订阅
	srv.OnPreInit = srv.OnPreInitHandler
	srv.OnPostInit = srv.OnPostInitHandler
	srv.OnDaprTopicEvent = srv.OnDaprTopicEventHandler
	srv.OnDaprSvcInvoke = srv.OnDaprSvcInvokeHandler
	srv.OnDaprBindInvoke = srv.OnDaprBindInvokeHandler
	srv.OnRegisterMetric = srv.RegisterMetrics
	srv.OnCfgCenterCB = srv.HandlerConfEvent
	srv.OnCronEveryHour = srv.OnCronEveryHourHandler

	srv.CmdLogicHandlerMap = make(map[string]*CmdInfo)
	srv.CloseFuncMap = sync.Map{}
	srv.SysMailMgr = NewSysMailMgr(srv)

	GSrv = srv
	return srv
}

func (s *ActorServer) RegisterMetrics() {
	if global.AppID == global.ACTOR_SVC {
		metrics.RegisterGauge(metrics.UserActorCount, false)
		metrics.RegisterGauge(metrics.RoomActorCount, false)
		metrics.RegisterGauge(metrics.AllianceActorCount, false)
		metrics.RegisterGauge(metrics.UserCount, false)

	} else if global.AppID == global.CENTER_SVC {
		metrics.RegisterGauge(metrics.AllUserCount, false)
	}
}

// RegisterCloseFunc 注册关闭功能
func (s *ActorServer) RegisterCloseFunc() {
	// 动态配置
	data, err := s.GetFromConfigCenter(db.KeyCfgGlobalCloseFunc)
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
			s.CloseFuncMap.Store(int32(id), id)
			logger.Infof("Register Close Func %d", id)
		}
	}
}

func (s *ActorServer) RegisterCmdInfo(name, desc, help string, typ CmdType) {
	if _, ok := s.CmdLogicHandlerMap[name]; !ok {
		s.CmdLogicHandlerMap[name] = &CmdInfo{
			Name: name,
			Desc: desc,
			Help: name + " " + help,
			Type: typ,
		}
		logger.Debugf("register cmd: %s", name)
	} else if ok {
		logger.Errorf("Duplicate cmd are registered: %s", name)
	}
}

func (s *ActorServer) InitCmd() {
	s.RegisterCmdInfo(myCommon.GM_ACTOR_SHOW, "打印actor信息", "[actorType]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ACTOR_DEL, "删除actor服务", "[actorType]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ADD_ITEM, "增加道具", "[itemId] [itemNum]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ADD_ITEM_ALL, "增加所有道具", "", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ADD_ITEM_BY_TYPE, "增加指定类型道具", "[type] [num]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DEL_ITEM_BY_ID, "删除道具", "[itemId] [itemNum]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ADD_CARD_EXP, "增加卡牌经验", "[cardId] [exp]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ADD_FAVORITE_EXP, "增加卡牌好感度", "[cardId] [exp]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_CARD, "测试抽卡", "[poolId] [num]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_KICKOUT, "测试踢人", "[second]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_BANNED, "测试封禁", "[second msg]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_SET_CARD_STRENGTH, "修改卡牌体力", "[cardId] [value]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_MAIL, "测试邮件", "[templateId] [num]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DEL_MONEY, "扣除货币", "[type] [value]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DEL_STAMINA, "扣除玩家体力", "[value]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DIRECT_LEVEL_UP, "设置玩家等级", "[value]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_RESET_LEVEL, "重置玩家等级", "", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_RESET_CARD_POOL_LOG, "重置抽卡保底记录", "", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_WEAR_EQUIP, "卡牌穿戴装备", "[cardId] [equipId]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_ADD_PLAYER_EXP, "增加玩家经验", "[exp]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DIRECT_COMPLETE_OBJECT, "完成剧情物件交互", "[objectId]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DIRECT_COMPLETE_QUEST, "完成剧情任务", "[questId]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_SIGN, "测试签到", "[groupId] [param]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_PROTO, "测试协议", "[cmd] [param]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_SET_LIGHTING_COMPOSE_TREE_TS, "设置光和树时间", "[refreshTS]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_SET_SUPER_CARD, "卡牌强化", "[cardId] [type]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_SAVE_STORY_FLAG, "保存剧情标记", "[flag]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_LEVEL_FINISH, "完成地图关卡", "[levelId] [result]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_UGC, "测试屏蔽字接口", "[content] [type]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_SENSITIVE, "测试本地屏蔽词库", "[content]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_Battle_chapter, "测试战斗校验", "", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_CHECKBATTLE_RELOAD_EXCEL, "战斗校验热更", "", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_GEN_CODE, "测试生成礼包码", "[giftId] [num]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_USE_CODE, "测试使用礼包码", "[code]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_DROP, "测试道具掉落组", "[dropId] [num]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_RESET_DUTY_TASK, "重置值日生任务", "", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_DIRECT_COMPLETE_DUTY_TASK, "直接完成值日生任务", "[taskId]", CMD_TYPE_PERSON)
	s.RegisterCmdInfo(myCommon.GM_TEST_RECOMMEND, "测试好友推荐列表", "", CMD_TYPE_PERSON)
}

func (s *ActorServer) OnPreInitHandler() error {
	if conf.Base().IsDebug {
		s.RegisterDaprSvcInvokeHandler("/api/hotReload", s.HotReload)
	}
	return nil
}

func (s *ActorServer) OnPostInitHandler() error {
	logger.Debug("ActorServer Init")

	s.LiveTime = time.Now().Unix() // 创建server时间戳

	// center 跳过业务配置加载
	if global.IsActor(s.AppId) {
		s.LoadWordCfg() // 加载静态屏蔽词库
		// s.InitDynamicWord()   // 加载动态屏蔽词
		s.RegisterCloseFunc() // 加载关闭的功能
		s.InitCmd()
		if err := s.GetSystemMail(s.SysMailMgr.Data); err != nil {
			logger.Errorf("load 全局邮件失败 %+v", err)
			return err
		}
		logger.Infof("load 全局邮件 %+v", s.SysMailMgr.Data)
		if err := s.LoadExcelData(); err != nil {
			return err
		}
	}

	// 服务启动埋点
	taptap.ServiceStart(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "actorserver")

	return nil
}

func (s *ActorServer) OnNetConnect(c *tcpx.Context) {
	logger.Debug("ActorServer:OnNetConnect,implement me")
}

func (s *ActorServer) OnNetMessage(c *tcpx.Context) {
	logger.Debug("ActorServer:OnNetMessage,implement me")
}

func (s *ActorServer) OnNetClose(c *tcpx.Context) {
	logger.Debug("ActorServer:OnNetClose,implement me")
}

func (s *ActorServer) OnHeartBeat(c *tcpx.Context) {
	logger.Debug("ActorServer:OnHeartBeat,implement me")
}

func (s *ActorServer) OnDaprTopicEventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Errorf("recover failed err:%v", err)
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
	return false, nil
}

func (s *ActorServer) OnDaprSvcInvokeHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("failed, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
		logger.Warn("failed, err: ", err)
		return nil, err
	}

	msg, err := base.UnPackProtoMsg(in.Data)
	if err != nil {
		logger.Warnf("UnPackProtoMsg text-plain err:%+v data:%s in:%+v ", err, in.Data, in)
		return nil, err
	}

	metrics.GaugeInc(metrics.InvokeSubCount)
	messageID, uid, data := msg.MsgId, msg.UserId, msg.Data
	logger.Debugf("ContentType:%s Verb:%s QueryString:%s msgId:%v msgId:%v uid:%s dataLen:%v",
		in.ContentType, in.Verb, in.QueryString, pb.Protocols(messageID), messageID, uid, len(data))
	if messageID == int32(pb.Protocols_PS2AS_GetGmListReq) {
		out, err = s.GetUserGMList(msg)
	} else if messageID == int32(pb.Protocols_PS2S_SendGMAddMailReq) {
		out, err = s.SysMailMgr.AddSystemMailReq(msg)
	} else if messageID == int32(pb.Protocols_PS2S_GetExcelConfigReq) {
		req := &pb.S2S_GetExcelConfigReq{}
		if err = msg.UnmarshalData(req); err != nil {
			return nil, err
		}
		excelData := comn.GetExcelData(req.SheetName)
		out = &common.Content{
			Data:        excelData,
			ContentType: "text/plain",
			DataTypeURL: "",
		}
	} else if messageID == int32(pb.Protocols_PS2S_SvcStatusReq) || messageID == int32(pb.Protocols_PS2S_HotReloadNotifyReq) {
		ret, err := s.ActorInvoke(stub.CenterActorType, global.CenterActorID, msg)
		if err != nil {
			logger.Warn("ActorInvoke err:", err)
		}
		b, e := ret.Marshal()
		out, _ = &common.Content{
			Data:        b,
			ContentType: "text/plain",
			DataTypeURL: "",
		}, e
	}

	return out, nil
}

func (s *ActorServer) GetUserGMList(msg *base.ProtoMsg) (*common.Content, error) {
	req := &pb.S2AS_GetGmListReq{}
	if err := msg.UnmarshalData(req); err != nil {
		return nil, err
	}
	rsp := make([]*logic.GmHelpRsp, 0)
	for _, x := range s.CmdLogicHandlerMap {
		if req.GetGlobalGM && x.Type == CMD_TYPE_SERVER {
			rsp = append(rsp, &logic.GmHelpRsp{
				CmdName: x.Name,
				Desc:    x.Desc,
				Help:    x.Help,
			})
		} else if !req.GetGlobalGM && x.Type == CMD_TYPE_PERSON {
			rsp = append(rsp, &logic.GmHelpRsp{
				CmdName: x.Name,
				Desc:    x.Desc,
				Help:    x.Help,
			})
		}
	}
	data, err := json.Marshal(&rsp)

	return &common.Content{Data: data}, err
}

func (s *ActorServer) OnDaprBindInvokeHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("OnDaprBindInvokeHandler failed, err: ", err)
		}
	}()

	logger.Debug("binding - Data:%s, Meta:%v", in.Data, in.Metadata)
	return nil, nil
}

func (s *ActorServer) Exit() {
	logger.Info("ActorServer Exit")

	// 退出埋点
	taptap.ServiceStop(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "actorserver", time.Now().Unix()-s.LiveTime)
}

func (s *ActorServer) Reload() error {
	var (
		err error
	)
	logger.Info("ActorServer Reload ===>>>")
	err = s.LoadConf()
	if err != nil {
		logger.Errorf("reload --> LoadConf got err:%+v", err)
		return err
	}

	err = s.LoadExcel()
	if err != nil {
		logger.Errorf("reload --> LoadExcel got err:%+v", err)
		return err
	}

	return nil
}

func (s *ActorServer) OnCronEveryHourHandler(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	logger.Debugf("ActorServer binding OnCronEveryHourHandler Data:%s, Meta:%v", in.Data, in.Metadata)

	hour := time.Now().Hour()
	switch hour {
	case 0:
		params, _ := json.Marshal(make([]byte, 0))
		runtime.GetActorRuntimeInstance().InvokeActors(stub.UserActorType, "Hour0Handler", params)
	case 5:
		// 隔天间隔，刷新在线玩家数据
		params, _ := json.Marshal(make([]byte, 0))
		runtime.GetActorRuntimeInstance().InvokeActors(stub.UserActorType, "Hour5Handler", params)
	case 23:
	}

	// 服务定时器埋点
	taptap.ServerHourComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "OnCronEveryHourHandler", time.Now().Unix()-s.LiveTime)

	return nil, nil
}

func (s *ActorServer) Main() {
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

func (s *ActorServer) sendPacket(req *base.ProtoMsg, msgId int32, res proto.Message) (*common.Content, error) {
	data, err := proto.Marshal(res)
	if err != nil {
		logger.Debug("proto.Marshal error:", res)
	}
	b, err := base.PackProtoMsg(msgId, req.UserId, req.RoleId, req.UAID, data, s.AppId, nil)
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

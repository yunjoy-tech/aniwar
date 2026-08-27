package server

import (
	"fmt"
	"github.com/yunjoy-tech/aniwar/src/common/actor/stub"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/baseconf"
	"github.com/yunjoy-tech/musae/gamelib/guid"
	"github.com/yunjoy-tech/musae/global"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/utils"
	"google.golang.org/protobuf/proto"
	"runtime"
	"strings"
	"time"
)

// 服务运行配置信息输出

// 服务基础运行信息
func (s *Server) BasicInfo() string {
	maxProcs := runtime.NumCPU()
	defMaxProcs := runtime.GOMAXPROCS(maxProcs)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s\n", "========== info =========="))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "Now", time.Now().String()))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "Version", global.VERSION))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "AppVersion", global.APP_VERSION))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "RollingVersion", global.ROLLING_VERSION))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "IsCloud", conf.Base().Cloud))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "IsDev", conf.Base().IsDebug))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "Metric", conf.Base().Metric))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "DefProcNum", defMaxProcs))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "NewProcNum", maxProcs))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "CPU", runtime.NumCPU()))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "GOROOT", runtime.GOROOT()))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "GOOS", runtime.GOOS))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "DefMAXPROCS", defMaxProcs))
	sb.WriteString(fmt.Sprintf("%-13s: %v\n", "MAXPROCS", maxProcs))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "AppId", s.AppId))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "InAddr", s.InAddr))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "OutAddr", s.OutAddr))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "WebAddr", s.WebAddr))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "GRPCPort", s.GRPCPort))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "Gateway", global.Gateway))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "TcpAddr", global.TcpAddr))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "Actors", s.Actors))
	sb.WriteString(fmt.Sprintf("%-13s: %d\n", "UserActor", global.UserActorCount))
	sb.WriteString(fmt.Sprintf("%-13s: %d\n", "SceneActor", global.RoomActorCount))
	sb.WriteString(fmt.Sprintf("%-13s: %d\n", "AllianceActor", global.AllianceActorCount))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "ConfFile", s.Args["config"]))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "PrivateTopic", s.PrivateTopicID()))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "LogDir", baseconf.GetLogConf().Dir))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "PProfAddr", s.PProfAddr))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "StartTime", time.Unix(global.StartTime, 0).Format("2006-01-02 15:04:05.000 -0700 MST")))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "updateAddrARD", conf.SrvAddr().UpdateAddrARD))
	sb.WriteString(fmt.Sprintf("%-13s: %s\n", "updateAddrIOS", conf.SrvAddr().UpdateAddrIOS))
	return sb.String()
}

// TODO 配置文件信息输出

func (s *Server) Status() string {
	data, err := proto.Marshal(&pb.S2S_SvcStatusReq{})
	if err != nil {
		logger.Warn("proto Marshal  err:", err)
		return ""
	}
	bytes := s.CenterSrvInvoke(int32(pb.Protocols_PS2S_SvcStatusReq), data)
	res := &pb.S2S_SvcStatusRes{}
	if err = proto.Unmarshal(bytes, res); err != nil {
		logger.Warn("proto unmarshal err:", err)
		return ""
	}

	status := fmt.Sprintf("\n%s\n", "========== status ==========")
	status += fmt.Sprintf("%s\n", utils.PrettyJson(res))
	szInfo := s.BasicInfo()
	szInfo = status + szInfo
	logger.Infof("server info:%s", szInfo)
	return szInfo
}

// 请求center协议
func (s *Server) CenterSrvInvoke(msgId int32, data []byte) []byte {
	msg, err := s.ActorInvoke(stub.CenterActorType, global.CenterActorID, &base.ProtoMsg{
		AppId:   ACTOR_SVC,
		MsgId:   msgId,
		UserId:  "",
		RoleId:  0,
		UAID:    global.CenterActorID,
		Data:    data,
		ErrCode: 0,
		ReqIdx:  guid.GenIntUuid(),
		Topic:   "",
		Uids:    nil,
	})
	if err != nil {
		logger.Warn("ActorInvoke err:", err)
		return nil
	}
	return msg.Data
}

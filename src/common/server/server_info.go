package server

import (
	"fmt"
	"gitee.com/bychannel/aniwar/src/common/conf"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/base"
	"google.golang.org/protobuf/proto"
	"runtime"
	"time"

	"gitee.com/bychannel/musae/framework/global"
	"gitee.com/bychannel/musae/framework/logger"
	"gitee.com/bychannel/musae/framework/utils"
)

func (s *Server) Info() string {
	// output version info
	maxProcs := runtime.NumCPU()
	defMaxProcs := runtime.GOMAXPROCS(maxProcs)
	var szInfo string
	szInfo += (fmt.Sprintf("\n%s\n", "========== info =========="))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "Now", time.Now().String()))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "Version", global.VERSION))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "AppVersion", global.APP_VERSION))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "RollingVersion", global.ROLLING_VERSION))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "IsCloud", global.IsCloud))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "IsDev", global.IsDev))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "Metric", s.IsMetric))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "DefProcNum", defMaxProcs))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "NewProcNum", maxProcs))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "CPU", runtime.NumCPU()))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "GOROOT", runtime.GOROOT()))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "GOOS", runtime.GOOS))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "DefMAXPROCS", defMaxProcs))
	szInfo += (fmt.Sprintf("%-13s: %v\n", "MAXPROCS", maxProcs))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "AppId", s.AppId))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "InAddr", s.InAddr))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "OutAddr", s.OutAddr))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "WebAddr", s.WebAddr))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "GRPCPort", s.GRPCPort))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "Gateway", global.Gateway))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "TcpAddr", global.TcpAddr))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "Actors", s.Actors))
	szInfo += (fmt.Sprintf("%-13s: %d\n", "UserActor", global.UserActorCount))
	szInfo += (fmt.Sprintf("%-13s: %d\n", "SceneActor", global.RoomActorCount))
	szInfo += (fmt.Sprintf("%-13s: %d\n", "AllianceActor", global.AllianceActorCount))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "ConfFile", s.ConfFile))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "PrivateTopic", s.PrivateTopicID()))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "LogDir", s.LogDir))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "PProfAddr", s.PProfAddr))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "StartTime", time.Unix(global.StartTime, 0).Format("2006-01-02 15:04:05.000 -0700 MST")))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "updateAddrARD", conf.SrvAddr().UpdateAddrARD))
	szInfo += (fmt.Sprintf("%-13s: %s\n", "updateAddrIOS", conf.SrvAddr().UpdateAddrIOS))
	szInfo += (fmt.Sprintf("%-13v: %s\n", "Args", s.Args))
	// szInfo += (fmt.Sprintf("\n%s\n", "========== server.conf =========="))
	// szInfo += (fmt.Sprintf("%s\n", utils.PrettyJson(conf.GConf())))
	return szInfo
}

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

	status := (fmt.Sprintf("\n%s\n", "========== status =========="))
	status += (fmt.Sprintf("%s\n", utils.PrettyJson(res)))
	szInfo := s.Info()
	szInfo = status + szInfo
	logger.Infof("server info:%s", szInfo)
	return szInfo
}

// 请求center协议
func (s *Server) CenterSrvInvoke(msgId int32, data []byte) []byte {
	msg, err := s.ActorInvoke(global.CenterActorType, global.CenterActorID, &base.ProtoMsg{
		AppId:   global.ACTOR_SVC,
		MsgId:   msgId,
		UserId:  "",
		RoleId:  0,
		UAID:    global.CenterActorID,
		Data:    data,
		ErrCode: 0,
		ReqIdx:  utils.GenIntUUID(),
		Topic:   "",
		Uids:    nil,
	})
	if err != nil {
		logger.Warn("ActorInvoke err:", err)
		return nil
	}
	return msg.Data
}

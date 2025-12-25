package server

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/global"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO 需要精简
// AnalysisArgs 运行参数解析
// 这里不使用flag包进行解析，主要是为了兼容sidecar运行模式，避免和dapr的参数解析冲突
func (s *Server) AnalysisArgs() {
	s.Args = make(map[string]string)
	for i := 1; i < len(os.Args); i++ {
		str := os.Args[i]
		fmt.Printf("run args %-3d: %s\n", i, str)
		szArgs := strings.SplitN(str, "=", 2)
		if len(szArgs) != 2 || szArgs[0] == "" || szArgs[1] == "" {
			fmt.Println("error arg: ", str)
			continue
		}
		s.Args[szArgs[0]] = szArgs[1]
	}
	// todo 未判定是否传参，后续处理
	global.AppID = s.Args["appId"]
	global.ROLLING_VERSION = s.Args["rollingVersion"]
	global.Gateway = s.Args["gateway"]
	global.TcpAddr = s.Args["tcpAddr"]

	// todo
	global.StartTime = time.Now().Unix()

	// TODO
	// global.RdsCfgCenterHost = s.Args["rdscfghost"]
	// global.RdsCfgCenterPass = s.Args["rdscfgpass"]
	// global.RdsCfgNameSpace = s.Args["rdscfgns"]
	// global.RdsCfgGroup = s.Args["rdscfggroup"]
}

func HelpInfo() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "help":
			fmt.Printf("version: %s\n", global.VERSION)
			fmt.Printf("app version: %s\n", global.APP_VERSION)
			fmt.Printf("\n")
			fmt.Printf("有效命令：\n")
			fmt.Printf("	ps: server [command]\n")
			fmt.Printf("	%-10s %s\n", "version", "打印当前版本号")

			fmt.Printf("\n")

			fmt.Printf("有效运行参数：\n")
			fmt.Printf("	ps: server [appid=xxx] [actor=xxx]\n")
			fmt.Printf("	%-10s %s\n", "appid", "服务器唯一服务类型ID, 可以有多个进程实例, 支持类型：[login, gate, lobby,actor]")
			fmt.Printf("	%-10s %s\n", "actor", "指定微服务类型, 在appid=actor时, 必须指定, 支持类型：[user, camp]")
			fmt.Printf("	%-10s %s\n", "inaddr", "服务监听端口")
			fmt.Printf("	%-10s %s\n", "outaddr", "服务对外监听端口, 服务面向用户监听, login, gate, idip")
			fmt.Printf("	%-10s %s\n", "webaddr", "web服务对外监听端口, idip, bill")
			fmt.Printf("	%-10s %s\n", "gport", "dapr监听的 gRPC 端口")
			fmt.Printf("	%-10s %s\n", "gateway", "服务网关地址")
			fmt.Printf("	%-10s %s\n", "tcpaddr", "长链接地址")
			fmt.Printf("	%-10s %s\n", "pprof", "pprof监听的端口")
			fmt.Printf("	%-10s %s\n", "conf", "服务配置文件, 默认路径：output/res/server.conf")
			fmt.Printf("	%-10s %s\n", "logdir", "服务日志文件, 默认路径：log")
			fmt.Printf("	%-10s %s\n", "datadir", "服务日志文件, 默认路径：output/data")
			fmt.Printf("	%-10s %s\n", "metric", "指标收集开关, 默认metric=0")
			fmt.Printf("	%-10s %s\n", "dev", "是否开发环境, dev=0")
			fmt.Printf("	%-10s %s\n", "cloud", "运行模式, 公有云或者内网, cloud=[1/0]")
			fmt.Printf("	%-10s %s\n", "version", "服务版本号")
			fmt.Printf("	%-10s %s\n", "rollingVersion", "滚动更新版本号")
			fmt.Printf("	%-10s %s\n", "restart", "服务滚动重启")
			os.Exit(0)
		default:
			fmt.Printf("无效命令: %s\n", os.Args[1])
			os.Exit(0)
		}
	}
}

func (s *Server) InitSrvArgs() error {
	s.AppId = global.AppID
	s.InAddr = fmt.Sprintf(":%s", s.Args["inAddr"])
	s.OutAddr = fmt.Sprintf(":%s", s.Args["outAddr"])
	s.WebAddr = fmt.Sprintf(":%s", s.Args["webAddr"])
	s.GRPCPort = s.Args["grpcPort"]
	s.PProfAddr = fmt.Sprintf("0.0.0.0:%s", s.Args["pprofAddr"])

	if !IsValidAppId(s.AppId) {
		fmt.Println("AppId error:", s.AppId)
	}

	if s.AppId == global.ACTOR_SVC || s.AppId == global.CENTER_SVC {
		actors := strings.Split(s.Args["actor"], "|")
		for _, v := range actors {
			switch v {
			case global.UserActorType,
				global.RoomActorType,
				global.AllianceActorType,
				global.CenterActorType,
				global.MailActorType:
				s.Actors = append(s.Actors, v)
			default:
				return errorx.Newf("unknown actor type: %s, actors:%+v", v, actors)
			}
		}
	}

	global.HostName = os.Getenv("HOSTNAME")
	if global.HostName == "" {
		global.HostName = global.AppID
	}
	fmt.Println("hostname:", global.HostName)
	if conf.Base().Cloud {
		ids := strings.Split(global.HostName, "-")
		if len(ids) == 2 {
			id, err := strconv.Atoi(ids[1])
			if err != nil {
				fmt.Printf("hostname error: %+v\n", err)
			}
			global.SID = int64(id)
		}
	} else {
		global.SID = 0
	}
	return nil
}

package server

import (
	"flag"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/global"
	"os"
	"strconv"
	"strings"
	"time"
)

// 运行参数定义
var (
	ArgConfig    string // 配置文件
	ArgAppId     string // app id
	ArgActor     string // 注册的actor类型
	ArgInAddr    string // 服务监听端口
	ArgOutAddr   string // 服务对外监听端口, 服务面向用户监听, login, gate, idip
	ArgWebAddr   string // web服务对外监听端口, idip, bill
	ArgGrpcPort  string // dapr监听的 gRPC 端口
	ArgPprofAddr string // pprof监听的端口
	ArgDev       int    // 是否开发环境
)

func init() {
	flag.StringVar(&ArgConfig, "config", "", "server config file path")
	flag.StringVar(&ArgAppId, "app-id", "", "server app id")
	flag.StringVar(&ArgActor, "actor", "", "actor server register actor type")
	flag.StringVar(&ArgInAddr, "in-addr", "", "server in port")
	flag.StringVar(&ArgOutAddr, "out-addr", "", "server out port")
	flag.StringVar(&ArgWebAddr, "web-addr", "", "server web port")
	flag.StringVar(&ArgGrpcPort, "grpc-port", "", "dapr server register grpc port")
	flag.StringVar(&ArgPprofAddr, "pprof-addr", "", "server pprof register port")
	flag.IntVar(&ArgDev, "dev", 0, "server run mode")
}

// 运行参数解析
func (s *Server) AnalysisArgs() {
	flag.Parse()

	s.Args = make(map[string]string)
	// 遍历所有注册过的参数，输出日志，并且保存到Args中
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Printf("运行参数: %s = %s\n", f.Name, f.Value.String())
		s.Args[f.Name] = f.Value.String()
	})

	// TODO 待删除
	// global.RdsCfgCenterHost = s.Args["rdscfghost"]
	// global.RdsCfgCenterPass = s.Args["rdscfgpass"]
	// global.RdsCfgNameSpace = s.Args["rdscfgns"]
	// global.RdsCfgGroup = s.Args["rdscfggroup"]
}

// 初始化运行参数，包括server层的参数和框架层的参数
func (s *Server) InitRunArgs() error {
	// 注册框架层的参数（全局参数，server和musae都会使用到的参数）
	global.AppID = s.Args["app-id"]
	global.ROLLING_VERSION = s.Args["rollingVersion"] // TODO
	global.Gateway = s.Args["gateway"]                // TODO
	global.TcpAddr = s.Args["tcpAddr"]                // TODO
	global.StartTime = time.Now().Unix()

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

	// 注册server层的参数
	s.AppId = global.AppID
	s.InAddr = fmt.Sprintf(":%s", s.Args["in-addr"])
	s.OutAddr = fmt.Sprintf(":%s", s.Args["out-addr"])
	s.WebAddr = fmt.Sprintf(":%s", s.Args["web-addr"])
	s.GRPCPort = s.Args["grpc-port"]
	s.PProfAddr = fmt.Sprintf("0.0.0.0:%s", s.Args["pprof-addr"])

	if !IsValidAppId(s.AppId) {
		return errorx.Newf("app-id error: %s", s.AppId)
	}

	if s.AppId == ACTOR_SVC || s.AppId == CENTER_SVC {
		actors := strings.Split(s.Args["actor"], "|")
		for _, v := range actors {
			switch v {
			case stub.UserActorType,
				stub.RoomActorType,
				stub.AllianceActorType,
				stub.CenterActorType,
				stub.MailActorType:
				s.Actors = append(s.Actors, v)
			default:
				return errorx.Newf("unknown actor type: %s, actors:%+v", v, actors)
			}
		}
	}

	return nil
}

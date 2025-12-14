package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/aniwar/src/common/gmeta"
	"gitee.com/aniwar2/aniwar/src/excel/data"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/baseconf"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/tcpx"
	"gitee.com/aniwar2/musae/framework/utils"
	"gitee.com/aniwar2/musae/framework/wordfilter"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"os"
	"strconv"
	"strings"
	"time"
)

func (s *Server) AnalysisArgs() {
	for i := 1; i < len(os.Args); i++ {
		str := os.Args[i]

		fmt.Printf("run args %-3d: %s\n", i, str)
		// szArgs := strings.Split(str, "=")
		szArgs := strings.SplitN(str, "=", 2)
		if len(szArgs) != 2 || szArgs[0] == "" || szArgs[1] == "" {
			fmt.Println("error arg: ", str)
			continue
		}
		s.Args[szArgs[0]] = szArgs[1]
	}

	confFile, ok := s.Args["conf"]
	if ok {
		s.ConfFile = confFile
	}
	cloud, ok := s.Args["cloud"]
	if ok {
		if cloud == "1" {
			global.IsCloud = true
		} else {
			global.IsCloud = false
		}
	}

	dev, ok := s.Args["dev"]
	if ok {
		if dev == "1" {
			global.IsDev = true
		} else {
			global.IsDev = false
		}
	}
	rollingVersion, ok := s.Args["rollingVersion"]
	if ok {
		global.ROLLING_VERSION = rollingVersion
	}

	global.RdsCfgCenterHost = s.Args["rdscfghost"]
	global.RdsCfgCenterPass = s.Args["rdscfgpass"]
	global.RdsCfgNameSpace = s.Args["rdscfgns"]
	global.RdsCfgGroup = s.Args["rdscfggroup"]

	fmt.Printf("run args: %+v\n", s.Args)
}

func (s *Server) Helper() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("version: %s\n", global.VERSION)
			os.Exit(0)
		case "appversion":
			fmt.Printf("app version: %s\n", global.APP_VERSION)
			os.Exit(0)
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
		}
	}

}

func (s *Server) InitSrvArgs() error {
	datadir, ok := s.Args["datadir"]
	if ok {
		s.DataDir = datadir
	}

	logdir, ok := s.Args["logdir"]
	if ok {
		s.LogDir = logdir
	}

	appid, ok := s.Args["appid"]
	if ok {
		s.AppId = appid
	}
	global.AppID = s.AppId

	inaddr, ok := s.Args["inaddr"]
	if ok {
		s.InAddr = fmt.Sprintf(":%s", inaddr)
	}

	outaddr, ok := s.Args["outaddr"]
	if ok {
		s.OutAddr = fmt.Sprintf(":%s", outaddr)
	}

	webaddr, ok := s.Args["webaddr"]
	if ok {
		s.WebAddr = fmt.Sprintf(":%s", webaddr)
	}

	pprof, ok := s.Args["pprof"]
	if ok {
		s.PProfAddr = fmt.Sprintf("0.0.0.0:%s", pprof)
	}

	gport, ok := s.Args["gport"]
	if ok {
		s.GRPCPort = gport
	}

	gateway, ok := s.Args["gateway"]
	if ok {
		// s.Gateway = gateway
		global.Gateway = gateway
	}

	tcpAddr, ok := s.Args["tcpaddr"]
	if ok {
		global.TcpAddr = tcpAddr
	}

	updateAddr, ok := s.Args["updateaddr"]
	if ok {
		global.UpdateAddr = updateAddr
	}

	metric, ok := s.Args["metric"]
	if ok {
		if metric == "1" {
			s.IsMetric = true
		} else {
			s.IsMetric = false
		}
	}

	metricPort, ok := s.Args["metricport"]
	if ok {
		if ok {
			global.MetricPort = fmt.Sprintf(":%s", metricPort)
		}
	}

	env, ok := s.Args["env"]
	if ok {
		global.Env = env
	}

	rdsSrvHost, ok := s.Args["rdssrvhost"]
	if ok {
		global.RdsSrvHost = rdsSrvHost
	}

	rdsSrvPass, ok := s.Args["rdssrvpass"]
	if ok {
		// pass, err := base64.StdEncoding.DecodeString(rdsSrvPass)
		// if err != nil {
		//	logger.Fatalf("decode password %s err:", err, rdsSrvPass)
		// }
		global.RdsSrvPass = string(rdsSrvPass)
	}

	esSrvHost, ok := s.Args["essrvhost"]
	if ok {
		global.ESSrvHost = esSrvHost
	}

	esSrvPass, ok := s.Args["essrvpass"]
	if ok {
		// pass, err := base64.StdEncoding.DecodeString(esSrvPass)
		// if err != nil {
		//	logger.Fatalf("decode password %s err:", err, esSrvPass)
		// }
		global.ESSrvPass = string(esSrvPass)
	}

	esSrvUser, ok := s.Args["essrvuser"]
	if ok {
		global.ESSrvUser = esSrvUser
	}
	s.ResetSrvArgs()
	/*restart, ok := s.Args["restart"]
	if ok {
		global.StartTime = restart
	} else {
		global.StartTime = time.Now().String()
	}*/
	// global.StartTime = time.Now().Format("2006-01-02 15:04:05.000 -0700 MST")
	global.StartTime = time.Now().Unix()
	if !IsValidAppId(s.AppId) {
		fmt.Println("AppId error:", s.AppId)
	}

	if s.AppId == global.ACTOR_SVC || s.AppId == global.CENTER_SVC {
		actor, ok := s.Args["actor"]
		if !ok {
			panic(any(fmt.Sprintf("Actor type error:", actor)))
		}
		actors := strings.Split(actor, "|")
		for _, v := range actors {
			switch v {
			case global.UserActorType,
				global.RoomActorType,
				global.AllianceActorType,
				global.CenterActorType,
				global.MailActorType:
				s.Actors = append(s.Actors, v)
			default:
				return errors.Errorf("unknown actor type: %s, actors:%+v", v, actors)
			}
		}
	}

	global.HostName = os.Getenv("HOSTNAME")
	if global.Env == global.ENV_PC {
		global.HostName = global.AppID
	}
	fmt.Println("hostname:", global.HostName)
	if global.IsCloud {
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

func (s *Server) ResetSrvArgs() {
	if global.UpdateAddr != "" {
		conf.SrvAddr().UpdateAddrARD = []string{global.UpdateAddr + "android/"}
		conf.SrvAddr().UpdateAddrIOS = []string{global.UpdateAddr + "ios/"}
	}

	if global.Env != "" {
		baseconf.GetBaseConf().Env = global.Env
	}

	if global.RdsSrvHost != "" {
		baseconf.GetBaseConf().RedisConf.Addr = global.RdsSrvHost
		baseconf.GetBaseConf().RedisConf.AddrDev = global.RdsSrvHost
	}

	if global.RdsSrvPass != "" {
		baseconf.GetBaseConf().RedisConf.Password = global.RdsSrvPass
	}

	if global.ESSrvHost != "" {
		baseconf.GetBaseConf().ESConf.Addr = []string{global.ESSrvHost}
		baseconf.GetBaseConf().ESConf.AddrDev = global.ESSrvHost
	}

	if global.ESSrvPass != "" {
		baseconf.GetBaseConf().ESConf.Password = global.ESSrvPass
	}

	if global.ESSrvUser != "" {
		baseconf.GetBaseConf().ESConf.UserName = global.ESSrvUser
	}
}

func (s *Server) Init() error {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Fatal("Server InitCfg recover, err: ", err)
			os.Exit(0)
		}
	}()

	s.InvokeFunc = make(map[uint32]base.FProtoMsgHandler)
	s.Args = make(map[string]string)
	s.ConfFile = "output/res/server.conf"
	s.DataDir = "output/res/data/"
	s.LogDir = "log"
	global.Env = global.ENV_PC

	s.Helper()
	s.AnalysisArgs()
	s.initRedisCenter()
	if err := s.LoadConf(); err != nil {
		return err
	}
	// 运行参数配置会覆盖server.conf配置
	if err := s.InitSrvArgs(); err != nil {
		return err
	}
	if err := s.InitLog(); err != nil {
		return err
	}

	if global.IsCloud {
		if len(conf.GConf().SrvAddr.HTTPAddr) > 0 {
			global.Gateway = conf.GConf().SrvAddr.HTTPAddr[0]
			global.TcpAddr = conf.GConf().SrvAddr.TCPAddr[0]
		}
	} else {
		localIP, err := utils.ExternalIP()
		if err != nil {
			logger.Warn("ExternalIP failed")
		}

		// 网关地址
		if global.Gateway == "" {
			if localIP == "" {
				logger.Fatal("global.Gateway nil")
			}
			global.Gateway = fmt.Sprintf("http://%s", localIP)
		}

		// 长链接地址
		if global.TcpAddr == "" {
			if localIP == "" {
				logger.Fatal("global.TcpAddr nil")
			}
			global.TcpAddr = localIP
		}

	}

	logger.Info(s.Info())
	s.pack = tcpx.NewPackx(tcpx.ProtobufMarshaller{})
	return nil
}

func (s *Server) initRedisCenter() {
	logger.Debugf("配置中心redis配置: %s:%s", global.RdsCfgCenterHost, global.RdsCfgCenterPass)
	if global.RdsCfgCenterHost != "" && global.RdsCfgCenterPass != "" {
		pass, err := base64.StdEncoding.DecodeString(global.RdsCfgCenterPass)
		if err != nil {
			logger.Fatalf("decode password %s err:", err, global.RdsCfgCenterPass)
			return
		}
		opts := &redis.Options{
			Addr:     global.RdsCfgCenterHost,
			Password: string(pass),
		}
		s.RedisCenter = redis.NewClient(opts)
		if s.RedisCenter == nil {
			logger.Fatalf("配置中心连接失败 host: %+v", opts)
		}
		pong := s.RedisCenter.Ping(context.Background())
		if pong.Err() != nil {
			logger.Fatalf("配置中心初始化失败 %v", pong.Err())
		}
		logger.Infof("配置中心redis链接ping: ", pong.Val())
	}
}

func (s *Server) LoadConf(params ...string) error {
	// 从配置中心读取
	if !global.IsDev && s.RedisCenter != nil {
		key := fmt.Sprintf("%s:%s:%s:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, "aniwar", global.ROLLING_VERSION, "server.conf")
		fmt.Println("配置中心server.conf key: ", key)
		stringCmd := s.RedisCenter.Get(context.Background(), key)
		if stringCmd.Val() != "" {
			params = append(params, stringCmd.Val())
			fmt.Println("拉取到配置中心server.conf val: ", params)
		}
	}

	// 运行参数配置会覆盖默认配置
	err := conf.LoadConf(s.ConfFile, params...)
	if err != nil {
		return fmt.Errorf("LoadConf: %+v", err)
	}
	// 默认日志等是Debug
	if conf.GConf().Base.LogLevel == "" {
		conf.GConf().Base.LogLevel = "debug"
	}
	logLevel, err := zapcore.ParseLevel(conf.GConf().Base.LogLevel)
	// 解析出错时，不改变日志等级
	if err == nil {
		logger.ResetLogLevel(logLevel)
	}
	s.IsMetric = conf.GConf().Base.Metric
	if global.IsCloud && conf.GConf().Base.LogDir != "" {
		s.LogDir = conf.GConf().Base.LogDir
	}
	// s.GateIP = conf.GConf().Base.GateIP
	// s.GatePort = conf.GConf().Base.GatePort

	if conf.GConf().Base.Version != "" && conf.GConf().Base.VersionCheck {
		s.version = ParseVersion(conf.GConf().Base.Version)
	}

	return nil
}

// 热重载配置
func (s *Server) ReloadConf() error {
	if err := s.LoadConf(); err != nil {
		return err
	}
	// 运行参数配置会覆盖server.conf配置
	s.ResetSrvArgs()

	return nil
}

// 加载excel配置表数据
func (s *Server) LoadExcelData() (err error) {
	if global.IsDev || s.RedisCenter == nil {
		// 加载本地配置表
		err = s.LoadExcel()
	} else {
		// 加载配置中心配置表
		err = s.LoadExcelFromRedis()
	}
	return
}

// 按需加载excel配置表数据
func (s *Server) LoadExcelDataByFiles(files []string) (err error) {
	if global.IsDev || s.RedisCenter == nil {
		err = data.LoadByFileNames(s.DataDir, files, s.AppId, "actorserver")
	} else {
		err = s.LoadExcelFromRedisByFile(files, s.AppId, "actorserver")
	}
	return
}

// 加载本地策划配置数据
func (s *Server) LoadExcel() error {
	logger.Info("\n===>>> LoadExcel begin")

	logger.Infof("===>>> 加载模式为: %d, (1:读取压缩文件, 0:读取json源文件)", baseconf.GetBaseConf().ExcelDataZip)

	err := gmeta.GetMetaMgr().LoadAllMeta()
	if err != nil {
		return err
	}

	logger.Info("===>>> LoadExcel end\n")

	return nil
}

// 加载配置中心策划配置数据
func (s *Server) LoadExcelFromRedis() error {
	logger.Info("\n===>>> LoadExcel from redis  begin")
	fileNames := s.GetAllExcelFileName()
	for _, fileName := range fileNames {
		key := fmt.Sprintf("%s:%s:%s:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, "aniwar", global.ROLLING_VERSION, fileName)
		stringCmd := s.RedisCenter.Get(context.Background(), key)
		value, err := stringCmd.Bytes()
		if err != nil {
			logger.Errorf("===>>> LoadExcelFromRedis key:%s fail:%v", key, err)
			return err
		}
		if err = data.LoadByFileData(fileName, value); err != nil {
			logger.Errorf("===>>> LoadExcelFromRedis file:%s fail:%v", fileName, err)
			return err
		}
	}

	logger.Info("===>>> LoadExcel from redis end\n")
	return nil
}

func (s *Server) LoadExcelFromRedisByFile(fileNames []string, appId, serverName string) error {
	successArr := make([]string, 0) // 成功的文件列表
	errorArr := make([]string, 0)   // 失败的文件列表
	logger.Infof("BeginLoad fileNames:%+v appId:%s serverName:%s", fileNames, appId, serverName)
	for _, fileName := range fileNames {
		key := fmt.Sprintf("%s:%s:%s:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, "aniwar", global.ROLLING_VERSION, fileName)
		stringCmd := s.RedisCenter.Get(context.Background(), key)
		value, err := stringCmd.Bytes()
		if err != nil {
			errorArr = append(errorArr, fileName)
			continue
		}
		if err = data.LoadByFileData(fileName, value); err != nil {
			errorArr = append(errorArr, fileName)
			logger.Errorf("LoadFail fileNames:%+v err:%v key:%s", fileNames, err, key)

		} else {
			successArr = append(successArr, fileName)
			logger.Infof("LoadSuccess fileNames:%+v", fileNames)
		}
	}

	// 服务热更新埋点
	taptap.ServeReloadComm(appId, global.APP_VERSION, "", global.ROLLING_VERSION, serverName, taptap.ConvertList2Str(successArr), taptap.ConvertList2Str(errorArr))

	if len(errorArr) > 0 {
		str := fmt.Sprintf("EndLoad errorArr:%+v, successArr:%+v", errorArr, successArr)
		logger.Error(str)
		return fmt.Errorf(str)
	}

	logger.Infof("EndLoad fileNames:%+v successArr:%+v", fileNames, successArr)
	return nil
}

// 获取策划配置文件名
func (s *Server) GetAllExcelFileName() []string {
	if baseconf.GetBaseConf().ExcelDataZip == 1 {
		return data.GetAllJsonBinaryFileNames()
	} else if baseconf.GetBaseConf().ExcelDataZip == 0 {
		return data.GetAllJsonFileNames()
	}
	return []string{}
}

// 加载屏蔽词数据
func (s *Server) LoadWordCfg() error {
	logger.Info("===>>> LoadDirtyWords begin")

	count, err := wordfilter.LoadSensitiveWordCfg(baseconf.GetBaseConf().DirtyWords)
	if err != nil {
		logger.Errorf("LoadDirtyWords err:%v", err)
		return err
	}

	logger.Infof("===>>> LoadDirtyWords end, count:%d", count)
	return nil
}

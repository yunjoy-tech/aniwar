package server

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/gmeta"
	"gitee.com/aniwar2/musae/gamelib/sensitive"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/tcpx"
)

// 启动流程2: Server
// server初始化
func (s *Server) Init() error {
	defer func() {
		if err := recover(); err != nil {
			logger.Fatal("Server InitCfg recover, err: ", err)
		}
	}()
	// 解析运行参数
	s.AnalysisArgs()
	// 初始化运行参数
	if err := s.InitRunArgs(); err != nil {
		return err
	}
	// 初始化远程配置中心
	s.InitApolloConfigCenter(s.Args["config-center"])
	// 加载程序配置文件
	if err := s.LoadConf(); err != nil {
		return err
	}
	// 初始化日志输出
	if err := s.InitLog(); err != nil {
		return err
	}

	if conf.Base().Cloud {
		if len(conf.SrvAddr().HTTPAddr) > 0 {
			global.Gateway = conf.SrvAddr().HTTPAddr[0]
			global.TcpAddr = conf.SrvAddr().TCPAddr[0]
		}
	} else {
		// TODO
		// 网关地址
		if global.Gateway == "" {
			global.Gateway = fmt.Sprintf("http://%s", conf.SrvAddr().TCPAddr[0])
		}

		// 长链接地址
		if global.TcpAddr == "" {
			global.TcpAddr = conf.SrvAddr().TCPAddr[0]
		}
	}

	logger.Info(s.BasicInfo())
	s.pack = tcpx.NewPackx(tcpx.ProtobufMarshaller{})
	return nil
}

func (s *Server) LoadConf() error {
	// TODO 从配置中心读取
	// if /*!global.IsDev &&*/ s.RedisCenter != nil {
	// 	key := fmt.Sprintf("%s:%s:%s:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, "aniwar", global.ROLLING_VERSION, "server.conf")
	// 	fmt.Println("配置中心server.conf key: ", key)
	// 	stringCmd := s.RedisCenter.Get(context.Background(), key)
	// 	if stringCmd.Val() != "" {
	// 		// params = append(params, stringCmd.Val())
	// 		// fmt.Println("拉取到配置中心server.conf val: ", params)
	// 	}
	// }

	// 运行参数配置会覆盖默认配置
	err := conf.LoadConf(s.Args["config"])
	if err != nil {
		return fmt.Errorf("LoadConf: %+v", err)
	}

	if conf.Base().Version != "" && conf.Base().VersionCheck {
		s.version = ParseVersion(conf.Base().Version)
	}

	return nil
}

// 热重载配置
func (s *Server) ReloadConf() error {
	if err := s.LoadConf(); err != nil {
		return err
	}
	// todo 参数热更新
	// 运行参数配置会覆盖server.conf配置
	// s.ResetSrvArgs()

	return nil
}

// 加载excel配置表数据
func (s *Server) LoadExcelData() (err error) {
	if conf.Base().IsDebug /*|| s.RedisCenter == nil */ {
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
	if conf.Base().IsDebug /*|| s.RedisCenter == nil*/ {
		// TODO
		// err = data.LoadByFileNames(conf.Base().MetaDir, files, s.AppId, "actorserver")
	} else {
		err = s.LoadExcelFromRedisByFile(files, s.AppId, "actorserver")
	}
	return
}

// 加载本地策划配置数据
func (s *Server) LoadExcel() error {
	logger.Info("\n===>>> LoadExcel begin")

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
	// fileNames := s.GetAllExcelFileName()
	// for _, fileName := range fileNames {
	// key := fmt.Sprintf("%s:%s:%s:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, "aniwar", global.ROLLING_VERSION, fileName)
	// stringCmd := s.RedisCenter.Get(context.Background(), key)
	// value, err := stringCmd.Bytes()
	// if err != nil {
	// 	logger.Errorf("===>>> LoadExcelFromRedis key:%s fail:%v", key, err)
	// 	return err
	// }
	// // TODO
	// // err = data.LoadByFileData(fileName, value)
	// if err != nil {
	// 	logger.Errorf("===>>> LoadExcelFromRedis file:%s fail:%v", fileName, err)
	// 	return err
	// }
	// }

	logger.Info("===>>> LoadExcel from redis end\n")
	return nil
}

func (s *Server) LoadExcelFromRedisByFile(fileNames []string, appId, serverName string) error {
	successArr := make([]string, 0) // 成功的文件列表
	errorArr := make([]string, 0)   // 失败的文件列表
	logger.Infof("BeginLoad fileNames:%+v appId:%s serverName:%s", fileNames, appId, serverName)
	// for _, fileName := range fileNames {
	// 	key := fmt.Sprintf("%s:%s:%s:%s:%s", global.RdsCfgNameSpace, global.RdsCfgGroup, "aniwar", global.ROLLING_VERSION, fileName)
	// 	stringCmd := s.RedisCenter.Get(context.Background(), key)
	// 	value, err := stringCmd.Bytes()
	// 	if err != nil {
	// 		errorArr = append(errorArr, fileName)
	// 		continue
	// 	}
	// 	if err = data.LoadByFileData(fileName, value); err != nil {
	// 		errorArr = append(errorArr, fileName)
	// 		logger.Errorf("LoadFail fileNames:%+v err:%v key:%s", fileNames, err, key)
	// 	} else {
	// 		successArr = append(successArr, fileName)
	// 		logger.Infof("LoadSuccess fileNames:%+v", fileNames)
	// 	}
	// }

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
	// if conf.Base().ExcelDataZip == 1 {
	// 	return data.GetAllJsonBinaryFileNames()
	// } else if conf.Base().ExcelDataZip == 0 {
	// 	return data.GetAllJsonFileNames()
	// }
	return []string{}
}

// 加载屏蔽词数据
func (s *Server) LoadWordCfg() error {
	logger.Info("===>>> LoadDirtyWords begin")

	count, err := sensitive.LoadSensitiveWord(conf.Actor().DirtyWords)
	if err != nil {
		logger.Errorf("LoadDirtyWords err:%v", err)
		return err
	}

	logger.Infof("===>>> LoadDirtyWords end, count:%d", count)
	return nil
}

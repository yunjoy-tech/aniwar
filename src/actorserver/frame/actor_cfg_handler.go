package frame

import (
	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"
	"gitee.com/bychannel/aniwar/src/common/db"
	"gitee.com/bychannel/musae/framework/global"
	"gitee.com/bychannel/musae/framework/logger"
	"github.com/dapr/go-sdk/client"
	"strings"
)

/*func (s *ActorServer) SubConfCenter() error {
	keys := []string{
		db.KeyCfgGlobalServer,
	}
	if err := s.SubscribeConfigCenter(keys, s.HandlerConfEvent); err != nil {
		logger.Errorf("ActorServer: SubConfCenter err :%v", err)
		return err
	}
	return nil
}*/

func (s *ActorServer) HandlerConfEvent(id string, items map[string]*client.ConfigurationItem) {

	// 服务配置事件埋点
	configTemp := make(map[string]string)
	for k, v := range items {
		configTemp[k] = v.Value // 埋点日志用

		logger.Infof("===>>>ConfigUpdate id = %s, key = %s, value = %s", id, k, v.Value)

		switch k {
		// case db.KeyCfgReloadConf: // server.conf热更
		//	err := s.LoadConf(v.Value)
		//	if err != nil {
		//		logger.Errorf("reload --> LoadConf got err:%+v", err)
		//	}

		// case db.KeyCfgReloadExcel: // excel配置热更
		//	var err error
		//	if strings.Compare(v.Value, "all") == 0 {
		//		err = s.LoadExcelData()
		//	} else {
		//		files := strings.Split(v.Value, "|")
		//		if global.IsDev { // 开发模式
		//			err = data.LoadByFileNames(s.DataDir, files, s.AppId, "actorserver")
		//		} else {
		//			err = s.LoadExcelFromRedisByFile(files, s.AppId, "actorserver")
		//		}
		//	}
		//	if err != nil {
		//		logger.Errorf("reload --> LoadExcel got err:%+v", err)
		//	}

		case db.KeyCfgGlobalDirtyWord: // 动态屏蔽词热更
			words := strings.Split(v.Value, "|")
			dynamicWordMgr.AddWord(words...)

		case db.KeyCfgReloadDirtyWord: // 静态屏蔽词更新
			s.Server.LoadWordCfg()

		case db.KeyCfgGlobalCloseFunc:
			s.RegisterCloseFunc()
		}
	}

	// 开始埋点
	taptap.ConfeventComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "actorserver", id, taptap.ConvertMap2Str(configTemp))
}

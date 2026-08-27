package logic

import (
	"github.com/dapr/go-sdk/client"
	"github.com/yunjoy-tech/aniwar/src/common/datalog/taptap"
	"github.com/yunjoy-tech/musae/global"
	"github.com/yunjoy-tech/musae/logger"
)

/*func (s *IDIPServer) SubConfCenter() error {
	keys := []string{
		db.KeyCfgGlobalServer,
	}
	if err := s.SubscribeConfigCenter(keys, s.HandlerConfEvent); err != nil {
		logger.Errorf("IDIPServer: SubConfCenter err :%v", err)
		return err
	}
	return nil
}*/

func (s *IDIPServer) HandlerConfEvent(id string, items map[string]*client.ConfigurationItem) {

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
		// case db.KeyCfgReloadExcel: // excel配置热更:
		//	var err error
		//	if strings.Compare(v.Value, "all") == 0 {
		//		err = s.LoadNeedExcel(nil)
		//	} else {
		//		files := strings.Split(v.Value, "|")
		//		err = s.LoadNeedExcel(files)
		//	}
		//	if err != nil {
		//		logger.Errorf("reload --> LoadExcel got err:%+v", err)
		//	}
		}

	}

	// 开始埋点
	taptap.ConfeventComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "idipserver", id, taptap.ConvertMap2Str(configTemp))
}

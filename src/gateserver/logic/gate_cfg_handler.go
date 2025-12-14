package logic

import (
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"
	"gitee.com/bychannel/aniwar/src/common/db"
	"github.com/dapr/go-sdk/client"
)

/*func (s *GateServer) SubConfCenter() error {
	keys := []string{
		db.KeyCfgGlobalServer,
	}
	if err := s.SubscribeConfigCenter(keys, s.HandlerConfEvent); err != nil {
		logger.Errorf("GateServer: SubConfCenter err :%v", err)
		return err
	}
	return nil
}*/

func (s *GateServer) HandlerConfEvent(id string, items map[string]*client.ConfigurationItem) {
	// 服务配置事件埋点
	configTemp := make(map[string]string)
	for k, v := range items {
		configTemp[k] = v.Value // 埋点日志用

		logger.Infof("===>>>ConfigUpdate id = %s, key = %s, value = %s", id, k, v.Value)

		switch k {
		case db.KeyCfgGlobalDeprecatedMsg:
			s.RegisterDeprecatedMsg()
		case db.KeyCfgServerRollingNotice:
			s.HandlePushRollingNotice(v.Value)
		}

	}

	// 开始埋点
	taptap.ConfeventComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "gateserver", id, taptap.ConvertMap2Str(configTemp))
}

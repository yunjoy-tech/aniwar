package logic

import (
	"github.com/dapr/go-sdk/client"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"strings"
)

func (s *GuideServer) HandlerConfEvent(id string, items map[string]*client.ConfigurationItem) {
	// 服务配置事件埋点
	configTemp := make(map[string]string)
	for k, v := range items {
		configTemp[k] = v.Value // 埋点日志用
		logger.Debugf("===>>>ConfigUpdate id = [%s], key = [%s], value = [%s]", id, k, v.Value)
		switch {
		//case strings.Compare(k, db.KeyCfgReloadConf) == 0: // server.conf热更
		//	err := s.LoadConf(v.Value)
		//	if err != nil {
		//		logger.Error("ServerConfReload, key = [%s], value = [%s], err:[%v]", k, v.Value, err)
		//	}
		case strings.Compare(k, db.KeyCfgCVersionAndroid) == 0: // android version update
			logger.Warnf("VersionUpdate android, key = [%s], value = [%s]", k, v.Value)
		case strings.Compare(k, db.KeyCfgCVersionIOS) == 0: // ios version update
			logger.Warnf("VersionUpdate ios, key = [%s], value = [%s]", k, v.Value)
		default:
			continue
		}
	}
	// 开始埋点
	taptap.ConfeventComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "GuideServer", id, taptap.ConvertMap2Str(configTemp))
}

package logic

import (
	"github.com/dapr/go-sdk/client"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

/*func (s *LobbyServer) SubConfCenter() error {
	keys := []string{
		db.KeyCfgGlobalServer,
	}
	if err := s.SubscribeConfigCenter(keys, s.HandlerConfEvent); err != nil {
		logger.Errorf("LobbyServer: SubConfCenter err :%v", err)
		return err
	}
	return nil
}*/

func (s *LobbyServer) HandlerConfEvent(id string, items map[string]*client.ConfigurationItem) {
	// 服务配置事件埋点
	configTemp := make(map[string]string)
	for k, v := range items {
		configTemp[k] = v.Value // 埋点日志用
		logger.Infof("===>>>ConfigUpdate id = %s, key = %s, value = %s", id, k, v.Value)
	}

	// 开始埋点
	taptap.ConfeventComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "lobbyserver", id, taptap.ConvertMap2Str(configTemp))
}

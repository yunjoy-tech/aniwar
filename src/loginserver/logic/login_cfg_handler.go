package logic

import (
	"github.com/dapr/go-sdk/client"
	"github.com/yunjoy-tech/aniwar/src/common/datalog/taptap"
	"github.com/yunjoy-tech/musae/global"
	"github.com/yunjoy-tech/musae/logger"
)

func (s *LoginServer) HandlerConfEvent(id string, items map[string]*client.ConfigurationItem) {

	// 服务配置事件埋点
	configTemp := make(map[string]string)
	for k, v := range items {
		configTemp[k] = v.Value // 埋点日志用

		logger.Infof("===>>>ConfigUpdate id = %s, key = %s, value = %s", id, k, v.Value)
	}

	// 开始埋点
	taptap.ConfeventComm(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "loginserver", id, taptap.ConvertMap2Str(configTemp))
}

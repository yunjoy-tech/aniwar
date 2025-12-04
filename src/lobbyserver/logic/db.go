package logic

import (
	"errors"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"
	"google.golang.org/protobuf/proto"
)

func (s *LobbyServer) SaveDB(key string, value proto.Message) error {

	kvTable, err := db.BuildKvTable(value, key)
	if err != nil {
		return err
	}

	if baseconf.GetBaseConf().IsDebug {
		kvTable.DataSrc = utils.PrettyJson(value)
	}
	// 保存
	err = s.SaveMongoMail(key, kvTable, nil)
	if err != nil {
		logger.Debug("lobby server SaveDB failed")
	}

	logger.Debugf("lobby server SaveDB, %s", kvTable.Str())
	return nil
}

func (s *LobbyServer) LoadDB(key string, value proto.Message) error {
	if kvTable, err := s.GetMongoMail(key, nil); err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			return nil // db中没有数据
		} else {
			return err
		}
	} else {
		err = proto.Unmarshal(kvTable.Data, value)
		if err != nil {
			return err
		}
	}

	return nil
}

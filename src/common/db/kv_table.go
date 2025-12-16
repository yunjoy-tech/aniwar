package db

import (
	"encoding/json"
	"gitee.com/aniwar2/musae/baseconf"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/utils"
	"github.com/pkg/errors"
	"time"

	"gitee.com/aniwar2/musae/state"
	"google.golang.org/protobuf/proto"
)

// BuildKvTable 构建kv-table
func BuildKvTable(value proto.Message, key string) (*state.KvTable, error) {
	if value == nil {
		return nil, nil
	}
	var (
		nowSec = time.Now().Unix()
	)

	temp, err := proto.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, "SaveDB Marshal err")
	}

	var dataSrc []byte
	if baseconf.GetBaseConf().IsDebug {
		dataSrc, err = json.Marshal(value)
		if err != nil {
			logger.Warnf("SaveDB DataSrc Marshal err:%+v", err.Error())
		}
	}

	// kvtable包装
	kvTable := &state.KvTable{
		Key:     key,
		Id:      0,
		Data:    temp,
		UpSecTS: nowSec,
		InSecTS: 0,
		DataSrc: string(dataSrc),
	}

	logger.Debug("===>>>kvTable: ", kvTable.Str())
	return kvTable, nil
}

func ParseKvTable(kvTable *state.KvTable, value proto.Message) error {
	if kvTable == nil || value == nil {
		return nil
	}

	err := proto.Unmarshal(kvTable.Data, value)
	if err != nil {
		return errors.WithStack(err)
	}
	if baseconf.GetBaseConf().IsDebug {
		kvTable.DataSrc = utils.PrettyJson(value)
	}
	logger.Debug("===>>>kvTable: ", kvTable.Str())
	return nil
}

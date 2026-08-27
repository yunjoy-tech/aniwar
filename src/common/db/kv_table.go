package db

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/utils"
	"time"

	"github.com/yunjoy-tech/musae/state"
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
	if conf.Base().IsDebug {
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

	logger.Debug("===>>>kvTable: ", kvTable.ToString())
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
	if conf.Base().IsDebug {
		kvTable.DataSrc = utils.PrettyJson(value)
	}
	logger.Debug("===>>>kvTable: ", kvTable.ToString())
	return nil
}

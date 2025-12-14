package server

import (
	"context"
	"encoding/json"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"google.golang.org/protobuf/proto"
	"time"

	"gitee.com/aniwar2/musae/framework/baseconf"
	"gitee.com/aniwar2/musae/framework/utils"
	"github.com/pkg/errors"

	"gitee.com/aniwar2/musae/framework/service"

	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/metrics"
	"gitee.com/aniwar2/musae/framework/state"
	dapr "github.com/dapr/go-sdk/client"
)

// SaveMongoGame save to mongo game
func (s *Server) SaveMongoGame(key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
	var (
		err    error
		startT = time.Now()
	)

	err = s.Service.SaveMongo(service.MongoDbType_MongoGame, key, table, meta, so...)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return errors.WithStack(err)
}

// GetMongoGame load from mongo game
func (s *Server) GetMongoGame(key string, meta map[string]string) (*state.KvTable, error) {
	var (
		err     error
		startT  = time.Now()
		kvTable *state.KvTable
	)

	kvTable, err = s.Service.GetMongo(service.MongoDbType_MongoGame, key, meta)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")
	return kvTable, errors.WithStack(err)
}

// SaveMongoMail save to mongo mail
func (s *Server) SaveMongoMail(key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
	var (
		err    error
		startT = time.Now()
	)

	err = s.Service.SaveMongo(service.MongoDbType_MongoMail, key, table, meta, so...)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return err
}

// GetMongoMail load from mongo mail
func (s *Server) GetMongoMail(key string, meta map[string]string) (*state.KvTable, error) {
	var (
		err     error
		startT  = time.Now()
		kvTable *state.KvTable
	)

	kvTable, err = s.Service.GetMongo(service.MongoDbType_MongoMail, key, meta)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return kvTable, err
}

// SaveMongoAccount save to mongo account
func (s *Server) SaveMongoAccount(key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
	var (
		err    error
		startT = time.Now()
	)

	err = s.Service.SaveMongo(service.MongoDbType_MongoAccount, key, table, meta, so...)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return errors.WithStack(err)
}

// GetMongoAccount load from mongo account
func (s *Server) GetMongoAccount(key string, meta map[string]string) (*state.KvTable, error) {
	var (
		err     error
		startT  = time.Now()
		kvTable *state.KvTable
	)

	kvTable, err = s.Service.GetMongo(service.MongoDbType_MongoAccount, key, meta)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return kvTable, errors.WithStack(err)
}

// SaveMongoGmt save to mongo gmt
func (s *Server) SaveMongoGmt(key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
	startT := time.Now()
	err := s.Service.SaveMongo(service.MongoDbType_MongoGmt, key, table, meta, so...)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")
	return err
}

// GetMongoGmt load from mongo gmt
func (s *Server) GetMongoGmt(key string, meta map[string]string) (*state.KvTable, error) {
	startT := time.Now()
	kvTable, err := s.Service.GetMongo(service.MongoDbType_MongoGmt, key, meta)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")
	return kvTable, err
}

// func (s *Server) saveMongo(db MongoDbType, key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
//	data, err := json.Marshal(table)
//	if err != nil {
//		logger.Error("saveMongo Marshal err: ", table, err)
//		return DB_ERROR_MARSHAL
//	}
//
//	dataLen := len(table.Data)
//	ctx := context.Background()
//	now := time.Now()
//	err = s.Daprc.SaveState(ctx, string(db), key, data, meta, so...)
//	if err != nil {
//		logger.Error("saveMongo err: ", err, dataLen)
//		metrics.GaugeInc(metrics.MongoWErr)
//		return DB_ERROR_TIMEOUT
//	}
//	metrics.HistogramPut(metrics.MongoWDelayHist, time.Since(now).Milliseconds(), metrics.Mongo)
//	metrics.GaugeInc(metrics.MongoWCount)
//	metrics.GaugeAdd(metrics.MongoWSize, int64(dataLen))
//	logger.Debugf("SaveMongo db:[%v], key:[%v], kvTable: %v", db, key, table.Str())
//	return nil
// }
//
// func (s *Server) getMongo(db MongoDbType, key string, meta map[string]string) (*state.KvTable, error) {
//
//	ctx := context.Background()
//	now := time.Now()
//	item, err := s.Daprc.GetState(ctx, string(db), key, meta)
//	if err != nil {
//		logger.Error("getMongo GetState err: ", err)
//		metrics.GaugeInc(metrics.MongoRErr)
//		return nil, DB_ERROR_TIMEOUT
//	}
//	metrics.HistogramPut(metrics.MongoRDelayHist, time.Since(now).Milliseconds(), metrics.Mongo)
//	logger.Debugf("getMongo key: %v,len: %v", key, len(item.Value))
//
//	metrics.GaugeInc(metrics.MongoRCount)
//	// 初始状态
//	if len(item.Value) == 0 {
//		logger.Debugf("getMongo KvTable nil,key: %v", key)
//		return nil, DB_ERROR_NOT_EXIST
//	}
//
//	table := &state.KvTable{}
//	err = json.Unmarshal(item.Value, table)
//	if err != nil {
//		logger.Errorf("getMongo Unmarshal KvTable err: %v, %v, %+v", err, key, item)
//		return nil, DB_ERROR_UNMARSHAL
//	}
//	logger.Debugf("getMongo db:%v, key:%v, dataLen:%v", db, key, len(table.Data))
//	return table, nil
// }

// UpsertMongoTableTransaction update or insert to mongo by transaction
func (s *Server) UpsertMongoTableTransaction(db service.MongoDbType, meta map[string]string, kvTableMap map[string]*state.KvTable) error {
	var (
		err    error
		startT = time.Now()
	)

	opts := make([]*dapr.StateOperation, 0)

	for dbKey, kvTable := range kvTableMap {
		data, err := json.Marshal(kvTable)
		if err != nil {
			logger.Errorf("UpsertMongoTableTransaction Marshal err: ", kvTable, err)
			return err
		}

		opt := &dapr.StateOperation{
			Type: dapr.StateOperationTypeUpsert,
			Item: &dapr.SetStateItem{
				Key:      dbKey,
				Value:    data,
				Metadata: meta,
				Options: &dapr.StateOptions{
					Concurrency: dapr.StateConcurrencyLastWrite, // 最终一致性
					Consistency: dapr.StateConsistencyStrong,
				},
			},
		}
		opts = append(opts, opt)
	}

	err = s.SaveMongoTransaction(db, meta, opts)
	if err != nil {
		return err
	}
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "UpsertMongoTableTransaction")

	return err
}

// SaveMongoTransaction save to mongo by transaction
func (s *Server) SaveMongoTransaction(db service.MongoDbType, meta map[string]string, opts []*dapr.StateOperation) error {
	optCount := len(opts)
	if optCount <= 0 {
		return nil
	}

	var (
		err    error
		startT = time.Now()
	)

	ctx, _ := context.WithTimeout(context.Background(), global.DB_INVOKE_TIMEOUT*time.Second)
	now := time.Now()
	logger.Debugf("SaveMongoTransaction === db:%v, meta:%v, opts:%v", db, meta, opts)
	// err = s.Daprc.ExecuteStateTransaction(ctx, string(db), meta, opts)
	_, err = utils.RetryDoSyncInterval(
		baseconf.GetBaseConf().AniwarDbSetRetryCount,
		baseconf.GetBaseConf().AniwarDbRetryInterval,
		func() (any, error) {
			return nil, s.Daprc.ExecuteStateTransaction(ctx, string(db), meta, opts)
		})
	if err != nil {
		logger.Error("SaveMongoTransaction err: ", err, optCount)
		metrics.GaugeInc(metrics.MongoWErr)
		return service.DB_ERROR_TIMEOUT
	}
	metrics.HistogramPut(metrics.MongoWDelayHist, time.Since(now).Milliseconds(), metrics.Mongo)
	metrics.GaugeInc(metrics.MongoWCount)
	metrics.GaugeAdd(metrics.MongoWSize, int64(optCount))
	logger.Debugf("SaveMongoTransaction db:[%v], opts: %v, meta: %v", db, opts, meta)

	logger.WarnDelayf(time.Since(startT).Milliseconds(), "SaveMongoTransaction")
	return nil
}

func (s *Server) SaveSystemMail(value proto.Message) error {
	kvTable, err := db.BuildKvTable(value, db.KeySystemMail())
	if err != nil {
		return err
	}

	if baseconf.GetBaseConf().IsDebug {
		kvTable.DataSrc = utils.PrettyJson(value)
	}
	// 保存
	err = s.SaveMongoGame(db.KeySystemMail(), kvTable, nil)
	if err != nil {
		logger.Errorf("SaveSystemMail got err: %v", err)
	}

	logger.Infof("保存系统邮件数据 data: %+v", value)
	return nil
}

func (s *Server) GetSystemMail(systemMail *pb.PSystemMailInfo) error {
	if systemMail.SystemMail == nil {
		systemMail.SystemMail = make(map[int64]*pb.PSysMailInfo)
	}
	if kvTable, err := s.GetMongoGame(db.KeySystemMail(), nil); err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			return nil // db中没有数据
		} else {
			return err
		}
	} else {
		err = proto.Unmarshal(kvTable.Data, systemMail)
		if err != nil {
			return err
		}
		// nil容错
		if systemMail.SystemMail == nil {
			systemMail.SystemMail = make(map[int64]*pb.PSysMailInfo)
		}
	}
	logger.Infof("加载系统邮件数据 data: %+v", systemMail)
	return nil
}

package server

import (
	"context"
	"encoding/json"
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

func (s *Server) SaveGlobalRedis(key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
	return errors.WithStack(s.Service.SaveRedis(service.RedisGlobal, key, table, meta, so...))
}

func (s *Server) GetGlobalRedis(key string, meta map[string]string) (*state.KvTable, error) {
	return s.Service.GetRedis(service.RedisGlobal, key, meta)
}

func (s *Server) SaveCacheRedis(key string, table *state.KvTable, meta map[string]string, so ...dapr.StateOption) error {
	return errors.WithStack(s.Service.SaveRedis(service.RedisCache, key, table, meta, so...))
}

func (s *Server) GetCacheRedis(key string, meta map[string]string) (*state.KvTable, error) {
	return s.Service.GetRedis(service.RedisCache, key, meta)
}

// UpsertRedisTableTransaction update or insert to redis by transaction
func (s *Server) UpsertRedisTableTransaction(db service.RedisDbType, meta map[string]string, kvTableMap map[string]*state.KvTable) error {
	opts := make([]*dapr.StateOperation, 0)

	if len(kvTableMap) == 0 {
		return nil
	}

	var (
		err    error
		startT = time.Now()
	)

	for dbKey, kvTable := range kvTableMap {
		data, err := json.Marshal(kvTable)
		if err != nil {
			logger.Errorf("UpsertRedisTableTransaction Marshal err: ", kvTable, err)
			return err
		}

		opt := &dapr.StateOperation{
			Type: dapr.StateOperationTypeUpsert,
			Item: &dapr.SetStateItem{
				Key:      dbKey,
				Value:    data,
				Metadata: meta,
				Options: &dapr.StateOptions{
					Concurrency: dapr.StateConcurrencyFirstWrite,
					Consistency: dapr.StateConsistencyStrong,
				},
			},
		}
		opts = append(opts, opt)
	}

	err = s.SaveRedisTransaction(db, meta, opts)
	if err != nil {
		return err
	}
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "UpsertMongoTableTransaction")

	return nil
}

// SaveRedisTransaction save to redis by transaction
func (s *Server) SaveRedisTransaction(db service.RedisDbType, meta map[string]string, opts []*dapr.StateOperation) error {
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
	// err := s.Daprc.ExecuteStateTransaction(ctx, string(db), meta, opts)
	_, err = utils.RetryDoSyncInterval(
		baseconf.GetBaseConf().AniwarDbSetRetryCount,
		baseconf.GetBaseConf().AniwarDbRetryInterval,
		func() (any, error) {
			return nil, s.Daprc.ExecuteStateTransaction(ctx, string(db), meta, opts)
		})
	if err != nil {
		logger.Error("SaveRedisTransaction err: ", err, optCount)
		metrics.GaugeInc(metrics.RedisWErr)
		return service.DB_ERROR_TIMEOUT
	}
	metrics.HistogramPut(metrics.RedisWDelayHist, time.Since(now).Milliseconds(), metrics.Redis)
	metrics.GaugeInc(metrics.RedisWCount)
	metrics.GaugeAdd(metrics.RedisWSize, int64(optCount))
	logger.Debugf("SaveRedisTransaction db:[%v], opts: %v, meta: %v", db, opts, meta)
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "SaveRedisTransaction")

	return nil
}

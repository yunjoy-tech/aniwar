package server

import (
	"context"
	"encoding/json"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/state"
	dapr "github.com/dapr/go-sdk/client"
	"google.golang.org/protobuf/proto"
)

type ICache interface {
	SetFromDbIfNotCacheHandler(service.MongoDbType, string, *state.KvTable) error   // 指定缓存数据的落地库
	GetFromDbIfNotCacheHandler(service.MongoDbType, string) (*state.KvTable, error) // 指定缓存中数据的加载源
}

// 落地到mongo和redis缓存
func (s *Server) SaveMongoAndRedis(mongoDbType service.MongoDbType, key string, table *state.KvTable, meta map[string]string, icache ICache, so ...dapr.StateOption) error {
	if key == "" {
		return nil
	}

	data, err := json.Marshal(table)
	if err != nil {
		logger.Errorf("SaveMongo Marshal err: ", table, err)
		return service.DB_ERROR_MARSHAL
	}

	if icache != nil {
		err = icache.SetFromDbIfNotCacheHandler(mongoDbType, key, table)
		if err != nil {
			return err
		}
	}

	if meta == nil {
		meta = make(map[string]string)
	}

	// 过期时间
	if _, ok := meta[service.REDIS_TTL_NAME]; !ok {
		meta[service.REDIS_TTL_NAME] = conf.Base().UserActorGCTime
	}

	err = s.SaveRedis(service.RedisCache, key, table, meta, so...)
	if err != nil {
		logger.Error("SaveMongoAndRedis err: ", err, key, data)
		return service.DB_ERROR_TIMEOUT
	}

	logger.Debug("SaveMongoAndRedis:", key, len(data), meta)
	return nil
}

// CacheRedisKeyExist redis中缓存是否存在
func (s *Server) CacheRedisKeyExist(key string, message proto.Message) (*state.KvTable, bool) {
	reply, err := s.getCacheOnlyFromRedis(key, nil)
	if err != nil {
		// if err != nil {
		return nil, false
	}

	if reply.Data == nil {
		return reply, false
	}

	if message != nil {
		err = proto.Unmarshal(reply.Data, message)
		if err != nil {
			return nil, false
		}
	}

	return reply, true
}

// GetCache 从redis缓存中获取，没有就从mongo查找并写入缓存
func (s *Server) GetCache(mongoDbName service.MongoDbType, key string, iCache ICache) (*state.KvTable, error) {
	var (
		ok      bool // key是否在redis中存在
		err     error
		kvTable *state.KvTable
	)

	if kvTable, ok = s.CacheRedisKeyExist(key, nil); !ok && iCache != nil {
		// 缓存失效，从mongo中获取数据
		// kvTable, err = s.GetMongoGame(key, nil)
		kvTable, err = iCache.GetFromDbIfNotCacheHandler(mongoDbName, key)
		// if err != nil || kvTable == nil || kvTable.Data == nil || len(kvTable.Data) == 0 {
		if err != nil {
			return nil, err
		}

		// 缓存到redis
		err = s.SaveMongoAndRedis(mongoDbName, key, kvTable, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	// logger.Infof("UserActor LoadDB ret: %v, %, %v", err, key, utils.PrettyJson(value))
	return kvTable, nil
}

func (s *Server) getCacheOnlyFromRedis(key string, meta map[string]string) (*state.KvTable, error) {
	ctx := context.Background()
	item, err := s.Daprc.GetState(ctx, string(service.RedisCache), key, meta)
	if err != nil {
		logger.Error("GetRedisCache GetState err: ", err)
		return nil, service.DB_ERROR_TIMEOUT
	}
	logger.Debugf("GetRedisCache key: %v, len: %d", key, len(item.Value))

	// 初始状态
	if len(item.Value) == 0 {
		return nil, service.DB_ERROR_NOT_EXIST
	}

	table := &state.KvTable{}
	err = json.Unmarshal(item.Value, table)
	if err != nil {
		logger.Errorf("GetRedisCache Unmarshal KvTable err: %v, %v, %+v", err, key, item)
		return nil, service.DB_ERROR_UNMARSHAL
	}
	return table, nil
}

func (s *Server) GetCacheOnlyFromRedis(key string, meta map[string]string, msg proto.Message) (*state.KvTable, error) {
	kvTable, err := s.getCacheOnlyFromRedis(key, meta)
	if err != nil {
		return nil, err
	}

	err = db.ParseKvTable(kvTable, msg)
	if err != nil {
		return kvTable, err
	}

	return kvTable, err
}

package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"gitlab.musadisca-games.com/wangxw/musae/framework/state"
	"google.golang.org/protobuf/proto"
)

func (s *Server) SaveAccount(account *cmd.UserData) error {
	temp, err := proto.Marshal(account)
	if err != nil {
		return err
	}

	var dataSrc []byte
	if baseconf.GetBaseConf().IsDebug {
		dataSrc, err = json.Marshal(account)
		if err != nil {
			logger.Warnf("SaveDB DataSrc Marshal err:%+v", err.Error())
		}
	}

	kvTable := &state.KvTable{
		Id:      account.PlayerList.PlayerId,
		UID:     account.Account.Uid,
		Data:    temp,
		UpSecTS: time.Now().Unix(),
		InSecTS: account.CreateTs,
		DataSrc: string(dataSrc),
	}

	err = s.SaveMongoAccount(db.KeyAccountInfo(account.Account.Uid), kvTable, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetAccount(key string) (*cmd.UserData, error) {
	// 查询db中的数据
	kvTable, err := s.GetMongoAccount(key, nil)
	if err != nil {
		return nil, err
	}
	account := &cmd.UserData{}
	if kvTable != nil {
		if err = proto.Unmarshal(kvTable.Data, account); err != nil {
			return nil, service.DB_ERROR_MARSHAL
		}
	}
	err = db.ParseKvTable(kvTable, account)
	if err != nil {
		return nil, service.DB_ERROR_MARSHAL
	}
	return account, nil
}

func (s *Server) SAdd(ctx context.Context, key string, ttl int, vals ...interface{}) error {
	var (
		expireRet *redis.BoolCmd
	)
	var pipe redis.Pipeliner
	if global.IsCloud {
		pipe = s.RedisCluster.TxPipeline()
	} else {
		pipe = s.Redis.TxPipeline()
	}

	addRet := pipe.SAdd(ctx, key, vals...)
	if ttl > 0 {
		expireRet = pipe.Expire(ctx, key, time.Duration(ttl)*time.Second)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	if addRet.Err() != nil {
		return errors.Errorf("redis SAdd error:%v,key:%s, vals:%+v", addRet.Err(), key, vals)
	}
	if expireRet.Val() == false || expireRet.Err() != nil {
		return errors.Errorf("redis expire error:%v,key:%s, vals:%+v", expireRet.Err(), key, vals)
	}

	//ret := s.Redis.SAdd(ctx, key, vals...)
	//if ret.Err() != nil {
	//	return errors.Errorf("redis SAdd error:%v,key:%s, vals:%+v", ret.Err(), key, vals)
	//}
	//if ttl > 0 {
	//	if ret := s.Redis.RedisExpire(ctx, key, time.Duration(ttl)*time.Second); ret.Val() == false || ret.Err() != nil {
	//		return errors.Errorf("redis expire error:%v,key:%s, vals:%+v", ret.Err(), key, vals)
	//	}
	//}
	return nil
}

func (s *Server) SPopN(ctx context.Context, key string, num int64) ([]string, error) {
	var ret *redis.StringSliceCmd
	if global.IsCloud {
		ret = s.RedisCluster.SPopN(ctx, key, num)
	} else {
		ret = s.Redis.SPopN(ctx, key, num)
	}

	return ret.Val(), ret.Err()
}

func (s *Server) ZAdd(ctx context.Context, key string, members ...*redis.Z) (int64, error) {
	var ret *redis.IntCmd
	if global.IsCloud {
		ret = s.RedisCluster.ZAdd(ctx, key, members...)
	} else {
		ret = s.Redis.ZAdd(ctx, key, members...)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) ZCard(ctx context.Context, key string) (int64, error) {
	var ret *redis.IntCmd
	if global.IsCloud {
		ret = s.RedisCluster.ZCard(ctx, key)
	} else {
		ret = s.Redis.ZCard(ctx, key)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) (int64, error) {
	var ret *redis.IntCmd
	if global.IsCloud {
		ret = s.RedisCluster.ZRemRangeByRank(ctx, key, start, stop)
	} else {
		ret = s.Redis.ZRemRangeByRank(ctx, key, start, stop)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	var ret *redis.StringSliceCmd
	if global.IsCloud {
		ret = s.RedisCluster.ZRangeByScore(ctx, key, opt)
	} else {
		ret = s.Redis.ZRangeByScore(ctx, key, opt)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error) {
	var ret *redis.ZSliceCmd
	if global.IsCloud {
		ret = s.RedisCluster.ZRangeByScoreWithScores(ctx, key, opt)
	} else {
		ret = s.Redis.ZRangeByScoreWithScores(ctx, key, opt)
	}
	return ret.Val(), ret.Err()
}

// RedisExpire 设置key的过期时间
func (s *Server) RedisExpire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	var ret *redis.BoolCmd
	if global.IsCloud {
		ret = s.RedisCluster.Expire(ctx, key, expiration)
	} else {
		ret = s.Redis.Expire(ctx, key, expiration)
	}
	return ret.Val(), ret.Err()
}

// RedisDel 删除key
// return 返回删除key的数量
func (s *Server) RedisDel(ctx context.Context, key ...string) (int64, error) {
	var ret *redis.IntCmd
	if global.IsCloud {
		ret = s.RedisCluster.Del(ctx, key...)
	} else {
		ret = s.Redis.Del(ctx, key...)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) RedisSet(ctx context.Context, key, val string, expiration time.Duration) (string, error) {
	var ret *redis.StatusCmd
	if global.IsCloud {
		ret = s.RedisCluster.Set(ctx, key, val, expiration)
	} else {
		ret = s.Redis.Set(ctx, key, val, expiration)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) RedisGet(ctx context.Context, key string) (string, error) {
	var ret *redis.StringCmd
	if global.IsCloud {
		ret = s.RedisCluster.Get(ctx, key)
	} else {
		ret = s.Redis.Get(ctx, key)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) RedisBitCount(ctx context.Context, key string, bitCount *redis.BitCount) (int64, error) {
	var ret *redis.IntCmd
	if global.IsCloud {
		ret = s.RedisCluster.BitCount(ctx, key, bitCount)
	} else {
		ret = s.Redis.BitCount(ctx, key, bitCount)
	}
	return ret.Val(), ret.Err()
}

func (s *Server) RedisSetBit(ctx context.Context, key string, offset int64, val int) (int64, error) {
	var ret *redis.IntCmd
	if global.IsCloud {
		ret = s.RedisCluster.SetBit(ctx, key, offset, val)
	} else {
		ret = s.Redis.SetBit(ctx, key, offset, val)
	}
	return ret.Val(), ret.Err()
}

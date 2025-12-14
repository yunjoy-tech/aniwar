package mailactor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/aniwar2/musae/framework/threading"
	"strconv"
	"time"

	"gitee.com/aniwar2/aniwar/src/actorserver/frame"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/baseactor"
	"gitee.com/aniwar2/musae/framework/baseconf"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/service"
	svc "gitee.com/aniwar2/musae/framework/service"
	"gitee.com/aniwar2/musae/framework/state"
	"github.com/dapr/go-sdk/actor"
	_ "github.com/dapr/go-sdk/actor"
)

type MailData struct {
	Data *pb.PSystemMailInfo
}

type MailActor struct {
	*frame.CommonActor
	MailData

	MailHandler *MailHandler
}

func New() actor.Server {
	a := &MailActor{
		CommonActor: frame.NewCommonActor(frame.GSrv),
	}
	a.ActorType = global.MailActorType
	a.SetActor(a)
	a.Srv = frame.GSrv
	// a.MsgFunc = make(map[int32]base.FProtoMsgHandler)
	a.HandlersMap = make(map[svc.MongoDbType][]baseactor.IBaseHandler, 0)

	// 协议注册
	a.initHandlers()

	return a
}

func (s *MailActor) SetID(id string) {
	s.ServerImplBase.SetID(id)
}

func (s *MailActor) Activate(invokeName string) error {
	defer func() {
		if err := recover(); err != any(nil) {
			s.Trace("MailActor.SaveState recover, err: ", err)
		}
	}()

	s.ReloadActorFromRedis(global.MailActorType)

	// 内存中没有数据
	if s.Data == nil {
		err := s.loadAllData()
		if err != nil {
			return err
		}
	}

	s.Infof("=================>MailActor Activate [%s] [%s]<=================", invokeName, s.ID())

	return nil
}

func (s *MailActor) Deactivate() error {
	s.Infof("=================>MailActor Deactivate [%s]<=================", s.ID())

	threading.RunSafe(func() {
		s.SaveActor2Redis(global.MailActorType)
	})

	return nil
}

func (s *MailActor) initHandlers() {
	s.MailHandler = NewMailHandler(s)
	s.KeepHandler(s.MailHandler)

}

// 其他mailactor更新了，从redis中加载数据到内存
func (s *MailActor) LoadNewMail(ctx context.Context, params []byte) error {
	return s.loadAllData()
}

// commit2Redis
//
//	@Description: 单独一个请求修改的数据提交到redis
//	@receiver s
func (s *MailActor) commit2Redis() error {
	var err error
	s.Debugf("commit2Redis UserActor, %s", s.ID())
	cacheMap := make(map[string]*pb.CacheKeyDataEx, 0)
	kvTableMap := make(map[string]*state.KvTable, 0)
	for mongoType, handlers := range s.HandlersMap {
		for _, handler := range handlers {
			if handler.IsRedisDirty() {
				_, dbKey, dbVal := handler.DBTable()

				kvTable, err := db.BuildKvTable(dbVal, dbKey)
				if err != nil {
					return err
				}

				if _, ok := kvTableMap[dbKey]; ok {
					return errors.New(fmt.Sprintf("重复的数据, dbKey=%s", dbKey))
				}

				kvTableMap[dbKey] = kvTable
				cacheMap[dbKey] = &pb.CacheKeyDataEx{
					Key: dbKey,
					// DataLen:     int32(len(kvTable.Data)),
					MongoDBType: string(mongoType),
				}
			}
		}

	}

	meta := map[string]string{"ttlInSeconds": strconv.Itoa(s.getCacheTTL())} // 过期时间
	err = s.Srv.UpsertRedisTableTransaction(svc.RedisCache, meta, kvTableMap)
	if err != nil {
		return err
	}

	err = s.saveCacheKeyEx(cacheMap, s.getCacheTTL()) // gc后再保留600s
	if err != nil {
		return err
	}

	// 提交成功, 清除标记
	if err == nil {
		for _, handlers := range s.HandlersMap {
			for _, handler := range handlers {
				handler.CleanRedisDirty()
			}
		}
	}

	return nil
}

func (s *MailActor) getCacheTTL() int {
	// 设置延迟同步的key
	gcTime, err := strconv.Atoi(baseconf.GetBaseConf().UserActorGCTime)
	if err != nil {
		gcTime = 600 // 默认600s秒
	}

	return gcTime*2 + 600 // gc后再保留600s
}

// 设置延迟同步的key
func (s *MailActor) saveCacheKeyEx(cacheMap map[string]*pb.CacheKeyDataEx, ttl int) error {
	var (
		err error
	)
	if len(cacheMap) == 0 {
		return nil
	}

	cacheKey := db.KeyCacheRedisData(s.ID())

	vals := make([]interface{}, 0)
	var val []byte
	for _, cacheData := range cacheMap {
		val, err = json.Marshal(cacheData)
		if err != nil {
			s.Errorf("--->>>CacheKey [%+v] Marshal error:%+v", cacheData, err)
			continue
		}
		if len(val) == 0 {
			continue
		}
		vals = append(vals, string(val))
	}
	if len(vals) > 0 {
		err = s.Srv.SAdd(context.Background(), cacheKey, ttl, vals...)
		if err != nil {
			return err
		}
	}
	logger.Debugf("--->>>write writeCacheKey=%v, 新增同步key:%v", cacheKey, vals)

	return nil
}

func (s *MailActor) loadAllData() error {
	var (
		err    error
		startT = time.Now()
	)

	mongoDBs := []service.MongoDbType{
		// service.MongoDbType_MongoAccount, // 账号db
		service.MongoDbType_MongoGame, // 游戏db
	}

	for _, eachDB := range mongoDBs {
		if err = s.loadDBDataByDBType(eachDB); err != nil {
			return err
		}
	}

	s.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return nil
}

// 全量加载用户数据
func (s *MailActor) loadDBDataByDBType(dbType service.MongoDbType) error {
	for _, handler := range s.HandlersMap[dbType] {
		dbTable, dbKey, dbVal := handler.DBTable()
		err := handler.LoadDBData(dbTable, dbKey, dbVal)
		if err != nil {
			return err
		}
	}
	return nil
}

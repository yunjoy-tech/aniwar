package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/baseactor"
	"gitee.com/aniwar2/musae/baseconf"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/mtime"
	"gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/state"
	"gitee.com/aniwar2/musae/utils"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"strconv"
	"time"
)

type CommonActor struct {
	baseactor.BaseActor
	Srv *ActorServer
	pb.ActorInfo
	Timer *mtime.TimeWheel
}

func NewCommonActor(actorServer *ActorServer) *CommonActor {
	commonActor := &CommonActor{
		Srv:   actorServer,
		Timer: mtime.NewTimeWheel(),
	}
	commonActor.UserMap = make(map[string]*pb.ActorUserInfo)

	commonActor.RegisterProtoHandler(int32(pb.Protocols_PS2S_TcpGateTopicReq2), commonActor.TcpGateTopicReq2) // 绑定长链接gate的topic

	return commonActor
}

func (s *CommonActor) TcpGateTopicReq2(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.S2S_TcpGateTopicReq2
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	s.UpdateGateTopic(&req)

	return &pb.S2S_TcpGateTopicRes2{}, nil, int32(pb.ErrorCode_Success)
}

func (s *CommonActor) Str() string {
	return fmt.Sprintf("Actor: %s", s.ID())
}

func (s *CommonActor) AddGateTopic(gateTopicId, uid string) {
	if s.UserMap == nil {
		s.UserMap = make(map[string]*pb.ActorUserInfo)
	}

	// if _, ok := s.UserMap[uid]; !ok {
	s.UserMap[uid] = &pb.ActorUserInfo{
		Uid:    uid,
		GateId: gateTopicId,
	}
	// }
	logger.Infof("Actor AddGateTopic, actorType:%v, actorId:%v, gateId:%v, uid:%v,userMap[%+v]", s.Type(), s.ID(), gateTopicId, uid, s.UserMap)
	// s.SaveActor2Redis()
}

func (s *CommonActor) SaveActor2Redis(actorType string) {
	_, err := s.Srv.RedisSet(context.Background(),
		db.KeyUserActor(actorType, s.ID()),
		s.ToJsonString(),
		time.Duration(baseconf.GetBaseConf().AccTokenTTL)*time.Second)
	if err != nil {
		return
	}
}

func (s *CommonActor) ReloadActorFromRedis(actorType string) {
	jsonData, err := s.Srv.RedisGet(context.Background(), db.KeyUserActor(actorType, s.ID()))
	logger.Infof("ReloadActorFromRedis jsonData:%v", jsonData)
	if err != nil {
		return
	}

	if jsonData == "" {
		return
	}

	err = json.Unmarshal([]byte(jsonData), &s.ActorInfo)
	if err != nil {
		return
	}
}

func (s *CommonActor) DelGateTopic(uid string) {
	if s.UserMap == nil {
		s.UserMap = make(map[string]*pb.ActorUserInfo)
	}

	logger.Infof("Actor DelGateTopic, actorType:%v, actorId:%v, uid:%v", s.Type(), s.ID(), uid)
	delete(s.UserMap, uid)

	// s.SaveActor2Redis()
}

func (s *CommonActor) CleanGateTopic() {
	s.UserMap = nil

	// s.SaveActor2Redis()
}

func (s *CommonActor) ToJsonString() string {
	commonActorJsonData, err := json.Marshal(&s.ActorInfo)
	if err != nil {
		return ""
	}

	return string(commonActorJsonData)
}

func (s *CommonActor) Invoke(ctx context.Context, in *base.ProtoMsg) (msg *base.ProtoMsg, err error) {

	now := time.Now()
	msg = &base.ProtoMsg{
		MsgId:   int32(pb.Protocols_PS2C_ErrorCodeNtf),
		UserId:  in.UserId,
		UAID:    s.ID(),
		Data:    []byte("user invoke error"),
		ErrCode: int32(pb.ErrorCode_InternalError),
	}

	msgId, uid, data := in.MsgId, in.UserId, in.Data
	defer func() (*base.ProtoMsg, error) {
		if err := recover(); err != nil {
			eStr := fmt.Sprintf("UserInvoke recover Msg:%+v %s %s err:%v", pb.Protocols(msgId), s.Str(), in.Str(), err)
			msg.Data = []byte(eStr)
			s.Error(errorx.Newf("UserInvoke recover error: %v", err))
		}
		delay := time.Since(now).Milliseconds()
		logStr := fmt.Sprintf("===>>>UserInvoke Msg:%v Delay:%d UAID:%s MSG-RET:%s", pb.Protocols(msg.MsgId), delay, s.ID(), msg.Str())
		s.Debugf(logStr)
		s.WarnDelayf(delay, logStr)
		return msg, nil
	}()

	logger.Debugf("===>>> Invoke protoMsg: %s", in.Str())
	if in.ReqIdx > 0 && s.ActorInfo.LastReqIdx == in.ReqIdx && s.ActorInfo.LastMsgId == msgId {
		// rpcErr := base.RpcError{Err: fmt.Errorf("RepeatMsg"), Code: int32(pb.ErrorCode_RepeatMsg)}
		// msg, err = &base.ProtoMsg{MsgId: int32(pb.Protocols_PS2C_ErrorCodeNtf), UserId: uid, Data: []byte(rpcErr.Error()), ErrCode: int32(pb.ErrorCode_RepeatMsg)}, nil
		s.Debugf("actor ReqRepeated msg:%+v actor.LastMsgId:%d actor.LastReqIdx:%d", in, s.LastMsgId, s.LastReqIdx)
		msg.Data = []byte("RepeatMsg")
		msg.ErrCode = int32(pb.ErrorCode_RepeatMsg)
		return msg, nil
	}

	// 线上关闭gm指令
	if msgId == int32(pb.Protocols_PC2LS_UseGameCommandReq) {
		if !conf.Base().IsDebug && in.AppId != "idip" { // idip部分指令可以使用
			msg.Data = []byte("illegal message id")
			msg.ErrCode = int32(pb.ErrorCode_IllegalOperationError)
			return msg, nil
		}
	}

	handler, ok := s.MsgFunc[msgId]
	if !ok {
		errStr := fmt.Sprintf("UserActor:UserInvoke invalid msgId:{%v,%v}, uid:%v", pb.Protocols(msgId), msgId, uid)
		s.Errorf(errStr)
		msg.Data = []byte(errStr)
		msg.ErrCode = int32(pb.ErrorCode_UnKnownMsg)
		return msg, nil
	}

	rsp, err, errCode := handler(ctx, in)

	if errCode == int32(pb.ErrorCode_Success) {
		// commit 写入redis
		utils.SafeRunNoError(func() {
			err = s.commit2Redis()
			if err != nil {
				s.Errorf(err.Error())
			}
		})
	}

	// fixme fail reset userdata 缓存数据已是脏了

	if /*err != nil ||*/ errCode != int32(pb.ErrorCode_Success) || rsp == nil {
		s.Warnf("UserInvoke handler error, msgId:{%v,%v}, uid: %v, data: %v, err:%+v, errCode:%v", pb.Protocols(msgId), msgId, uid, len(data), err, errCode)
		if err == nil {
			err = fmt.Errorf("error code: %d", err)
		}
		msg.Data = []byte(err.Error())
		msg.ErrCode = errCode
		return msg, nil
	}

	rspName := rsp.ProtoReflect().Descriptor().Name()
	msgId, ok = pb.Protocols_value[string("P"+rspName)]
	if !ok {
		errStr := fmt.Sprintf("UserActor:UserInvoke rsp invalid: rspName:%s, msgId:{%v,%v}, uid:%v, rsp:%+v",
			rspName, pb.Protocols(msgId), uid, msgId, rsp)
		s.Errorf(errStr)
		msg.Data = []byte(errStr)
		msg.ErrCode = int32(pb.ErrorCode_UnKnownMsg)
		return msg, nil
	}

	data, err = proto.Marshal(rsp)
	if err != nil {
		s.Errorf("UserActor:UserInvoke proto.Marshal error, msgId:{%v,%v}, uid:%v, err:%+v", pb.Protocols(msgId), msgId, uid, err)
		msg.Data = []byte(err.Error())
		msg.ErrCode = int32(pb.ErrorCode_SerializeError)
		return msg, nil
	}

	if in.ReqIdx > 0 {
		s.LastMsgId = in.MsgId
		s.LastReqIdx = in.ReqIdx
	}
	// msg, err = &base.ProtoMsg{MsgId: msgId, UserId: uid, Data: data, ErrCode: int32(pb.ErrorCode_Success)}, nil
	msg.MsgId = msgId
	msg.Data = data
	msg.ErrCode = int32(pb.ErrorCode_Success)
	if msg.MsgId != int32(pb.Protocols_PS2S_SvcStatusRes) {
		s.Infof("\n===>>>MSG-DOWN, msg:[%T], {%s}, {%s}, {%s}\n", rsp, utils.PrettyJsonLimit(rsp), in.Str(), msg.Str())
	}

	return msg, nil
}

func (s *CommonActor) SaveMongoDB(mongoDbName service.MongoDbType, key string, value proto.Message) error {
	kvTable, err := db.BuildKvTable(value, key)
	if err != nil {
		return err
	}

	return s.Srv.SaveDbByKvTable(mongoDbName, key, kvTable)
}

// GetCache 从redis缓存中获取, 没有就从db中获取并写入redis
func (s *CommonActor) GetCache(mongoDbName service.MongoDbType, key string, msg proto.Message) (*state.KvTable, error) {
	kvTable, err := s.Srv.GetCache(mongoDbName, key, server.ICache(s.Srv))
	if err != nil {
		return nil, err
	}

	err = db.ParseKvTable(kvTable, msg)
	if err != nil {
		return kvTable, err
	}

	return kvTable, nil
}

// 写到redis缓存
func (s *CommonActor) Cache2Redis(mongoDbType service.MongoDbType, uaid string, key string, value proto.Message) error {
	if key == "" {
		errors.Errorf("UserActor.CacheDelay got key is nil")
		return nil
	}

	kvTable, err := db.BuildKvTable(value, key)
	if err != nil {
		return err
	}
	cacheMap := make(map[string]*pb.CacheKeyDataEx, 0)
	cacheMap[key] = &pb.CacheKeyDataEx{
		Key: key,
		// DataLen:     int32(len(kvTable.Data)),
		MongoDBType: string(mongoDbType),
	}
	err = s.SaveCacheKeyEx(uaid, cacheMap, s.GetCacheTTL()) // gc后再保留600s
	if err != nil {
		return err
	}
	// 只是缓存, 不会同步到DB
	return s.Srv.SaveMongoAndRedis(mongoDbType, key, kvTable, nil, nil)
}

func (s *CommonActor) GetCacheTTL() int {
	// 设置延迟同步的key
	gcTime, err := strconv.Atoi(baseconf.GetBaseConf().UserActorGCTime)
	if err != nil {
		gcTime = 600 // 默认600s秒
	}

	return gcTime*2 + 600 // gc后再保留600s
}

// redis缓存数据同步到mongo
func (s *CommonActor) SyncCache2Mongo(uaid string) error {
	if err := s.doSyncCache2MongoEx(uaid); err != nil {
		logger.Errorf("syncCache2Mongo err:%v", err.Error())
		return err
	}
	return nil
}

// 设置延迟同步的key
func (s *CommonActor) SaveCacheKeyEx(uaid string, cacheMap map[string]*pb.CacheKeyDataEx, ttl int) error {
	var (
		err error
	)
	if len(cacheMap) == 0 {
		return nil
	}

	cacheKey := db.KeyCacheRedisData(uaid)

	vals := make([]interface{}, 0)
	var val []byte
	for _, cacheData := range cacheMap {
		val, err = json.Marshal(cacheData)
		if err != nil {
			logger.Errorf("--->>>CacheKey [%+v] Marshal error:%+v", cacheData, err)
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
	if conf.Base().IsDebug {
		logger.Debugf("--->>>write writeCacheKey=%v, 新增同步key:%v", cacheKey, vals)
	} else {
		logger.Debugf("--->>>write writeCacheKey=%v, uid:%v", cacheKey, s.ID())

	}

	return nil
}

func (s *CommonActor) doSyncCache2MongoEx(uaid string) error {
	var err error
	if uaid == "" {
		// 用户未登陆时, actor没有对应的ID
		return nil
	}

	// 读取缓存数据的key列表
	cacheKey := db.KeyCacheRedisData(uaid)
	vals, err := s.Srv.SPopN(context.Background(), cacheKey, 50)
	if err != nil {
		logger.Errorf("SPopN key[%s], val:[%v], error: %+v", cacheKey, vals, err)
	}
	if len(vals) == 0 {
		return nil
	}

	logger.Debugf("--->>>syncCache2Mongo 当前同步key列表:%v", cacheKey)

	dbTypeMap := make(map[service.MongoDbType]map[string]*state.KvTable)
	// kvTableMap := make(map[string]*state.KvTable, 0)
	// for dbKey, cacheVal := range cacheKeyData.Keys {
	for _, v := range vals {
		cacheKeyData := &pb.CacheKeyDataEx{}
		err = json.Unmarshal([]byte(v), cacheKeyData)
		if err != nil || cacheKeyData.Key == "" || cacheKeyData.MongoDBType == "" {
			logger.Errorf("--->>>syncCache2Mongo 指定的数据错误:%+v", cacheKeyData)
			continue
		}
		eachMongoDBType := service.MongoDbType(cacheKeyData.MongoDBType)
		if eachMongoDBType == service.MongoDbType_MongoNil {
			logger.Debugf("--->>>syncCache2Mongo 没有指定的数据库:%v", cacheKeyData)
			continue
		}

		if kvTable, ok := s.Srv.CacheRedisKeyExist(cacheKeyData.Key, nil); !ok {
			logger.Debugf("--->>>syncCache2Mongo 对应的缓存数据不存在:%v", cacheKeyData)
			continue
		} else {
			// 初始化未创建的key
			if _, ok = dbTypeMap[eachMongoDBType]; !ok {
				dbTypeMap[eachMongoDBType] = make(map[string]*state.KvTable)
			}
			// 缓存的数据
			dbTypeMap[eachMongoDBType][cacheKeyData.Key] = kvTable
			logger.Debugf("syncCache2Mongo cacheData read-success :%+v", cacheKeyData)
		}
	}

	for dbType, kvTableMap := range dbTypeMap {
		if dbType == service.MongoDbType_MongoNil {
			continue
		}

		// 事务形式写入db
		err = s.Srv.UpsertMongoTableTransaction(dbType, nil, kvTableMap)
		if err != nil {
			logger.Errorf("syncCache2Mongo UpsertMongoTableTransaction, mongoDbType=%v, got error:%v", dbType, err)

			// 重新将同步失败的缓存keys写回redis
			cacheMap := make(map[string]*pb.CacheKeyDataEx, 0)
			for dbKey, _ := range kvTableMap {
				cacheMap[dbKey] = &pb.CacheKeyDataEx{
					Key:         dbKey,
					MongoDBType: string(dbType),
				}
			}
			err = s.SaveCacheKeyEx(uaid, cacheMap, s.GetCacheTTL()) // gc后再保留600s
		}
	}
	// logger.Debugf("syncCache2Mongo transaction success :%v, %d", dbTypeMap, len(dbTypeMap))
	return nil
}

// commit2Redis
//
//	@Description: 单独一个请求修改的数据提交到redis
//	@receiver s
func (s *CommonActor) commit2Redis() error {
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
	err = s.Srv.UpsertRedisTableTransaction(service.RedisCache, meta, kvTableMap)
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

func (s *CommonActor) getCacheTTL() int {
	// 设置延迟同步的key
	gcTime, err := strconv.Atoi(baseconf.GetBaseConf().UserActorGCTime)
	if err != nil {
		gcTime = 600 // 默认600s秒
	}

	return gcTime*2 + 600 // gc后再保留600s
}

// 设置延迟同步的key
func (s *CommonActor) saveCacheKeyEx(cacheMap map[string]*pb.CacheKeyDataEx, ttl int) error {
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
	if conf.Base().IsDebug {
		logger.Debugf("--->>>write writeCacheKey=%v, 新增同步key:%v", cacheKey, vals)
	} else {
		logger.Debugf("--->>>write writeCacheKey=%v, id:%v", cacheKey, s.ID())
	}

	return nil
}

func (s *CommonActor) UpdateGateTopic(req *pb.S2S_TcpGateTopicReq2) {
	switch req.Opt {
	case pb.GateTopicOperator_GTO_bind: // 建立绑定
		s.AddGateTopic(req.GateId, req.Uid)

	case pb.GateTopicOperator_GTO_unbound: // 解除绑定
		s.DelGateTopic(req.Uid)

	default:
		s.Warnf("TcpGateTopicReq, 未支持的操作类型: %+v", req)
	}
	s.Debugf("收到useractor的广播, 更新topic, uaid:%s, req:%+v", s.ID(), req)
}

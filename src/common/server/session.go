package server

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/utils"
	dapr "github.com/dapr/go-sdk/client"
	"strconv"
	"time"
)

func (s *Server) GetUserSession(uid string) (*pb.UserSession, error, pb.ErrorCode) {
	if uid == "" {
		return nil, fmt.Errorf("accountId is empty"), pb.ErrorCode_ReLogin
	}

	kvTable, err := s.GetGlobalRedis(db.KeyUserSession(uid), nil)
	if err != nil {
		if errorx.Is(err, service.DB_ERROR_NOT_EXIST) { // session数据不存在,重新登录
			return nil, fmt.Errorf("db value not exist"), pb.ErrorCode_ReLogin
		} else {
			return nil, fmt.Errorf("internal error"), pb.ErrorCode_InternalError
		}
	}

	gUser := &pb.UserSession{}
	err = db.ParseKvTable(kvTable, gUser)
	if err != nil {
		return nil, err, pb.ErrorCode_ReLogin
	}

	return gUser, nil, pb.ErrorCode_Success
}

// func (s *Server) SaveUserSession2(session *pb.UserSession) error {
//	if session.Uid == "" {
//		return fmt.Newf("accountId is empty")
//	}
//
//	key := db.KeyUserSession(session.Uid)
//	kvTable, err := db.BuildKvTable(session, key)
//	if err != nil {
//		return err
//	}
//
//	data, err := json.Marshal(kvTable)
//	if err != nil {
//		logger.Newf("SaveUserSession Marshal err: ", kvTable, err)
//		return err
//	}
//
//	ttlMap := map[string]string{"ttlInSeconds": strconv.Itoa(conf.Base().AccTokenTTL)}
//
//	ctx, cancelFunc := context.WithTimeout(context.Background(), global.DB_INVOKE_TIMEOUT*time.Second)
//	defer cancelFunc()
//
//	item, err := s.Daprc.GetStateWithConsistency(ctx, string(service.RedisGlobal), key, nil, dapr.StateConsistencyStrong)
//	if err != nil {
//		logger.Newf("DBNext GetStateWithConsistency err: %v, %v, %+v", err, key, item)
//		return err
//	}
//
//	opt := &dapr.StateOperation{
//		Type: dapr.StateOperationTypeUpsert,
//		Item: &dapr.SetStateItem{
//			Key:      key,
//			Value:    data,
//			Metadata: ttlMap,
//			Etag: &dapr.ETag{
//				Value: item.Etag,
//			},
//			Options: &dapr.StateOptions{
//				Concurrency: dapr.StateConcurrencyFirstWrite,
//				Consistency: dapr.StateConsistencyStrong,
//			},
//		},
//	}
//
//	ctx2, cancelFunc2 := context.WithTimeout(context.Background(), global.DB_INVOKE_TIMEOUT*time.Second)
//	defer cancelFunc2()
//	err = s.Daprc.ExecuteStateTransaction(ctx2, string(service.RedisGlobal), nil, []*dapr.StateOperation{opt})
//	if err != nil {
//		logger.Newf("DBNext  SaveState err: %v, %v, %+v", err, key, item)
//		return err
//	}
//
//	//err = s.SaveGlobalRedis(key, kvTable, ttlMap)
//	//if err != nil {
//	//	return err
//	//}
//
//	return nil
// }

func (s *Server) SaveUserSession(session *pb.UserSession) error {
	_, err := utils.RetryDoSyncInterval(3, 30, func() (any, error) {
		return nil, s.doSaveUserSession(session)
	})

	return err
}

func (s *Server) doSaveUserSession(session *pb.UserSession) error {
	if session.Uid == "" {
		return fmt.Errorf("accountId is empty")
	}

	key := db.KeyUserSession(session.Uid)
	kvTable, err := db.BuildKvTable(session, key)
	if err != nil {
		return err
	}

	data, err := json.Marshal(kvTable)
	if err != nil {
		logger.Errorf("SaveUserSession Marshal err: ", kvTable, err)
		return err
	}

	ttlMap := map[string]string{"ttlInSeconds": strconv.Itoa(conf.Base().AccTokenTTL)}

	// retryPolicy := backoff.NewExponentialBackOff()

	// for {
	ctx, _ := context.WithTimeout(context.Background(), global.DB_INVOKE_TIMEOUT*time.Second)
	// defer cancelFunc()
	item, err := s.Daprc.GetStateWithConsistency(ctx, string(service.RedisGlobal), key, nil, dapr.StateConsistencyStrong)
	if err != nil {
		logger.Errorf("DBNext GetStateWithConsistency err: %v, %v, %+v", err, key, item)
		return err
		// waitTime := retryPolicy.NextBackOff()
		// time.Sleep(waitTime)
		// continue
	}

	// etag = item.Etag
	opt := &dapr.StateOperation{
		Type: dapr.StateOperationTypeUpsert,
		Item: &dapr.SetStateItem{
			Key:      key,
			Value:    data,
			Metadata: ttlMap,
			Etag: &dapr.ETag{
				Value: item.Etag,
			},
			Options: &dapr.StateOptions{
				Concurrency: dapr.StateConcurrencyFirstWrite,
				Consistency: dapr.StateConsistencyStrong,
			},
		},
	}
	// ExecuteStateTransaction方法调用
	err = s.Daprc.ExecuteStateTransaction(ctx, string(service.RedisGlobal), nil, []*dapr.StateOperation{opt})
	if err != nil {
		return err
	}
	// break
	// }

	return nil
}

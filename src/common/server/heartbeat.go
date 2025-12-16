package server

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gitee.com/aniwar2/musae/service"
	"google.golang.org/protobuf/proto"

	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/logger"
)

func (s *Server) SaveHeartBeat(uid string, gateTopic string) {
	key := db.KeyHeartBeat(uid)

	rsp := &pb.C2LS_HeartBeatRes{
		GateTopic: gateTopic,
	}

	kvTable, err := db.BuildKvTable(rsp, key)
	if err != nil {
		logger.Error(err)
		return
	}

	logger.Debugf("saveHeartBeat 更新心跳, %+v", rsp)
	ttlMap := map[string]string{"ttlInSeconds": strconv.Itoa(int(conf.GConf().Base.HeartbeatTimout))}
	err = s.SaveGlobalRedis(key, kvTable, ttlMap)
	if err != nil {
		logger.Error(err)
	}
}

// UpdateHeartBeatExpire 更新心跳超时时间
func (s *Server) UpdateHeartBeatExpire(uid string) {
	key := db.KeyHeartBeat(uid)
	_, err := s.RedisExpire(context.Background(), key, time.Duration(conf.GConf().Base.HeartbeatTimout)*time.Second)
	if err != nil {
		logger.Error(err)
	}
}

// GetHeartBeat 获取心跳数据
func (s *Server) GetHeartBeat(uid string) *pb.C2LS_HeartBeatRes {
	kvTable, err := s.GetRedis(service.RedisGlobal, db.KeyHeartBeat(uid), nil)
	if err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			return nil
		} else {
			logger.Errorf(err.Error())
			return nil
		}
	}

	if kvTable == nil {
		return nil
	}

	heartBeat := &pb.C2LS_HeartBeatRes{}
	err = proto.Unmarshal(kvTable.Data, heartBeat)
	if err != nil {
		logger.Errorf(err.Error())
		return nil
	}

	return heartBeat
}

// CleanHeartBeat 删除心跳数据
func (s *Server) CleanHeartBeat(uid string) error {
	_, err := s.RedisDel(context.Background(), db.KeyHeartBeat(uid))
	if err != nil {
		return err
	}
	return nil
}

// IsOnlineByUid 给定玩家是否在线
func (s *Server) IsOnlineByUid(uid string) bool {
	_, err := s.GetGlobalRedis(db.KeyHeartBeat(uid), nil)
	if err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			return false
		} else {
			logger.Errorf(err.Error())
			return false
		}
	}
	return true
}

func (s *Server) GetGateTopic(uid string) (string, bool) {
	heartBeat := s.GetHeartBeat(uid)
	if heartBeat == nil {
		return "", false
	}

	if heartBeat.GateTopic == "" {
		return "", false
	}

	return heartBeat.GateTopic, true
}

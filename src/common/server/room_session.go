package server

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
)

func (s *Server) SaveRoomBindingData(uid string, roomId string) error {
	if uid == "" {
		return fmt.Errorf("uid is empty")
	}

	key := db.KeyPlayerUidAndRoomId(uid)
	binding := &cmd.RoomBindingData{
		RoomId: roomId,
	}
	kvTable, err := db.BuildKvTable(binding, key)
	if err != nil {
		return err
	}

	ttlMap := map[string]string{"ttlInSeconds": strconv.Itoa(conf.GConf().Base.RoomTokenTTL)}
	return s.SaveGlobalRedis(key, kvTable, ttlMap)
}

// 处理玩家在大厅还是在房间
func (s *Server) GetRoomBindingData(uid string) (*cmd.RoomBindingData, error, cmd.ErrorCode) {
	if uid == "" {
		return nil, fmt.Errorf("roleId is empty"), cmd.ErrorCode_Room_player_not_exist
	}

	kvTable, err := s.GetGlobalRedis(db.KeyPlayerUidAndRoomId(uid), nil)
	if err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) { //session数据不存在,重新登录
			return nil, fmt.Errorf("db value not exist"), cmd.ErrorCode_Room_player_not_exist
		} else {
			return nil, fmt.Errorf("internal error"), cmd.ErrorCode_InternalError
		}
	}

	binding := &cmd.RoomBindingData{}
	err = db.ParseKvTable(kvTable, binding)
	if err != nil {
		return nil, err, cmd.ErrorCode_InternalError
	}

	return binding, nil, cmd.ErrorCode_Success
}

func (s *Server) CheckInRoom(uid string) bool {
	data, err, _ := s.GetRoomBindingData(uid)
	if err != nil {
		return false
	}
	return data.RoomId != ""
}

//func (s *Server) GetRoomSession(uid string) (*cmd.RoomSession, error, cmd.ErrorCode) {
//	if uid == "" {
//		return nil, fmt.Errorf("accountId is empty"), cmd.ErrorCode_Room_player_not_exist
//	}
//
//	kvTable, err := s.GetGlobalRedis(db.KeyRoomSession(uid), nil)
//	if err != nil {
//		if errors.Is(err, service.DB_ERROR_NOT_EXIST) { //session数据不存在,重新登录
//			return nil, fmt.Errorf("db value not exist"), cmd.ErrorCode_Room_player_not_exist
//		} else {
//			return nil, fmt.Errorf("internal error"), cmd.ErrorCode_InternalError
//		}
//	}
//
//	roomSession := &cmd.RoomSession{}
//	err = db.ParseKvTable(kvTable, roomSession)
//	if err != nil {
//		return nil, err, cmd.ErrorCode_ReLogin
//	}
//
//	return roomSession, nil, cmd.ErrorCode_Success
//}

//func (s *Server) SaveRoomSession(playerUid string, session *cmd.RoomSession) error {
//	if session.RoomId == "" {
//		return fmt.Errorf("accountId is empty")
//	}
//
//	key := db.KeyRoomSession(session.RoomId)
//	kvTable, err := db.BuildKvTable(session, key)
//	if err != nil {
//		return err
//	}
//
//	ttlMap := map[string]string{"ttlInSeconds": strconv.Itoa(conf.GConf().Base.RoomTokenTTL)}
//
//	err = s.SaveGlobalRedis(key, kvTable, ttlMap)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

//func (s *Server) GetToken(uid string) (string, error) {
//	if uid == "" {
//		return "", errors.New("uid is empty")
//	}
//
//	kv, err := s.GetCacheRedis(db.KeyUserToken(uid), nil)
//	if err != nil {
//		return "", errors.Wrap(err, "GetToken error")
//	}
//
//	return string(kv.Data), nil
//}
//
//func (s *Server) SaveToken(uid, token string) error {
//	if uid == "" {
//		return errors.New("uid is empty")
//	}
//	now := time.Now().Unix()
//	kvTable := &state.KvTable{
//		UID:     uid,
//		Data:    []byte(token),
//		UpSecTS: now,
//		InSecTS: now,
//		DataSrc: token,
//	}
//
//	ttlMap := map[string]string{"ttlInSeconds": strconv.Itoa(conf.GConf().Base.AccTokenTTL)}
//	err := s.SaveCacheRedis(db.KeyUserToken(uid), kvTable, ttlMap)
//	if err != nil {
//		return errors.Wrap(err, "SaveToken error")
//	}
//
//	return nil
//}

package server

import (
	"github.com/pkg/errors"
)

func (s *Server) GetToken(uid string) (string, error) {
	if uid == "" {
		return "", errors.New("uid is empty")
	}

	userSession, err, _ := s.GetUserSession(uid)
	if err != nil {
		return "", err
	}

	return userSession.Token, nil

	//kv, err := s.GetCacheRedis(db.KeyUserToken(uid), nil)
	//if err != nil {
	//	return "", errors.Wrap(err, "GetToken error")
	//}
	//
	//return string(kv.Data), nil
}

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

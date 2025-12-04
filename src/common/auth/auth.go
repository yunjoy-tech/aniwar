package auth

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/forgoer/openssl"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"
)

type Token struct {
	Uid         string `json:"uid"`
	Channel     string `json:"channel"`
	UUID        string `json:"uuid"`
	CreatedTime int64  `json:"created_time"`
	ExpiredTime int64  `json:"expired_time"`
}

//
// GenAuthToken
//  @Description: token gen func
//  @param prefix 前缀
//  @param key
//  @param uid
//  @param lifeTime
//  @return string
//

const AuthTokenSecret = "63c163@00e730387"

func EncodeAuthToken(uid, channel, uuid string, lifeTime int64) (string, error) {
	if len(uuid) <= 0 {
		uuid = utils.GenStrUUID()
	}
	if lifeTime <= 0 {
		lifeTime = global.TOKEN_LIFE_TIME
	}
	createdTime := time.Now().Unix()
	expiredTime := time.Now().Add(time.Duration(lifeTime) * time.Second).Unix()
	session := &Token{
		Uid:         uid,
		Channel:     channel,
		UUID:        uuid,
		CreatedTime: createdTime,
		ExpiredTime: expiredTime,
	}
	//logger.Debugf("session: %+v\n", session)
	data, err := json.Marshal(session)
	if err != nil || len(data) == 0 {
		return "", err
	}
	//logger.Debug("src:", len(data), data)
	data, err = openssl.AesECBEncrypt(data, []byte(AuthTokenSecret), openssl.PKCS7_PADDING)
	if err != nil || len(data) == 0 {
		return "", err
	}
	//logger.Debug("aes:", len(data), data)
	code := base64.StdEncoding.EncodeToString(data)
	//logger.Debug("code:", len(code), []byte(code))
	return code, nil
}

func DecodeAuthToken(token []byte) (*Token, error) {
	//logger.Debug("code:", len(token), token)
	code, err := base64.StdEncoding.DecodeString(string(token))
	if err != nil || len(code) == 0 {
		return nil, err
	}
	//logger.Debug("aes:", len(code), code)
	data, err := openssl.AesECBDecrypt(code, []byte(AuthTokenSecret), openssl.PKCS7_PADDING)
	if err != nil || len(data) == 0 {
		return nil, err
	}
	//logger.Debug("src:", len(data), data)
	session := &Token{}
	err = json.Unmarshal(data, session)
	if err != nil {
		return nil, err
	}

	//logger.Debugf("session: %+v\n", session)
	return session, nil

}

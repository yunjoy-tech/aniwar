package auth

import (
	"gitee.com/aniwar2/musae/gamelib/guid"
	"testing"
)

func Test_AuthToken(t *testing.T) {
	token, _ := EncodeAuthToken("test", "pc", guid.GenStrUuid(), 15)
	DecodeAuthToken([]byte(token))
}

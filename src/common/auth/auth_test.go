package auth

import (
	"github.com/yunjoy-tech/musae/gamelib/guid"
	"testing"
)

func Test_AuthToken(t *testing.T) {
	token, _ := EncodeAuthToken("test", "pc", guid.GenStrUuid(), 15)
	DecodeAuthToken([]byte(token))
}

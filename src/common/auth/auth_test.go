package auth

import (
	"testing"

	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"
)

func Test_AuthToken(t *testing.T) {
	token, _ := EncodeAuthToken("test", "pc", utils.GenStrUUID(), 15)
	DecodeAuthToken([]byte(token))
}

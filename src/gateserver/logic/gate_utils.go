package logic

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

func IsBroadcastCmd(messageId int32) bool {
	switch cmd.Protocols(messageId) {
	case cmd.Protocols_Protocols_None:
		return true
	default:
		return false
	}
}

func IsDeprecatedMsg(messageId int32) bool {
	_, ok := DeprecatedMsgId.Load(messageId)
	return ok
}

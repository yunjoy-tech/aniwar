package logic

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
)

func IsBroadcastCmd(messageId int32) bool {
	switch pb.Protocols(messageId) {
	case pb.Protocols_Protocols_None:
		return true
	default:
		return false
	}
}

func IsDeprecatedMsg(messageId int32) bool {
	_, ok := DeprecatedMsgId.Load(messageId)
	return ok
}

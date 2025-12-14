package logic

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
)

func (s *LoginServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	default:
	}
	return err
}

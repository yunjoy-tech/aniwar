package logic

import (
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
)

func (s *LoginServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	default:
	}
	return err
}

package logic

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
)

func (s *BillServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	default:
	}
	return err
}

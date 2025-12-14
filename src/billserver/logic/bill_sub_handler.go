package logic

import (
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/bychannel/aniwar/src/proto/pb"
)

func (s *BillServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	default:
	}
	return err
}

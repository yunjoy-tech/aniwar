package logic

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/process"
	"gitee.com/aniwar2/musae/framework/threading"
)

func (s *GateServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {

	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	case int32(pb.Protocols_PS2S_SvcRestartReq):
		threading.GoSafe(func() {
			process.Exit()
		})
	default:
		s.Send2Client(msg)
	}
	return err
}

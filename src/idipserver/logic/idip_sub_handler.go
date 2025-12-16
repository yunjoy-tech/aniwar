package logic

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/process"
	"gitee.com/aniwar2/musae/utils"
)

func (s *IDIPServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	case int32(pb.Protocols_PS2S_SvcRestartReq):
		utils.GoSafeRun(func() {
			process.Exit()
		}, nil)
	default:
	}
	return err
}

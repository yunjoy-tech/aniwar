package logic

import (
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/process"
	"github.com/yunjoy-tech/musae/utils"
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

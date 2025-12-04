package logic

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/process"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
)

func (s *GateServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {

	switch msg.MsgId {
	case int32(cmd.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	case int32(cmd.Protocols_PS2S_SvcRestartReq):
		threading.GoSafe(func() {
			process.Exit()
		})
	default:
		s.Send2Client(msg)
	}
	return err
}

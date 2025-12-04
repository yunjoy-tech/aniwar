package logic

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
)

func (s *LoginServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(cmd.Protocols_PS2S_HotReloadReq):
		err = s.HandlerHotEvent(msg)
	default:
	}
	return err
}

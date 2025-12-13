package roomactor

import (
	"gitee.com/bychannel/musae/framework/baseactor"
)

type USBaseHandler struct {
	*baseactor.BaseHandler
	actor *RoomActor
}

func NewUSBaseHandler(actor *RoomActor, handler string) *USBaseHandler {
	return &USBaseHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, handler),
	}
}

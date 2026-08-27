package roomactor

import (
	"github.com/yunjoy-tech/musae/baseactor"
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

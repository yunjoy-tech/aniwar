package allianceactor

import (
	"github.com/yunjoy-tech/musae/baseactor"
)

type USBaseHandler struct {
	*baseactor.BaseHandler
	actor *AllianceActor
}

func NewUSBaseHandler(actor *AllianceActor, handler string) *USBaseHandler {
	return &USBaseHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, handler),
	}
}

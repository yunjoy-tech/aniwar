package mailactor

import (
	"github.com/yunjoy-tech/musae/baseactor"
)

type UMBaseHandler struct {
	*baseactor.BaseHandler
	actor *MailActor
}

func NewUMBaseHandler(actor *MailActor, handler string) *UMBaseHandler {
	return &UMBaseHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, handler),
	}
}

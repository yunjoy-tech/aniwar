package mailactor

import (
	"gitee.com/aniwar2/musae/framework/baseactor"
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

package useractor

import "gitee.com/aniwar2/musae/baseactor"

type UABaseHandler struct {
	*baseactor.BaseHandler
	actor *UserActor
}

func NewUABaseHandler(actor *UserActor, handler string) *UABaseHandler {
	return &UABaseHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, handler),
	}
}

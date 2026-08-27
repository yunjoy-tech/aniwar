package useractor

import "github.com/yunjoy-tech/musae/baseactor"

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

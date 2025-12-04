package stub

import (
	"context"

	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
)

type MailStub struct {
	Invoke  func(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error)
	ActorId string
}

func NewMailStub(id string) *MailStub {
	stub := new(MailStub)
	stub.ActorId = id
	return stub
}

func (a *MailStub) Type() string {
	return global.MailActorType
}

func (a *MailStub) ID() string {
	return a.ActorId
}

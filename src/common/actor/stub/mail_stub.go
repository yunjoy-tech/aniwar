package stub

import (
	"context"

	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/global"
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

package stub

import (
	"context"
	"gitee.com/bychannel/musae/framework/base"
	"gitee.com/bychannel/musae/framework/global"
)

type UserStub struct {
	UserInvoke  func(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error)
	EventInvoke func(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error)
	ActorId     string
}

func NewUserStub(id string) *UserStub {
	stub := new(UserStub)
	stub.ActorId = id
	return stub
}

func (a *UserStub) Type() string {
	return global.UserActorType
}

func (a *UserStub) ID() string {
	return a.ActorId
}

package stub

import (
	"context"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/global"
)

type CenterStub struct {
	Invoke  func(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error)
	ActorId string
}

func NewCenterStub(id string) *CenterStub {
	stub := new(CenterStub)
	stub.ActorId = id
	return stub
}

func (a *CenterStub) Type() string {
	return global.CenterActorType
}

func (a *CenterStub) ID() string {
	return a.ActorId
}

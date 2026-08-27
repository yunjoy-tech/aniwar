package stub

import (
	"context"
	"github.com/yunjoy-tech/musae/base"
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
	return CenterActorType
}

func (a *CenterStub) ID() string {
	return a.ActorId
}

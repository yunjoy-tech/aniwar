package stub

import (
	"context"
	"gitee.com/aniwar2/musae/base"
)

type AllianceStub struct {
	Invoke  func(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error)
	ActorId string
}

func NewAllianceStub(id string) *AllianceStub {
	stub := new(AllianceStub)
	stub.ActorId = id
	return stub
}

func (a *AllianceStub) Type() string {
	return AllianceActorType
}

func (a *AllianceStub) ID() string {
	return a.ActorId
}

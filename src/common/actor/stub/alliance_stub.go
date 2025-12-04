package stub

import (
	"context"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
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
	return global.AllianceActorType
}

func (a *AllianceStub) ID() string {
	return a.ActorId
}

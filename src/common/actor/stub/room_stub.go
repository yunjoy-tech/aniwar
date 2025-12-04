package stub

import (
	"context"

	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
)

type RoomStub struct {
	Invoke  func(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error)
	ActorId string
}

func NewRoomStub(id string) *RoomStub {
	stub := new(RoomStub)
	stub.ActorId = id
	return stub
}

func (a *RoomStub) Type() string {
	return global.RoomActorType
}

func (a *RoomStub) ID() string {
	return a.ActorId
}

package useractor

import (
	"context"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"time"
)

type UserHandler struct {
	*UABaseHandler
}

func NewUserHandler(actor *UserActor) *UserHandler {
	h := &UserHandler{UABaseHandler: NewUABaseHandler(actor, "UserHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_HeartBeatReq), h.HeartbeatReq) // 心跳
	return h
}

func (h *UserHandler) Init() error {
	return nil
}

func (h *UserHandler) EnterGame() error {
	return nil
}

func (h *UserHandler) DailyRefresh() error {
	return nil
}

func (h *UserHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *UserHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

func (h *UserHandler) HeartbeatReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_HeartBeatReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 在线人数维护
	h.actor.Srv.OnlinePlayers.Store(h.actor.ID(), time.Now().Unix())

	// 更新心跳key过期时间
	h.actor.Srv.UpdateHeartBeatExpire(h.actor.uid)

	return &pb.C2LS_HeartBeatRes{}, nil, 0
}

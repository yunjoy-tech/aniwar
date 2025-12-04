package useractor

import (
	"context"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/musae/framework/base"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"google.golang.org/protobuf/proto"
)

// HeartBeatHandler tcp心跳包
type HeartBeatHandler struct {
	*UABaseHandler
}

func NewHeartBeatHandler(actor *UserActor) *HeartBeatHandler {
	h := &HeartBeatHandler{UABaseHandler: NewUABaseHandler(actor, "HeartBeatHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_HeartBeatReq), h.HeartbeatReq) // 心跳

	return h
}

func (h *HeartBeatHandler) Init() error {
	return nil
}

func (h *HeartBeatHandler) EnterGame() error {
	return nil
}

func (h *HeartBeatHandler) DailyRefresh() error {
	return nil
}

func (h *HeartBeatHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *HeartBeatHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

func (h *HeartBeatHandler) HeartbeatReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_HeartBeatReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 在线人数维护
	h.actor.Srv.OnlinePlayers.Store(h.actor.ID(), time.Now().Unix())

	// 更新心跳key过期时间
	h.actor.Srv.UpdateHeartBeatExpire(h.actor.uid)

	return &cmd.C2LS_HeartBeatRes{}, nil, 0
}

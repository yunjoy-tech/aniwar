package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type RedPointFunc = func(*clidto.Comdata, []int64) error

type RedPointHandler struct {
	*UABaseHandler
	HandlerMap map[int32]RedPointFunc
}

func NewRedPointHandler(actor *UserActor) *RedPointHandler {
	h := &RedPointHandler{UABaseHandler: NewUABaseHandler(actor, "RedPointHandler"),
		HandlerMap: make(map[int32]RedPointFunc),
	}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerRedPointReq), h.RedPointReq)
	return h
}

func (h *RedPointHandler) Init() error {
	return nil
}

func (h *RedPointHandler) EnterGame() error {
	return nil
}

func (h *RedPointHandler) DailyRefresh() error {
	return nil
}

func (h *RedPointHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *RedPointHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

// 延迟初始化红点协议映射
func (h *RedPointHandler) tryInitMap() {
	if len(h.HandlerMap) > 0 {
		return
	}
	h.HandlerMap[int32(cmd.RedPointModuleType_Card_Module)] = h.actor.CardHandler.handleRedPoint
	h.HandlerMap[int32(cmd.RedPointModuleType_Camp_Module)] = h.actor.CampHandler.handleRedPoint
	h.HandlerMap[int32(cmd.RedPointModuleType_Trial_module)] = h.actor.TrialHandler.handleRedPoint
	h.HandlerMap[int32(cmd.RedPointModuleType_Skin_module)] = h.actor.SkinHandler.handleRedPoint
	h.HandlerMap[int32(cmd.RedPointModuleType_Head_module)] = h.actor.LoginHandler.handleRedPoint
	h.Debugf("tryInitMap init success")
}

func (h *RedPointHandler) RedPointReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_PlayerRedPointReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	h.tryInitMap()
	handler := h.HandlerMap[req.Module]
	if handler == nil {
		return nil, fmt.Errorf("param error %d", req.Module), int32(cmd.ErrorCode_ParamError)
	}

	if len(req.IdList) == 0 {
		return &cmd.LS2C_PlayerRedPointRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
	}
	err = handler(h.actor.comData, req.IdList)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	return &cmd.LS2C_PlayerRedPointRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

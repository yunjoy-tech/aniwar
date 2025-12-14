package centeractor

import (
	"context"
	"fmt"
	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/baseactor"
	"gitee.com/aniwar2/musae/framework/baseconf"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/service"
	svc "gitee.com/aniwar2/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"strconv"
	"sync"
	"time"
)

type HotReloadHandler struct {
	*baseactor.BaseHandler
	actor *CenterActor
}

func (h *HotReloadHandler) Init() error {
	// implement me
	return nil
}

func (h *HotReloadHandler) SetDBData(dbData proto.Message) error {
	// implement me
	return nil
}

func (h *HotReloadHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	// implement me
	return service.MongoDbType_MongoGame, "", nil
}

func (h *HotReloadHandler) EnterGame() error {
	// implement me
	return nil
}

func (h *HotReloadHandler) DailyRefresh() error {
	// implement me
	return nil
}

func NewHotReloadHandler(actor *CenterActor) *HotReloadHandler {
	h := &HotReloadHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, "HotReloadHandler"),
	}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_HotReloadReq), h.HotReloadReq)             // 发起热更请求
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_HotReloadNotifyReq), h.HotReloadNotifyReq) // 热更通知结果
	return h
}

func (h *HotReloadHandler) HotReloadReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &pb.S2S_HotReloadReq{}
	now := time.Now().Unix()
	in.UnmarshalData(req)
	logger.Infof("HotReloadReq time:%v req:%+v ", now, req)
	rsp := &pb.S2S_HotReloadRes{Progress: map[string]string{}}
	if req.Type == 1 {
		// 通知到所有的actor和idip
		h.actor.Data.HotReloadMap = &sync.Map{}
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.ACTOR_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.IDIP_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.BATTLE_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.GUIDE_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.LOGIN_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.GATE_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.BILL_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.CENTER_SVC], now)

	} else if req.Type == 99 { // 系统邮件刷新
		h.actor.Data.HotReloadMap = &sync.Map{}
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.ACTOR_SVC], now)
	} else { // 返回
		h.actor.Data.HotReloadMap.Range(func(key, value any) bool {
			rsp.Progress[key.(string)] = value.(string)
			return true
		})

	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *HotReloadHandler) Send2PubTopic(req *pb.S2S_HotReloadReq, svcMap *sync.Map, now int64) {
	var err error
	svcMap.Range(func(key, value any) bool {
		svcData := value.(*pb.ServiceData)
		svcName := key.(string)
		if now < svcData.ReportTS+int64(baseconf.GetBaseConf().ServerHeartbeatTimout) {
			if len(req.Services) > 0 {
				if myUtils.ArrayContain(req.Services, svcName) {
					h.actor.Data.HotReloadMap.Store(svcName, "0")
					err = h.actor.Srv.PubTopicEvent(svc.EVENT_PRIVATE, svcName, h.actor.ID(), nil, req)
				}
			} else {
				h.actor.Data.HotReloadMap.Store(svcName, "0")
				err = h.actor.Srv.PubTopicEvent(svc.EVENT_PRIVATE, svcName, h.actor.ID(), nil, req)
			}
			if err != nil {
				logger.Errorf("HotReloadEvent error svc:%s", svcName)
			}
		}
		return true
	})
}

func (h *HotReloadHandler) HotReloadNotifyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &pb.S2S_HotReloadNotifyReq{}
	now := time.Now().Unix()
	err := in.UnmarshalData(req)
	fmt.Println("err:", err)
	logger.Infof("HotReloadNotifyReq time:%v req:%+v ", now, req)
	if req.Service != "" && req.Ts > 0 {
		h.actor.Data.HotReloadMap.Store(req.Service, strconv.Itoa(int(req.Ts)))
	}
	res := &pb.S2S_HotReloadNotifyRes{}
	return res, nil, int32(pb.ErrorCode_Success)
}

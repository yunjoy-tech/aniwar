package centeractor

import (
	"context"
	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/baseactor"
	"gitee.com/aniwar2/musae/baseconf"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	svc "gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/utils"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

type SvcRestartHandler struct {
	*baseactor.BaseHandler
	actor *CenterActor
}

func (h *SvcRestartHandler) Init() error {
	// implement me
	return nil
}

func (h *SvcRestartHandler) SetDBData(dbData proto.Message) error {
	// implement me
	return nil
}

func (h *SvcRestartHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	// implement me
	return service.MongoDbType_MongoGame, "", nil
}

func (h *SvcRestartHandler) EnterGame() error {
	// implement me
	return nil
}

func (h *SvcRestartHandler) DailyRefresh() error {
	// implement me
	return nil
}

func NewSvcRestartHandler(actor *CenterActor) *SvcRestartHandler {
	h := &SvcRestartHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, "SvcRestartHandler"),
	}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_SvcRestartReq), h.SvcRestartReq) // 发起热更请求
	return h
}

func (h *SvcRestartHandler) SvcRestartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &pb.S2S_SvcRestartReq{}
	now := time.Now().Unix()
	in.UnmarshalData(req)
	logger.Infof("HotReloadReq time:%v req:%s ", now, utils.PrettyJson(req))
	rsp := &pb.S2S_SvcRestartRes{Progress: map[string]int64{}}
	if req.Type == 1 {
		h.actor.Data.RestartEventTime = now
		h.actor.Data.SvcRestartMap = &sync.Map{}
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.GATE_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.ACTOR_SVC], now)
		h.Send2PubTopic(req, h.actor.Data.SvcMaps[global.IDIP_SVC], now)
	} else { // 返回
		h.actor.Data.SvcRestartMap.Range(func(key, value any) bool {
			rsp.Progress[key.(string)] = value.(int64)
			return true
		})
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *SvcRestartHandler) SvcRestart(svcName string, restartTime int64) {
	h.actor.Data.SvcRestartMap.Store(svcName, restartTime)
}

func (h *SvcRestartHandler) Send2PubTopic(req *pb.S2S_SvcRestartReq, svcMap *sync.Map, now int64) {
	var err error
	svcMap.Range(func(key, value any) bool {
		svcData := value.(*pb.ServiceData)
		svcName := key.(string)
		if now < svcData.ReportTS+int64(baseconf.GetBaseConf().ServerHeartbeatTimout) {
			if len(req.Services) > 0 {
				if myUtils.ArrayContain(req.Services, svcName) {
					h.actor.Data.SvcRestartMap.Store(svcName, "0")
					err = h.actor.Srv.PubTopicEvent(svc.EVENT_PRIVATE, svcName, h.actor.ID(), nil, req)
				}
			} else {
				h.actor.Data.SvcRestartMap.Store(svcName, "0")
				err = h.actor.Srv.PubTopicEvent(svc.EVENT_PRIVATE, svcName, h.actor.ID(), nil, req)
			}
			if err != nil {
				logger.Errorf("HotReloadEvent error svc:%s", svcName)
			}
		}
		return true
	})
}

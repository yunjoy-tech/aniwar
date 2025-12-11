package centeractor

import (
	"context"
	"encoding/json"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/http/request"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseactor"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/metrics"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"net/http"
	"strings"
	"time"
)

type SvcStatusHandler struct {
	*baseactor.BaseHandler
	actor *CenterActor
}

func (h *SvcStatusHandler) Init() error {
	// implement me
	return nil
}

func (h *SvcStatusHandler) SetDBData(dbData proto.Message) error {
	// implement me
	return nil
}

func (h *SvcStatusHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	// implement me
	return service.MongoDbType_MongoGame, "", nil
}

func (h *SvcStatusHandler) EnterGame() error {
	// implement me
	return nil
}

func (h *SvcStatusHandler) DailyRefresh() error {
	// implement me
	return nil
}

func NewSvcStatusHandler(actor *CenterActor) *SvcStatusHandler {
	h := &SvcStatusHandler{
		actor:       actor,
		BaseHandler: baseactor.NewBaseHandler(actor, "SvcStatusHandler"),
	}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_SvcStatusReq), h.SvcStatusReq)
	return h
}

func (h *SvcStatusHandler) SvcStatusReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &pb.S2S_SvcStatusReq{}
	now := time.Now().Unix()
	in.UnmarshalData(req)
	// logger.Debugf("SvcStatusReq: %s", utils.PrettyJson(req))
	for svcType := range h.actor.Data.SvcMaps {
		if req.Service != nil && strings.Contains(req.Service.Name, svcType) {
			if h.actor.Data.RestartEventTime > 0 {
				svc, ok := h.actor.Data.SvcMaps[svcType].Load(req.Service.Name)
				if ok && svc.(*pb.ServiceData).StartTime != req.Service.StartTime {
					h.actor.SvcRestartHandler.SvcRestart(req.Service.Name, req.Service.StartTime)
				}
				// 30分钟后关闭收集
				if now-h.actor.Data.RestartEventTime > 1800 {
					h.actor.Data.RestartEventTime = 0
				}
			}
			h.actor.Data.SvcMaps[svcType].Store(req.Service.Name, req.Service)

			logger.Debugf("SvcStatusReq, svcType:%s svc:%+v", string(svcType), req.Service)
			break
		}
	}
	if len(req.Actor) > 0 {
		for k, v := range req.Actor {
			if len(v.Counts) > 0 {
				v.LastTime = now
				h.actor.Data.ActorStatusMap.Store(k, v)
			}
		}
		h.UpdateActorStatus()
	}
	rsp := &pb.S2S_SvcStatusRes{Counts: []*pb.ActorCount{
		{
			Type:  global.PlayerCountType,
			Count: h.actor.Data.TotalPlayerCount,
		},
		{
			Type:  global.UserActorType,
			Count: h.actor.Data.UserActorCount,
		},
		{
			Type:  global.RoomActorType,
			Count: h.actor.Data.RoomActorCount,
		},
	}}

	for _, svcMap := range h.actor.Data.SvcMaps {
		svcMap.Range(func(key, value any) bool {
			svc := value.(*pb.ServiceData)
			if now < svc.ReportTS+int64(baseconf.GetBaseConf().ServerHeartbeatTimout) {
				rsp.Services = append(rsp.Services, svc)
			}
			return true
		})
	}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *SvcStatusHandler) UpdateActorStatus() {
	h.actor.Data.TotalPlayerCount = 0
	h.actor.Data.UserActorCount = 0
	h.actor.Data.RoomActorCount = 0
	now := time.Now().Unix()
	h.actor.Data.ActorStatusMap.Range(func(key, value any) bool {
		status := value.(*pb.ActorStatus)
		if status != nil {
			if now < status.LastTime+int64(baseconf.GetBaseConf().ServerHeartbeatTimout) {
				for _, actor := range status.Counts {
					switch actor.Type {
					case global.PlayerCountType:
						h.actor.Data.TotalPlayerCount += actor.Count
					case global.UserActorType:
						h.actor.Data.UserActorCount += actor.Count
					case global.RoomActorType:
						h.actor.Data.RoomActorCount += actor.Count
					}
				}
			}
		}
		return true
	})
	metrics.GaugeSet(metrics.AllUserCount, int64(h.actor.Data.UserActorCount))
	h.reportOnlineToTap(h.actor.Data.TotalPlayerCount)
}

func (h *SvcStatusHandler) reportOnlineToTap(total int32) {
	// {
	//    "client_id":"ClientID",
	//    "onlines":[{
	//      "server":"s1",
	//      "online":123,
	//      "timestamp":1489739590
	//    },{
	//      "server":"s2",
	//      "online":188,
	//      "timestamp":1489739560
	//    }]
	// }
	// 5分钟上报一次
	if h.actor.Data.UploadTapTs != 0 {
		nextTime := time.Unix(h.actor.Data.UploadTapTs, 0).Add(time.Minute * 5)
		if nextTime.After(time.Now()) {
			return
		}
		h.actor.Data.UploadTapTs += 60 * 5
	} else {
		h.actor.Data.UploadTapTs = time.Now().Unix()
	}
	// 构建数据
	data := map[string]interface{}{
		"client_id": taptap.TAPTAP_CLIENT_ID,
		"onlines": []struct {
			Server    string `json:"server"`
			Online    int32  `json:"online"`
			Timestamp int64  `json:"timestamp"`
		}{{Server: "s1", Online: h.actor.Data.TotalPlayerCount, Timestamp: time.Now().Unix()}},
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		logger.Error(err)
		return
	}
	req := request.New("https://se.tapdb.net/tapdb/online")
	resp, err := req.Method(http.MethodPost).JSONBytesBody(bytes).Send("")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	ret := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{}
	err = myUtils.DecodeReader(resp.Body, &ret)
	if err != nil {
		logger.Error(err)
		return
	}

	logger.Infof("成功上报在线人数到taptap status: %v, msg: %v", resp.StatusCode, ret)
}

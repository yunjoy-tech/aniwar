package centeractor

import (
	"github.com/dapr/go-sdk/actor"
	"github.com/yunjoy-tech/aniwar/src/actorserver/frame"
	"github.com/yunjoy-tech/aniwar/src/common/actor/stub"
	"github.com/yunjoy-tech/aniwar/src/common/server"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/service"
	"github.com/yunjoy-tech/musae/state"
	"github.com/yunjoy-tech/musae/utils"
	"google.golang.org/protobuf/proto"
	"sync"
)

type CenterData struct {
	Data           *pb.Room
	HotReloadMap   *sync.Map
	SvcRestartMap  *sync.Map
	ActorStatusMap *sync.Map
	SvcMaps        map[string]*sync.Map
	UploadTapTs    int64 // 上报tap时间戳

	TotalPlayerCount int32 // 在线player数量
	UserActorCount   int32 // 在线UserActor数量
	RoomActorCount   int32 // 在线RoomActor数量
	RestartEventTime int64 // 服务重启触发时间
}

type CenterActor struct {
	*frame.CommonActor
	Data CenterData

	SvcStatusHandler  *SvcStatusHandler
	HotReloadHandler  *HotReloadHandler
	SvcRestartHandler *SvcRestartHandler
}

func New() actor.Server {
	s := &CenterActor{
		CommonActor: frame.NewCommonActor(frame.GSrv),
	}
	s.ActorType = stub.CenterActorType
	s.SetActor(s)
	s.Srv = frame.GSrv
	// s.MsgFunc = make(map[int32]base.FProtoMsgHandler)
	s.SvcStatusHandler = NewSvcStatusHandler(s)
	s.HotReloadHandler = NewHotReloadHandler(s)
	s.SvcRestartHandler = NewSvcRestartHandler(s)
	s.Data.HotReloadMap = &sync.Map{}
	s.Data.ActorStatusMap = &sync.Map{}
	s.Data.SvcRestartMap = &sync.Map{}
	s.Data.SvcMaps = make(map[string]*sync.Map)
	s.Data.SvcMaps[server.GUIDE_SVC] = &sync.Map{}
	s.Data.SvcMaps[server.LOGIN_SVC] = &sync.Map{}
	s.Data.SvcMaps[server.GATE_SVC] = &sync.Map{}
	s.Data.SvcMaps[server.ACTOR_SVC] = &sync.Map{}
	s.Data.SvcMaps[server.BILL_SVC] = &sync.Map{}
	s.Data.SvcMaps[server.IDIP_SVC] = &sync.Map{}
	s.Data.SvcMaps[server.CENTER_SVC] = &sync.Map{}
	s.Data.UploadTapTs = 0

	return s
}

func (c *CenterActor) SetID(id string) {
	c.ServerImplBase.SetID(id)
}

func (c *CenterActor) Activate(invokeName string) error {
	// implement me
	// panic("implement me")
	c.ReloadActorFromRedis(stub.CenterActorType)

	return nil
}

func (c *CenterActor) Deactivate() error {

	utils.SafeRunNoError(func() {
		c.SaveActor2Redis(stub.CenterActorType)
	})

	return nil
}

func (c *CenterActor) GetCache(mongoDbName service.MongoDbType, key string, msg proto.Message) (*state.KvTable, error) {
	// implement me
	return nil, nil
}

func (c *CenterActor) Cache2Redis(mongoDbType service.MongoDbType, uaid string, key string, value proto.Message) error {
	// implement me
	return nil
}

func (c *CenterActor) SaveMongoDB(mongoDbName service.MongoDbType, key string, value proto.Message) error {
	// implement me
	return nil
}

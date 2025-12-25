package roomactor

import (
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/musae/utils"
	"strconv"
	"time"

	"gitee.com/aniwar2/aniwar/src/actorserver/frame"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/baseactor"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/service"
	svc "gitee.com/aniwar2/musae/service"
	"github.com/dapr/go-sdk/actor"
	_ "github.com/dapr/go-sdk/actor"
)

type RoomData struct {
	Data *pb.Room
	Tug  *pb.Tug
}

type RoomActor struct {
	*frame.CommonActor
	RoomData

	// Srv *frame.ActorServer

	RoomHandler *RoomHandler
	TugHandler  *TugHandler
}

func New() actor.Server {
	a := &RoomActor{
		CommonActor: frame.NewCommonActor(frame.GSrv),
		RoomData:    RoomData{},
	}
	a.ActorType = global.RoomActorType
	a.SetActor(a)

	a.Srv = frame.GSrv
	// a.RoomData = &pb.PvpRoom{}

	// a.MsgFunc = make(map[int32]base.FProtoMsgHandler)

	a.HandlersMap = make(map[svc.MongoDbType][]baseactor.IBaseHandler, 0)

	// 协议注册
	a.initHandlers()

	return a
}

func (s *RoomActor) SetID(id string) {
	s.ServerImplBase.SetID(id)
}

func (s *RoomActor) Activate(invokeName string) error {
	defer func() {
		if err := recover(); err != any(nil) {
			s.Error("RoomActor.SaveState recover, err: ", err)
		}
	}()

	s.ReloadActorFromRedis(global.RoomActorType)

	// 内存中没有数据
	if s.Data == nil {
		if err := s.loadAllData(); err != nil {
			return err
		}
		// redis中也没数据，初始化默认的值就行
		if s.Data == nil {
			s.Data = &pb.Room{
				RoomId:     "",
				RoomSecret: "",
				RoomState:  pb.RoomState_RoomState_idle, // 初始空闲状态
				PlayType:   0,
				OwnerUid:   "",
				Players:    nil,
				IsRecruit:  0,
			}
		}
	}

	s.Infof("=================>RoomActor Activate [%s]<=================", s.ID())

	return nil
}

func (s *RoomActor) Deactivate() error {
	utils.SafeRunNoError(func() {
		s.SaveActor2Redis(global.RoomActorType)
		// 判定是否超时deactivate
		now := time.Now()
		update := time.Unix(s.Data.UpdateTs, 0)
		gcTime, err := strconv.Atoi(conf.Base().UserActorGCTime)
		if err != nil {
			gcTime = 600 // 默认600s秒
		}
		if now.After(update.Add(time.Second * time.Duration(gcTime))) {
			s.RoomHandler.dismissRoomBySystem()
			// 清空数据
			s.Data = &pb.Room{
				RoomId:     "",
				RoomSecret: "",
				RoomState:  pb.RoomState_RoomState_idle, // 初始空闲状态
				PlayType:   0,
				OwnerUid:   "",
				Players:    nil,
				IsRecruit:  0,
			}
			mongoDbType, dbKey, dbMsg := s.RoomHandler.DBTable()
			err = s.Cache2Redis(mongoDbType, s.ID(), dbKey, dbMsg)
			if err != nil {
				s.Error(err)
			}
		}
	})

	s.Infof("=================>RoomActor Deactivate [%s]<=================", s.ID())
	return nil
}

func (s *RoomActor) initHandlers() {
	s.RoomHandler = NewRoomHandler(s)
	s.KeepHandler(s.RoomHandler)

	s.TugHandler = NewTugHandler(s)
	s.KeepHandler(s.TugHandler)

}

func (s *RoomActor) loadAllData() error {
	var (
		err    error
		startT = time.Now()
	)

	mongoDBs := []service.MongoDbType{
		service.MongoDbType_MongoAccount, // 账号db
		service.MongoDbType_MongoGame,    // 游戏db
	}

	for _, eachDB := range mongoDBs {
		if err = s.loadDBDataByDBType(eachDB); err != nil {
			return err
		}
	}

	s.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return nil
}

// 全量加载用户数据
func (s *RoomActor) loadDBDataByDBType(dbType service.MongoDbType) error {
	for _, handler := range s.HandlersMap[dbType] {
		dbTable, dbKey, dbVal := handler.DBTable()
		err := handler.LoadDBData(dbTable, dbKey, dbVal)
		if err != nil {
			return err
		}
	}
	return nil
}

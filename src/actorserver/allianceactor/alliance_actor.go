package allianceactor

import (
	"gitee.com/aniwar2/aniwar/src/actorserver/frame"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/baseactor"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/utils"
	"github.com/dapr/go-sdk/actor"
	"time"
)

type AllianceData struct {
	Data *pb.PServerAllianceInfo
}

type AllianceActor struct {
	*frame.CommonActor
	*AllianceData

	// Cache *CacheMgr

	AllianceHandler *AllianceHandler
}

func New() actor.Server {
	a := &AllianceActor{
		CommonActor:  frame.NewCommonActor(frame.GSrv),
		AllianceData: &AllianceData{},
	}
	// a.Cache = NewCacheMgr(a)
	a.ActorType = global.AllianceActorType
	a.SetActor(a)

	a.Srv = frame.GSrv
	a.HandlersMap = make(map[service.MongoDbType][]baseactor.IBaseHandler, 0)
	// 注册协议
	a.initHandlers()
	return a
}

func (a *AllianceActor) SetID(id string) {
	a.ServerImplBase.SetID(id)
}

func (a *AllianceActor) Activate(invokeName string) error {
	defer func() {
		if err := recover(); err != any(nil) {
			a.Error("AllianceActor.SaveState recover, err: ", err)
		}
	}()

	a.ReloadActorFromRedis(global.AllianceActorType)

	// 内存中没有数据
	if a.Data == nil {
		err := a.loadAllData()
		if err != nil {
			return err
		}
	}

	if err := a.EnterGame(); err != nil {
		a.Warnf("DoEnterGame got err: %+v", err)
	}

	a.Infof("=================>AllianceActor Activate [%s]<=================", a.ID())

	return nil
}

func (a *AllianceActor) Deactivate() error {
	a.Infof("=================>AllianceActor Deactivate [%s]<=================", a.ID())

	utils.SafeRunNoError(func() {
		a.SaveActor2Redis(global.AllianceActorType)
	})

	return nil
}

func (a *AllianceActor) initHandlers() {
	a.AllianceHandler = NewAllianceHandler(a)
	a.KeepHandler(a.AllianceHandler)
}

func (a *AllianceActor) loadAllData() error {
	var (
		err    error
		startT = time.Now()
	)

	mongoDBs := []service.MongoDbType{
		service.MongoDbType_MongoAccount, // 账号db
		service.MongoDbType_MongoGame,    // 游戏db
	}

	for _, eachDB := range mongoDBs {
		if err = a.loadDBDataByDBType(eachDB); err != nil {
			return err
		}
	}

	a.WarnDelayf(time.Since(startT).Milliseconds(), "")
	return nil
}

// 全量加载用户数据
func (a *AllianceActor) loadDBDataByDBType(dbType service.MongoDbType) error {
	for _, handler := range a.HandlersMap[dbType] {
		dbTable, dbKey, dbVal := handler.DBTable()
		err := handler.LoadDBData(dbTable, dbKey, dbVal)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *AllianceActor) EnterGame() error {
	var (
		err error
	)
	// 刷新所有模块
	for _, handlers := range u.HandlersMap {
		for _, handler := range handlers {
			err = handler.EnterGame()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

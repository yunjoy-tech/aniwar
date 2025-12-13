package useractor

import (
	"context"
	"fmt"
	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"
	"strconv"

	"gitee.com/bychannel/aniwar/src/actorserver/frame"
	"github.com/dapr/go-sdk/actor"

	"gitee.com/bychannel/aniwar/src/common/clidto"
	"github.com/pkg/errors"

	"gitee.com/bychannel/musae/framework/service"
	"gitee.com/bychannel/musae/framework/state"

	"gitee.com/bychannel/musae/framework/baseconf"
	"gitee.com/bychannel/musae/framework/global"

	"time"

	"gitee.com/bychannel/aniwar/src/actorserver/useractor/event"
	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/common/db"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/base"
	"gitee.com/bychannel/musae/framework/baseactor"
	svc "gitee.com/bychannel/musae/framework/service"
	"gitee.com/bychannel/musae/framework/threading"
	"google.golang.org/protobuf/proto"
)

type ActorState int

const (
	State_None     ActorState = 0 // UserActor 容器创建,无数据
	State_Active   ActorState = 1 // UserActor 容器激活,有数据
	State_Online   ActorState = 2 // UserActor 用户在线
	State_Offline  ActorState = 3 // UserActor 用户离线
	State_DeActive ActorState = 4 // UserActor 容器析构
)

type UserActor struct {
	// baseactor.BaseActor
	*frame.CommonActor
	UserData
	state ActorState
	// UserInfo string
	// id        string //player-UserActor id (等同UAID)
	uid      string // user id for svc invoke (等同AccountId)
	roleId   uint64
	liveTime int64 // 生存时间戳
	ctx      context.Context

	// Srv *frame.ActorServer

	eventManager   *event.Manager
	TaskTypeMgr    *TaskTypeMgr
	TaskTriggerMgr *TaskTriggerMgr
	comData        *clidto.Comdata

	// 延迟落库
	// handlersMap map[svc.MongoDbType][]baseactor.IBaseHandler

	AccountHandler      *AccountHandler
	UserHandler         *UserHandler
	OrderHandler        *OrderHandler
	BattleHandler       *BattleHandler
	LoginHandler        *LoginHandler
	BagHandler          *BagHandler
	GmHandler           *GmHandler
	CurrencyHandler     *CurrencyHandler
	ShopHandler         *ShopHandler
	MailHandler         *MailHandler
	DutyHandler         *DutyHandler
	GuideTaskHandler    *GuideTaskHandler
	FriendHandler       *FriendHandler
	RoleDetailHandler   *RoleDetailHandler
	UserRoomHandler     *UserRoomHandler
	FuncUnlockHandler   *FuncUnlockHandler
	UserChatHandler     *UserChatHandler
	OfflineEventHandler *OfflineEventHandler
	UserAllianceHandler *UserAllianceHandler
	ActivityHandler     *ActivityHandler
}

func New() actor.Server {
	// daprc, err := dapr.NewClientWithPort(server.GSrv.GRPCPort)
	// if err != nil || daprc == nil {
	//	u.Errorf("UserActor, dapr client error: %v", err)
	// }
	a := &UserActor{
		CommonActor: frame.NewCommonActor(frame.GSrv),
	}

	a.ActorType = global.UserActorType
	a.SetActor(a)
	a.HandlersMap = make(map[svc.MongoDbType][]baseactor.IBaseHandler, 0)
	// actor.Daprc = server.GSrv.Daprc
	// a.Daprc = daprc
	a.state = State_None
	// a.Srv = frame.GSrv
	a.ctx = context.Background()

	// a.MsgFunc = make(map[int32]base.FProtoMsgHandler)
	a.RpcMethods = make(map[string]*baseactor.RpcMethod)
	a.Data = &pb.PlayerData{}

	a.eventManager = event.NewManager(a.ID())
	a.TaskTypeMgr = NewTaskTypeMgr(a)
	a.TaskTriggerMgr = NewTaskTriggerMgr(a)
	a.liveTime = time.Now().Unix() // 创建server时间戳
	a.comData = clidto.BuildComData()

	// 协议注册
	a.initHandlers()
	// 异步函数注册
	a.initAsyncFunc()

	// delay save db timer
	// actor.Srv.AddTimer(true, time.Second*DELAY_SAVE_DB_TIME, actor.Delay2DB)

	threading.GoSafeWithParam(func(ua interface{}) {
		t := time.NewTicker(time.Second * common.FIXED_SAVE_DB_TIME)
		defer t.Stop()
		for {
			select {
			case <-ua.(*UserActor).ctx.Done():
				ua.(*UserActor).Info("UserActor closed")
				return
			case <-t.C:
				threading.RunSafe(ua.(*UserActor).FixedTime2DB)
			}
		}
	}, a)

	// 启动timer定时器
	threading.GoSafeWithParam(func(ua interface{}) {
		ua.(*UserActor).Timer.Run()
	}, a)

	a.Debugf("UserActorFactory create UserActor, %s", a.ID())
	return a
}

func (u *UserActor) Type() string {
	return global.UserActorType
}

func (u *UserActor) SetState(state ActorState) {
	u.state = state
}

func (u *UserActor) GetState() ActorState {
	return u.state
}

func (u *UserActor) Str() string {
	return fmt.Sprintf("UserActor:{ID:%v,UID:%v,PID:%v,State:%v}", u.ID(), u.uid, u.roleId, u.state)
}

func (u *UserActor) Activate(invokeName string) error {
	defer func() {
		if err := recover(); err != any(nil) {
			u.Trace("UserActor.SaveState recover, err: ", err)
		}
	}()

	if invokeName == "Delete" {
		// 删除actor操作, 不继续
		return errors.New("this is Delete, DO NOT Activate")
	}

	u.ReloadActorFromRedis(global.UserActorType)

	bMini := invokeName == "EventInvoke"
	startTime := time.Now()
	var err error
	var doLogin bool
	u.Debugf("UserActor Activate,InvokeName:%s, ID:%s", invokeName, u.ID())
	if u.Srv.State() != base.PState_Running {
		u.Warnf("Server is unavailable. cur state: %v", u.Srv.State())
		return fmt.Errorf("server is unavailable")
	}

	threading.RunSafe(func() {
		err = u.SyncCache2Mongo(u.ID())
		if err != nil {
			u.Errorf("Activate syncCache2Mongo err, %s, %+v", u.ID(), err)
		}
	})

	if u.GetStateManager() != nil {
		err = u.GetStateManager().Save()
	}

	if err != nil {
		u.Debug("SaveState err:", err)
		return err
	}
	u.SetState(State_Active)

	if u.Account == nil || u.Data.Base == nil {
		var bTrue bool
		bTrue, err = u.GetStateManager().Contains(db.KeyAccountInfo(u.uid))
		if err != nil && !errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			u.Errorf("u.GetStateManager().Contains,key:%s, err:%+v", db.KeyAccountInfo(u.uid), err)
			return errors.Wrapf(err, "account %s nil", u.uid)
		}
		// 1.判断角色是否已创建
		bTrue, err = u.GetStateManager().Contains(db.KeyUserBaseInfo(u.ID()))
		if err != nil && !errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			u.Debugf("u.GetStateManager().Contains, %+v", err)
			// return nil
		}
		if bTrue { // db中有数据
			// 角色已创建 加载数据
			if err = u.loadAllData(bMini); err != nil {
				u.Errorf("SaveState, load user data failed: %+v", err)
				return err // 将err向上返, 中断后续请求
			}
			doLogin = !bMini

		}
	}
	u.IsMiniMode = bMini
	if doLogin {
		u.SetState(State_Online)
		// EnterGame接口
		if err = u.EnterGame(); err != nil {
			u.Warnf("DoEnterGame got err: %+v", err)
		}
		// 尝试刷新跨天数据
		if err = u.DailyRefreshAll(); err != nil {
			u.Warnf("DailyRefreshAll got err: %+v", err)
		}
		// 执行离线事件
		u.OfflineEventHandler.ExecOfflineEvent()
	}

	// 埋点
	taptap.UserActorComm(u.ID(), "actorserver", 1, time.Now().Unix()-u.liveTime)

	u.Infof("=================>ActivateOn%s [%s]<=================", invokeName, u.ID())
	// metrics.GaugeInc(metrics.UserActorCount)
	u.WarnDelayf(time.Since(startTime).Milliseconds(), "UserActor Activate, %s", u.Str())
	return nil
}

func (u *UserActor) deactivate() error {
	// 用户下线，同步actor生命周期内修改的数据
	startTime := time.Now()
	if global.IsCloud { // 内网研发环境不开启，防止在不同私服先后登录，旧UserActor数据覆盖新数据的问题
		threading.RunSafe(func() {
			u.OfflineSync2DB()
		})
	}

	threading.RunSafe(func() {
		err := u.SyncCache2Mongo(u.ID())
		if err != nil {
			u.Errorf("Deactivate syncCache2Mongo err, %s, %+v", u.ID(), err)
		}
	})

	if u.GetUserData() != nil && u.GetUserData().Common.OfflineTime <= 0 {
		u.LoginHandler.UpdateOfflineTS(time.Now().Unix())
	}
	u.SaveActor2Redis(global.UserActorType)

	u.Timer.Stop()

	// 埋点
	taptap.UserActorComm(u.ID(), "actorserver", 0, time.Now().Unix()-u.liveTime)
	// metrics.GaugeDec(metrics.UserActorCount)
	u.SetState(State_DeActive)

	u.Infof("=================>Deactivate [%s]<=================", u.ID())
	u.WarnDelayf(time.Since(startTime).Milliseconds(), "UserActor Deactivate, %s", u.Str())

	return nil
}

func (u *UserActor) Deactivate() error {
	threading.RunSafe(func() {
		err := u.deactivate()
		if err != nil {
			u.Errorf("UserActor Deactivate got err: %v", err)
		}
	})

	return nil
}

func (u *UserActor) OfflineSync2DB() {
	var err error
	u.Debugf("用户下线，同步actor生命周期内修改的数据")
	for _, handlers := range u.HandlersMap {
		for _, handler := range handlers {
			if handler.IsDirty() {
				err = handler.SaveDB(true)
				if err != nil {
					u.Errorf("actor 退出时, 同步数据报错:%v", err.Error())
				}
			}
		}
	}
}

func (u *UserActor) IsHeartbeatTimeout() bool {
	if u.GetState() != State_Online {
		return false
	}
	if time.Now().Unix() > u.LastMsgTs+int64(baseconf.GetBaseConf().HeartbeatTimout) {
		return true
	}
	return false
}

func (u *UserActor) FixedTime2DB() {
	if u.state == State_None || u.state == State_DeActive {
		return
	}

	threading.RunSafe(func() {
		err := u.commit2Redis()
		if err != nil {
			u.Errorf("FixedTime2DB commit2Redis UserActor got err, %s, %+v", u.ID(), err)
		}
	})

	threading.RunSafe(func() {
		err := u.SyncCache2Mongo(u.ID())
		if err != nil {
			u.Errorf("FixedTime2DB syncCache2Mongo UserActor got err, %s, %+v", u.ID(), err)
		}
	})

	// 离线判断
	if u.GetState() == State_Online {
		threading.RunSafe(func() {
			if u.IsHeartbeatTimeout() {
				u.SetState(State_Offline)

				// 房间退出判定
				u.UserRoomHandler.tryExitRoom()

				// 离线时间更新
				u.LoginHandler.UpdateOfflineTS(time.Now().Unix())

				// 向allianceActor 发送topic 信息
				u.UserAllianceHandler.PushTopic2Alliance(pb.GateTopicOperator_GTO_unbound, "")

				// 下线埋点
				// threading.RunSafe(func() {
				//	lilith.WriteDataLog(&lilith.RoleLogout{
				//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_RoleLogout, u.uid, u.Account.CliDeviceInfo),
				//		RoleId:   u.ID(),
				//		Level:    int32(u.LoginHandler.getRoleLevel()),
				//		VipLevel: 0,
				//		Recharge: 0,
				//	})
				// })
				threading.RunSafe(func() {
					loginDate := time.Unix(u.Data.Base.Common.OnlineTime, 0)
					times := time.Now().Sub(loginDate).Seconds()
					e := &taptap.RoleLogout{
						PropertyFieldInfo: taptap.BuildPropertyFieldInfo(u.Account.CliDeviceInfo),
						RoleId:            u.ID(),
						Level:             int32(u.LoginHandler.getRoleLevel()),
						VipLevel:          0,
						Recharge:          0,
						GameTime:          strconv.Itoa(int(times)),
					}
					taptap.WriteDataLog(taptap.LogType_RoleLogout, u.uid, u.Account.TapUserInfo, e)
				})
			}
		})
	}
}

// commit2Redis
//
//	@Description: 单独一个请求修改的数据提交到redis
//	@receiver s
func (u *UserActor) commit2Redis() error {
	var err error
	// u.Debugf("commit2Redis UserActor, %s", u.ID())
	cacheMap := make(map[string]*pb.CacheKeyDataEx, 0)
	kvTableMap := make(map[string]*state.KvTable, 0)
	for mongoType, handlers := range u.HandlersMap {
		for _, handler := range handlers {
			if handler.IsRedisDirty() {
				_, dbKey, dbVal := handler.DBTable()

				kvTable, err := db.BuildKvTable(dbVal, dbKey)
				if err != nil {
					return err
				}

				if _, ok := kvTableMap[dbKey]; ok {
					return errors.New(fmt.Sprintf("重复的数据, dbKey=%s", dbKey))
				}

				kvTableMap[dbKey] = kvTable
				cacheMap[dbKey] = &pb.CacheKeyDataEx{
					Key: dbKey,
					// DataLen:     int32(len(kvTable.Data)),
					MongoDBType: string(mongoType),
				}
			}
		}

	}

	meta := map[string]string{"ttlInSeconds": strconv.Itoa(u.GetCacheTTL())} // 过期时间
	err = u.Srv.UpsertRedisTableTransaction(svc.RedisCache, meta, kvTableMap)
	if err != nil {
		return err
	}

	err = u.SaveCacheKeyEx(u.ID(), cacheMap, u.GetCacheTTL()) // gc后再保留600s
	if err != nil {
		return err
	}

	// 提交成功, 清除标记
	if err == nil {
		for _, handlers := range u.HandlersMap {
			for _, handler := range handlers {
				handler.CleanRedisDirty()
			}
		}
	}

	return nil
}

func (u *UserActor) SaveState() error {
	defer func() {
		if err := recover(); err != any(nil) {
			u.Trace("UserActor.SaveState recover, err: ", err)
		}
	}()

	// 处理 commit to redis

	return nil
}

func (u *UserActor) PushMsg2Gate(msg proto.Message) error {
	for _, topic := range u.UserMap {
		err := u.Srv.PubTopicEvent(svc.EVENT_PRIVATE, topic.GateId, u.ID(), []string{u.uid}, msg)
		if err != nil {
			return err
		}
	}
	return nil
	/*var err error
	switch conf.GConf().Base.Actor2GateType {
	case base.Actor2GateOnRpc:
		_, err = u.Srv.SvcInvoke(u.UserInfo, u.uid, u.roleId, u.ID(), msg)
	case base.Actor2GateOnCh:
		err = u.Srv.PubTopicEvent(svc.EVENT_PRIVATE, u.UserInfo, u.uid, u.roleId, u.ID(), msg)
	}
	return err*/
}

func (u *UserActor) SendErrCode(uid string, code pb.ErrorCode) {
	ntf := &pb.S2C_ErrorCodeNtf{
		ErrorCode: uint32(code),
		Param:     nil,
	}

	err := u.PushMsg2Gate(ntf)
	if err != nil {
		u.Error("SendErrCode failed, err: ", err)
	}
}

func (u *UserActor) UserInvoke(ctx context.Context, in *base.ProtoMsg) (msg *base.ProtoMsg, err error) {
	if u.IsMiniMode {
		if err = u.Activate("UserInvoke"); err != nil {
			msg = &base.ProtoMsg{
				MsgId:   int32(pb.Protocols_PS2C_ErrorCodeNtf),
				UserId:  in.UserId,
				UAID:    u.ID(),
				Data:    []byte(err.Error()),
				ErrCode: int32(pb.ErrorCode_LoadDBError),
			}
			return msg, err
		}
	}

	if in.MsgId == int32(pb.Protocols_PS2S_SvcInvokeReq) {
		msg.MsgId = int32(pb.Protocols_PS2S_SvcInvokeRes)
		msg.ErrCode = int32(pb.ErrorCode_Success)
		msg.Data = nil
		return msg, nil
	}

	msg, err = u.Invoke(ctx, in)
	// 清除actor的comdata缓存
	if u.comData.Flag {
		u.comData = clidto.BuildComData()
	}
	u.LastMsgTs = time.Now().Unix()
	return msg, err
}

func (u *UserActor) EventInvoke(ctx context.Context, in *base.ProtoMsg) (msg *base.ProtoMsg, err error) {
	return u.Invoke(ctx, in)
}

func (u *UserActor) initHandlers() {
	u.AccountHandler = NewAccountHandler(u)
	u.KeepHandler(u.AccountHandler)
	u.Type()

	u.UserHandler = NewUserHandler(u)
	u.KeepHandler(u.UserHandler)

	u.OrderHandler = NewOrderHandler(u)
	u.KeepHandler(u.OrderHandler)

	u.BattleHandler = NewBattleHandler(u)
	u.KeepHandler(u.BattleHandler)

	u.LoginHandler = NewLoginHandler(u)
	u.KeepHandler(u.LoginHandler)

	u.CurrencyHandler = NewCurrencyHandler(u)
	u.KeepHandler(u.CurrencyHandler)

	u.ShopHandler = NewShopHandler(u)
	u.KeepHandler(u.ShopHandler)

	u.MailHandler = NewMailHandler(u)
	u.KeepHandler(u.MailHandler)

	u.FriendHandler = NewFriendHandler(u)
	u.KeepHandler(u.FriendHandler)

	u.GmHandler = NewGmHandler(u)
	u.KeepHandler(u.GmHandler)

	u.BagHandler = NewBagHandler(u)
	u.KeepHandler(u.BagHandler)

	u.RoleDetailHandler = NewRoleDetailHandler(u)
	u.KeepHandler(u.RoleDetailHandler)

	u.UserRoomHandler = NewUserUserRoomHandler(u)
	u.KeepHandler(u.UserRoomHandler)

	u.FuncUnlockHandler = NewFuncUnlockHandler(u)
	u.KeepHandler(u.FuncUnlockHandler)

	u.UserChatHandler = NewUserChatHandler(u)
	u.KeepHandler(u.UserChatHandler)

	u.OfflineEventHandler = NewOfflineEventHandler(u)
	u.KeepHandler(u.OfflineEventHandler)

	u.UserAllianceHandler = NewUserUserAllianceHandler(u)
	u.KeepHandler(u.UserAllianceHandler)

	u.ActivityHandler = NewActivityHandler(u)
	u.KeepHandler(u.ActivityHandler)

	// ------------------任务数据初始化，必须放置在最后处理，否则导致任务初始化异常-------------------
	u.DutyHandler = NewDutyHandler(u)
	u.KeepHandler(u.DutyHandler)

	u.GuideTaskHandler = NewGuideTaskHandler(u)
	u.KeepHandler(u.GuideTaskHandler)
}

func (u *UserActor) initAsyncFunc() {
	// 异步事件监听
	u.eventManager.Listen(TASK_EVENT_ROLE_LEVEL_CHANGE, event.ListenerFunc(u.DutyHandler.tryInitData))
	u.eventManager.Listen(TASK_EVENT_QUEST_COMPLETE, event.ListenerFunc(u.DutyHandler.tryInitData))
	u.eventManager.Listen(TASK_EVENT_CARD_CREATE, event.ListenerFunc(u.LoginHandler.tryUnlockHeadsEvent))
	u.eventManager.Listen(TASK_EVENT_BREAKTHROUGH, event.ListenerFunc(u.LoginHandler.tryUnlockHeadsEvent))
	u.eventManager.Listen(TASK_EVENT_CARD_CREATE, event.ListenerFunc(u.RoleDetailHandler.tryHandleRoleLife))
	u.eventManager.Listen(TASK_EVENT_PLAYER_LOGIN, event.ListenerFunc(u.RoleDetailHandler.tryHandleRoleLife))
	u.eventManager.Listen(TASK_EVENT_ACHIEVE_COMPLETE, event.ListenerFunc(u.RoleDetailHandler.tryHandleRoleLife))
	u.eventManager.Listen(TASK_EVENT_ENTER_GAME, event.ListenerFunc(u.UserAllianceHandler.tryAddAllianceContribution))
	u.eventManager.Listen(TASK_EVENT_DUTY_ACTIVE_CHANGE, event.ListenerFunc(u.UserAllianceHandler.tryAddAllianceContribution))

	// 任务监听
	u.TaskTypeMgr.RegisterTaskTypeHandler(event.ListenerFunc(u.GuideTaskHandler.handleTaskType))
	u.TaskTypeMgr.RegisterTaskTypeHandler(event.ListenerFunc(u.DutyHandler.handleTaskType))
	u.TaskTypeMgr.RegisterTaskTypeHandler(event.ListenerFunc(u.ActivityHandler.handleTaskType))
	// 触发器监听
	u.TaskTriggerMgr.RegisterTaskTriggerHandler(event.ListenerFunc(u.ActivityHandler.handleTaskTrigger))
	u.TaskTriggerMgr.RegisterTaskTriggerHandler(event.ListenerFunc(u.GuideTaskHandler.handleTaskTrigger))
}

func (u *UserActor) DailyRefreshAll() error {
	// 判断是否刷新
	if ok, err := u.LoginHandler.TryUpdateLastLoginDate(); ok && err == nil {
		u.Info("执行每日刷新逻辑")
		// 刷新所有模块
		for _, handlers := range u.HandlersMap {
			for _, handler := range handlers {
				err = handler.DailyRefresh()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (u *UserActor) EnterGame() error {
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

func (u *UserActor) Hour0Handler(ctx context.Context, params []byte) error {
	u.Debugf("====>>> Hour0Handler")

	// 判断玩家是否在线跨0点, 跨天日志埋点
	if u.GetState() != State_Online {
		return nil
	}

	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.UserLogin{
	//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_UserLogin, u.uid, u.Account.CliDeviceInfo),
	//	})
	// })

	return nil
}

func (u *UserActor) Hour5Handler(ctx context.Context, params []byte) error {
	u.Debugf("====>>> Hour5Handler")
	if u.GetState() == State_None || u.GetState() == State_DeActive {
		return nil
	}

	// 每日刷新
	err := u.DailyRefreshAll()
	if err != nil {
		return err
	}

	return nil
}

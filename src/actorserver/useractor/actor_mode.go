package useractor

//
// import (
//	"context"
//	"time"
//
//	"gitee.com/aniwar2/musae/framework/threading"
//
//	"gitee.com/aniwar2/aniwar/src/common/datalog"
//	"gitee.com/aniwar2/musae/framework/global"
//
//	"github.com/dapr/go-sdk/actor"
//	"gitee.com/aniwar2/musae/framework/base"
//	"gitee.com/aniwar2/musae/framework/logger"
// )
//
// type UserActorMode struct {
//	actor.ServerImplBase
//	Player *UserActor
// }
//
// func NewUserActorMode() actor.Server {
//	var (
//		startT = time.Now()
//	)
//
//	logger.Debug("=================>NewUserActorMode<=================")
//	_actor := &UserActorMode{Player: NewUserActor()}
//
//	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")
//
//	return _actor
// }
//
// func (s *UserActorMode) Activate() error {
//	var (
//		err       error
//		startTime = time.Now()
//	)
//
//	err = s.Player.Activate()
//	s.Player.WarnDelayf(time.Since(startTime).Milliseconds(), "")
//
//	return err
// }
//
// func (s *UserActorMode) Deactivate() error {
//	var (
//		err    error
//		startT = time.Now()
//	)
//
//	err = s.Player.Deactivate()
//
//	s.Player.WarnDelayf(time.Since(startT).Milliseconds(), "")
//
//	return err
// }
//
// func (s *UserActorMode) Type() string {
//	return global.UserActorType
// }
//
// func (s *UserActorMode) SetID(id string) {
//	s.ServerImplBase.SetID(id)
//	s.Player.SetID(id)
// }
//
// func (s *UserActorMode) Reload() (err error) {
//	return nil
// }
//
// /*func (s *UserActorMode) GetPlayer() *UserActor {
//	if s.Player == nil {
//		s.Player = NewUserActor()
//	}
//	return s.Player
// }*/
//
// func (s *UserActorMode) SaveState() error {
//	logger.Debugf("UserActorMode SaveState UserActor, %s.", s.ID())
//
//	//s.Player.SetID(s.ID())
//	return s.Player.SaveState()
// }
//
// func (s *UserActorMode) Invoke(ctx context.Context, req string) (string, error) {
//	logger.Debug("get req = ", req)
//	return req, nil
// }
//
// func (s *UserActorMode) UserInvoke(ctx context.Context, req *base.ProtoMsg) (*base.ProtoMsg, error) {
//	var (
//		err    error
//		msg    *base.ProtoMsg
//		startT = time.Now()
//	)
//
//	msg, err = s.Player.UserInvoke(ctx, req)
//	if err != nil {
//		return nil, err
//	}
//
//	s.Player.WarnDelayf(time.Since(startT).Milliseconds(), "")
//
//	return msg, err
// }
//
// func (s *UserActorMode) Hour0Handler(ctx context.Context, params []byte) error {
//	logger.Debugf("====>>> Hour0Handler")
//
//	//判断玩家是否在线跨0点, 跨天日志埋点
//	if s.Player.GetState() != State_Online {
//		return nil
//	}
//
//	threading.RunSafe(func() {
//		datalog.Write(&datalog.UserLogin{
//			SystemFieldInfo: datalog.BuildHeadInfo(datalog.LogType_UserLogin, s.Player.uid, s.Player.Account.CliDeviceInfo),
//		})
//	})
//
//	return nil
// }
//
// func (s *UserActorMode) Hour5Handler(ctx context.Context, params []byte) error {
//	logger.Debugf("====>>> Hour5Handler")
//	if s.Player.GetState() == State_None || s.Player.GetState() == State_DeActive {
//		return nil
//	}
//
//	// 每日刷新
//	err := s.Player.DailyRefreshAll()
//	if err != nil {
//		return err
//	}
//
//	//err := s.Player.LoginHandler.dailyRefresh()
//	//if err != nil {
//	//	logger.Errorf(err.Error())
//	//}
//	//err = s.Player.DutyHandler.dailyRefresh()
//	//if err != nil {
//	//	logger.Error(err.Error())
//	//}
//	//err = s.Player.SignHandler.dailyRefresh()
//	//if err != nil {
//	//	logger.Error(err.Error())
//	//}
//
//	return nil
// }

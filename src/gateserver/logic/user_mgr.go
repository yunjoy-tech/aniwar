package logic

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"gitee.com/aniwar2/musae/utils"
	"sync"
	"time"
)

type UserMgr struct {
	s     *GateServer
	users sync.Map
}

func NewUserMgr(srv *GateServer) *UserMgr {
	mgr := &UserMgr{s: srv,
		users: sync.Map{}}
	return mgr
}

func (m *UserMgr) AddUser(uid string, roleId uint64, c *tcpx.Context, session *pb.UserSession) *User {
	user := NewUser(uid, roleId, c, m.s)
	err := user.SetSession(session)
	if err != nil {
		return nil
	}

	m.users.Store(uid, user)
	metrics.GaugeInc(metrics.UserConn)
	logger.Debugf("AddUser 记录user, %v, %v", uid, roleId)
	return user
}

func (m *UserMgr) GetUser(uid string) *User {
	value, ok := m.users.Load(uid)
	if ok && value != nil {
		return value.(*User)
	}
	return nil
}

func (m *UserMgr) DelUser(uid string) {

	m.users.Delete(uid)
	metrics.GaugeDec(metrics.UserConn)
	logger.Warnf("UserMgr 用户下线, %v", uid)
}

func (m *UserMgr) BroadcastMsg(msgId int32, appid string, data []byte) *User {
	logger.Infof("Send2Gate开始全服消息:%d,广播共可以通知人数:%d", msgId, m.UserNum())
	i := 0
	m.users.Range(func(key, value any) bool {
		logger.Debugf("全服广播空指针调试，user：%v", value.(*User))
		err := value.(*User).ReplyWithBody(msgId, 0, pb.ErrorCode_Success, data)
		if err != nil {
			logger.Warnf("[BroadcastMsg] ReplyWithBody error, %v %v %v %v", key, msgId, appid, err)
		} else {
			i++
		}
		return true
	})
	logger.Infof("Send2Gate结束全服消息:%d,广播成功通知人数:%d", msgId, i)
	return nil
}

func (m *UserMgr) Logout(uid, reason string) error {
	var user *User
	userData, ok := m.users.Load(uid)
	if !ok {
		return fmt.Errorf("logout user %s nil", uid)
	} else {
		user = userData.(*User)
	}

	/*
		//INTRO: 不删除, 准备后期做离线消息存储
		if user.ctx == nil {
			return nil
		}

		// 通知userActor广播actors解除绑定的topic
		err := m.s.NotifyActorGateTopic(user.uid, user.uaid, pb.GateTopicOperator_GTO_unbound)
		if err != nil {
			logger.Errorf(err.Error())
		}*/

	user.offlineTs = time.Now().Unix()
	if user.ctx.IsOnline() {
		user.ctx.CloseConn()
	}
	user.ctx = nil

	// m.DelUser(uid)

	logger.Infof("uid:%s reason:%s", uid, reason)
	return nil
}

func (m *UserMgr) UserNum() int32 {
	num := int32(0)
	m.users.Range(func(key, value any) bool {
		num++
		return true
	})
	return num
}

func (m *UserMgr) HeartbeatCheck() error {
	// now := time.Now().Unix()
	// m.users.Range(func(key, value any) bool {
	//	v := value.(*User)
	//	diff := int32(now - v.lastHeartbeatTs)
	//	if diff >= conf.Base().HeartbeatTimout {
	//		v.Logout(key.(string), "heartbeat timeout")
	//		m.DelUser(key.(string))
	//		if diff >= conf.Base().UserCacheTTL {
	//			//TODO 断线重连，复用数据，是否有必要？
	//		}
	//	}
	//	return true
	// })
	return nil
}

func (m *UserMgr) ReportDataMinute() error {
	err := reportOnlineUser(m)
	if err != nil {
		return err
	}
	return nil
}

/**************************************************function***********************************************************/

func reportOnlineUser(m *UserMgr) error {
	userNum := m.UserNum()

	// utils.SafeRunNoError(func() {
	//	lilith.WriteDataLog(&lilith.Online{
	//		LogType:     lilith.LogType_Online,
	//		Version:     strconv.Itoa(lilith.VERSION),
	//		EventTime:   time.Now().Format(lilith.FORMAT_DATETIME_LOG),
	//		GameId:      conf.GConf().Sdk.GameId,
	//		OnlineCount: int64(userNum),
	//		ServerTag:   m.s.AppId,
	//	})
	// })
	utils.SafeRunNoError(func() {
		e := &taptap.Online{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(nil),
			OnlineCount:       int64(userNum),
			ServerTag:         m.s.AppId,
		}
		taptap.WriteDataLog(taptap.LogType_Online, "", nil, e)
	})

	return nil
}

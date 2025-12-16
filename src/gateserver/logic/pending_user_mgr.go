package logic

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/tcpx"
	"google.golang.org/protobuf/proto"
)

/*type PendingUserMgr struct {
	s         *GateServer
	ticket    chan struct{}
	//pendingCh chan *PendingUser //排队中用户

}*/

type PendingUser struct {
	msg     proto.Message
	msgId   int32
	reqIdx  uint32
	uid     string
	data    []byte
	ctx     *tcpx.Context
	session *pb.UserSession
	startTs int64
	enter   bool
}

/*func NewPendingUserMgr(srv *GateServer) *PendingUserMgr {
	mgr := &PendingUserMgr{
		s: srv,
	}
	mgr.ticket = make(chan struct{}, int(conf.GConf().Base.GateLoginRateLimit))
	if conf.GConf().Base.GateLoginRateLimit <= 0 {
		conf.GConf().Base.GateLoginRateLimit = 1000
	}
	//mgr.pendingCh = make(chan *PendingUser, int(conf.GConf().Base.GateLoginRateLimit*global.SVC_INVOKE_TIMEOUT))
	return mgr
}*/

// GrantLoginToken 每秒钟下发的登录令牌，用于控制登录频率
// func (m *PendingUserMgr) GrantLoginToken() {
//	for i := 0; i < int(conf.GConf().Base.GateLoginRateLimit); i++ {
//		select {
//		case m.ticket <- struct{}{}:
//		default:
//			//logger.Debug("do nothing")
//		}
//	}
//	//logger.Debug("grant login token, size:", conf.GConf().Base.GateLoginRateLimit)
// }

/*func (m *PendingUserMgr) Push(req *PendingUser) pb.ErrorCode {
	select {
	case m.pendingCh <- req:
		return pb.ErrorCode_Success
	default:
		logger.Debug("PendingUserMgr drop PendingUser %s", req.uid)
	}
	return pb.ErrorCode_SystemBusy

}*/

/*func (m *PendingUserMgr) Execute() {
	for {
		threading.RunSafe(func() {
			user := <-m.pendingCh
			err := m.s.ExecuteLoginGame(user)
			if err != nil {
				logger.Debugf("OnNetMessage Auth error, %v %v %v", user.ctx.ClientIP(), user.ctx.Network(), err)
				user.ctx.CloseConn()
			}
			metrics.GaugeInc(metrics.GateAuthCount)
			metrics.GaugeDec(metrics.PendingUserCount)
		})
	}
}*/

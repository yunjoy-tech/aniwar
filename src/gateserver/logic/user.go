package logic

import (
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"gitlab.musadisca-games.com/wangxw/musae/framework/errorx"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/actor/stub"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/metrics"
	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
)

type Msg struct {
	msgId  int32
	reqIdx uint32
	Data   []byte
}

// 消息包缓存
type LastRspData struct {
	ReqIdx  uint32        // 客户端索引
	Up      cmd.Protocols // 上行协议号
	Down    cmd.Protocols // 下行协议号
	RspData []byte        // 接口返回数据
}

type User struct {
	s               *GateServer
	ctx             *tcpx.Context
	actor           *stub.UserStub
	ch              chan *Msg
	uid             string
	limitTs         int64
	limitNum        uint32
	limitSize       uint32
	lastHeartbeatTs int64
	//lastRspData     *LastRspData
	actorId   string
	roleId    uint64
	uaid      string // UserActor id
	onlineTs  int64
	offlineTs int64
	session   *cmd.UserSession
}

func NewUser(uid string, roleId uint64, c *tcpx.Context, s *GateServer) *User {
	u := &User{ctx: c, s: s, uid: uid, roleId: roleId}
	u.uaid = s.UAID(u.uid, u.roleId)
	u.ch = make(chan *Msg, 10)
	u.actor = stub.NewUserStub(u.uaid)
	u.s.ImpActorStub(u.actor)
	u.lastHeartbeatTs = time.Now().Unix()
	threading.GoSafe(u.GoHandleClientMsg)
	return u
}

func NewUserExt(session *cmd.UserSession, s *GateServer) *User {
	u := &User{ctx: nil, s: s, session: session}
	u.uaid = session.Uaid
	u.uid = session.Uid
	u.roleId = session.PlayerId
	u.actor = stub.NewUserStub(u.uaid)
	u.s.ImpActorStub(u.actor)
	u.lastHeartbeatTs = time.Now().Unix()
	return u
}

func (u *User) String() string {
	return fmt.Sprintf("{uid:%s,roleId:%v,uaid:%s}", u.uid, u.roleId, u.uaid)
}

func (u *User) CreateActorStub() {
	u.actor = stub.NewUserStub(u.uaid)
	u.s.ImpActorStub(u.actor)
}

func (u *User) PushClientMsg(msg *Msg) {
	select {
	case u.ch <- msg:
	default:
		atomic.AddInt32(&DS.LoseMsg, 1)
		metrics.GaugeInc(metrics.DropUpMsgCount)
		logger.Debugf("user msg chan full, drop msg: [%s] [%d]", u.uid, msg.msgId)
	}
}

func (u *User) GoHandleClientMsg() {
	for {
		msg := <-u.ch
		u.HandleClientMsg(msg.msgId, msg.Data, msg.reqIdx)
	}
}

func (u *User) HandleClientMsg(msgId int32, data []byte, reqIdx uint32) ([]byte, cmd.Protocols, cmd.ErrorCode) {
	// 废弃消息过滤
	var err error
	var b []byte
	var messageId cmd.Protocols
	var errCode cmd.ErrorCode

	if IsDeprecatedMsg(msgId) {
		b, messageId = u.HandleDeprecatedMsg(msgId)
		return b, messageId, cmd.ErrorCode_DeprecatedMsgError
	}

	// 合法的上行消息
	if common.IsUp(cmd.Protocols(msgId)) {
		if common.IsUserActorCmd(cmd.Protocols(msgId)) {
			b, messageId, errCode = u.HandleUserActor(msgId, data, reqIdx)
		} else if common.IsRoomActorCmd(cmd.Protocols(msgId)) {
			b, messageId, errCode = u.HandleRoomActor(msgId, data, reqIdx)
		} else {
			logger.Error("OnNetMessage unknown service cmd:", u.String(), msgId, len(data), data)
		}
	} else {
		logger.Error("OnNetMessage unknown cmd:", u.String(), msgId, len(data), data)
	}
	// 使用proto反序列化可能该字段为nil
	if b == nil {
		b = make([]byte, 0)
	}
	if b != nil && messageId != cmd.Protocols_Protocols_None {
		if u.ctx != nil {
			if messageId == cmd.Protocols_PS2C_ErrorCodeNtf {
				rsp := &cmd.S2C_ErrorCodeNtf{ErrorCode: uint32(errCode), Param: []string{strconv.Itoa(int(errCode))}}
				b, err = proto.Marshal(rsp)
				if err != nil {
					logger.Warn("OnNetMessage, HandleDeprecatedMsg Marshal, reply error:", u.String(), messageId, messageId, errorx.Wrap(err).Error())
				}
				err = u.ReplyWithBody(int32(messageId), reqIdx, errCode, b) // errCode不走加密
				if err != nil {
					logger.Warn("OnNetMessage, HandleDeprecatedMsg, reply error:", u.String(), messageId, messageId, errorx.Wrap(err).Error())
				}
			} else {
				err = u.ReplyWithBody(int32(messageId), reqIdx, cmd.ErrorCode_Success, b)
				if err != nil {
					logger.Warn("OnNetMessage, HandleDeprecatedMsg, reply error:", u.String(), messageId, messageId, errorx.Wrap(err).Error())
				}
			}

		} else {
			return b, messageId, errCode
		}
	}
	return nil, cmd.Protocols_Protocols_None, cmd.ErrorCode_InternalError
}

func (u *User) ReplyWithBody(downMsgId int32, reqIdx uint32, errCode cmd.ErrorCode, body []byte) error {
	metrics.GaugeInc(metrics.GateMsgCount)
	metrics.GaugeAdd(metrics.GateDownMsgSize, int64(len(body)))

	err := u.s.setLastRespData(u.uid, downMsgId, reqIdx, body)
	if err != nil {
		return err
	}

	if u.ctx == nil {
		logger.Debug("ReplyWithBody ctx is nil:", u.ctx)
		return errors.New("ctx is nil")
	}
	err = u.ctx.ReplyWithBody(downMsgId, int32(errCode), body)
	if err != nil {

	}
	return err
}

//func (u *User) Reply(messageID int32, src interface{}) error {
//	return u.ctx.Reply(messageID, src)
//}

func (u *User) DDosCheck(dataLen uint32) bool {
	now := time.Now().Unix()
	// ddos check
	if now >= u.limitTs+int64(conf.GConf().DDos.TimeInterval) {
		u.limitTs = now
		u.limitNum = 0
		u.limitSize = 0
	}

	u.limitNum++
	u.limitSize += dataLen
	if (conf.GConf().DDos.LimitPktNum > 0 && u.limitNum > conf.GConf().DDos.LimitPktNum) ||
		(conf.GConf().DDos.LimitByteNum > 0 && u.limitSize > conf.GConf().DDos.LimitByteNum) {
		logger.Info("[DDosCheck] user ", u.uid, " net limit and kickoff,time:", u.limitTs, ", pkt num:", u.limitNum, ", byte num:", u.limitSize)
		u.limitTs = now
		u.limitNum = 0
		u.limitSize = 0
		return true
	}
	return false
}

func (u *User) SetSession(session *cmd.UserSession) error {
	err := u.s.SaveUserSession(session)
	if err != nil {
		return err
	}
	u.session = session

	return err
}

//func (u *User) SetLastRsp(lastRsp *LastRspData) {
//	u.lastRspData = lastRsp
//}
//
//func (u *User) GetLastRsp() *LastRspData {
//	return u.lastRspData
//}

//func (u *User) SaveLastRsp(downNum int32, reqIdx uint32, rspBody []byte) {
//	if common.IsDown(cmd.Protocols(downNum)) {
//		if cmd.Protocols(downNum) == cmd.Protocols_PC2LS_HeartBeatRes {
//			// 忽略心跳
//			return
//		}
//
//		err, upNum := common.DownNumber2UpNumber(downNum)
//		if err != nil {
//			logger.Errorf("SaveLastRsp, DownNumber2UpNumber got err:%v", err)
//			return
//		}
//
//		//u.session.LastRspData = &cmd.LastRspData{
//		//	ReqIdx:  reqIdx,
//		//	UpCmd:   upNum,
//		//	DownCmd: downNum,
//		//	RspData: rspBody,
//		//}
//
//		u.s.SaveUserSession(u.session)
//	}
//}

func (u *User) IsOnline() bool {
	if u.offlineTs == 0 && u.ctx != nil {
		return true
	}
	return false
}

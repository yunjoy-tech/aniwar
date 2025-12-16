package logic

import (
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"gitee.com/aniwar2/musae/errorx"

	"gitee.com/aniwar2/aniwar/src/common"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/tcpx"
	"gitee.com/aniwar2/musae/threading"
)

type Msg struct {
	msgId  int32
	reqIdx uint32
	Data   []byte
}

// 消息包缓存
type LastRspData struct {
	ReqIdx  uint32       // 客户端索引
	Up      pb.Protocols // 上行协议号
	Down    pb.Protocols // 下行协议号
	RspData []byte       // 接口返回数据
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
	// lastRspData     *LastRspData
	actorId   string
	roleId    uint64
	uaid      string // UserActor id
	onlineTs  int64
	offlineTs int64
	session   *pb.UserSession
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

func NewUserExt(session *pb.UserSession, s *GateServer) *User {
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

func (u *User) HandleClientMsg(msgId int32, data []byte, reqIdx uint32) ([]byte, pb.Protocols, pb.ErrorCode) {
	// 废弃消息过滤
	var err error
	var b []byte
	var messageId pb.Protocols
	var errCode pb.ErrorCode

	if IsDeprecatedMsg(msgId) {
		b, messageId = u.HandleDeprecatedMsg(msgId)
		return b, messageId, pb.ErrorCode_DeprecatedMsgError
	}

	// 合法的上行消息
	if common.IsUp(pb.Protocols(msgId)) {
		if common.IsUserActorCmd(pb.Protocols(msgId)) {
			b, messageId, errCode = u.HandleUserActor(msgId, data, reqIdx)
		} else if common.IsRoomActorCmd(pb.Protocols(msgId)) {
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
	if b != nil && messageId != pb.Protocols_Protocols_None {
		if u.ctx != nil {
			if messageId == pb.Protocols_PS2C_ErrorCodeNtf {
				rsp := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(errCode), Param: []string{strconv.Itoa(int(errCode))}}
				b, err = proto.Marshal(rsp)
				if err != nil {
					logger.Warn("OnNetMessage, HandleDeprecatedMsg Marshal, reply error:", u.String(), messageId, messageId, errorx.Wrap(err, "").Error())
				}
				err = u.ReplyWithBody(int32(messageId), reqIdx, errCode, b) // errCode不走加密
				if err != nil {
					logger.Warn("OnNetMessage, HandleDeprecatedMsg, reply error:", u.String(), messageId, messageId, errorx.Wrap(err, "").Error())
				}
			} else {
				err = u.ReplyWithBody(int32(messageId), reqIdx, pb.ErrorCode_Success, b)
				if err != nil {
					logger.Warn("OnNetMessage, HandleDeprecatedMsg, reply error:", u.String(), messageId, messageId, errorx.Wrap(err, "").Error())
				}
			}

		} else {
			return b, messageId, errCode
		}
	}
	return nil, pb.Protocols_Protocols_None, pb.ErrorCode_InternalError
}

func (u *User) ReplyWithBody(downMsgId int32, reqIdx uint32, errCode pb.ErrorCode, body []byte) error {
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

// func (u *User) Reply(messageID int32, src interface{}) error {
//	return u.ctx.Reply(messageID, src)
// }

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

func (u *User) SetSession(session *pb.UserSession) error {
	err := u.s.SaveUserSession(session)
	if err != nil {
		return err
	}
	u.session = session

	return err
}

// func (u *User) SetLastRsp(lastRsp *LastRspData) {
//	u.lastRspData = lastRsp
// }
//
// func (u *User) GetLastRsp() *LastRspData {
//	return u.lastRspData
// }

// func (u *User) SaveLastRsp(downNum int32, reqIdx uint32, rspBody []byte) {
//	if common.IsDown(pb.Protocols(downNum)) {
//		if pb.Protocols(downNum) == pb.Protocols_PC2LS_HeartBeatRes {
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
//		//u.session.LastRspData = &pb.LastRspData{
//		//	ReqIdx:  reqIdx,
//		//	UpCmd:   upNum,
//		//	DownCmd: downNum,
//		//	RspData: rspBody,
//		//}
//
//		u.s.SaveUserSession(u.session)
//	}
// }

func (u *User) IsOnline() bool {
	if u.offlineTs == 0 && u.ctx != nil {
		return true
	}
	return false
}

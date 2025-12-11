package logic

import (
	"fmt"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/base"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"
)

func (s *GateServer) HandleLoginGame(c *tcpx.Context, session *pb.UserSession, messageID int32, data []byte, reqIdx uint32) ([]byte, *base.RpcError) {
	var req pb.C2G_LoginGameReq
	err := base.UnmarshalData(data, &req)
	if err != nil {
		return nil, &base.RpcError{Err: err, Code: int32(pb.ErrorCode_DeSerializeError)}
	}

	// if conf.GConf().Base.VersionCheck {
	//	err = s.VersionCheck(req.GetVersion())
	//	if err != nil {
	//		return nil, &base.RpcError{Err: err, Code: int32(pb.ErrorCode_VersionLimit)}
	//	}
	// }

	accountId := req.AccountId
	logger.Infof("GateServer:HandleLoginGame begin, %v", &req)
	// TODO 账号校验
	if session == nil {
		session, err, _ = s.GetUserSession(accountId)
		if session == nil {
			return nil, &base.RpcError{Err: fmt.Errorf("account[%s] session not found, req=%+v, err=%v ", accountId, req, err),
				Code: int32(pb.ErrorCode_ReLogin)}
		} else if err != nil {
			return nil, &base.RpcError{Err: err,
				Code: int32(pb.ErrorCode_ReLogin)}
		}
	}

	if session.Token != req.Token {
		return nil, &base.RpcError{Err: fmt.Errorf("account[%s] session token error, req=%+v, session=%+v", accountId, req, session),
			Code: int32(pb.ErrorCode_ReLogin)}

	}
	pendingUser := &PendingUser{
		msg:     &req,
		reqIdx:  reqIdx,
		uid:     accountId,
		data:    data,
		startTs: time.Now().Unix(),
		enter:   true,
		session: session,
	}

	if c != nil {
		c.SetAccountId(accountId)
		pendingUser.ctx = c
		return s.TcpLoginGame(pendingUser)
	} else {
		return s.HttpLoginGame(pendingUser)
	}

	return nil, &base.RpcError{Err: fmt.Errorf("ErrorCode,%v", pb.ErrorCode_InternalError),
		Code: int32(pb.ErrorCode_InternalError)}
}

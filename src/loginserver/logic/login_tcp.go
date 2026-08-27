package logic

import (
	"strconv"

	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/errorx"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/tcpx"
)

func (s *LoginServer) OnTcp(c *tcpx.Context) {

	messageID, e := tcpx.MessageIDOf(c.Stream)
	if e != nil {
		logger.Warn(errorx.Wrap(e, "").Error())
		return
	}
	logger.Debug("OnTcp: ", c.ClientIP(), c.Network(), pb.Protocols(messageID), len(c.Stream), c.Stream)

	data, err := tcpx.BodyBytesOf(c.Stream)
	if err != nil {
		logger.Warn("OnTcp BodyBytesOf", errorx.Wrap(err, "").Error())
		return
	}

	dataLen := len(data)
	// 包体大小限制
	if dataLen > conf.Base().GateMsgMaxSize {
		logger.Warn("OnTcp BodyBytesOf", errorx.Wrap(err, "").Error())
		return
	}

	// TODO test code wangxw
	switch pb.Protocols(messageID) {
	case pb.Protocols_PC2LS_LoginReq:
		block := make([]byte, dataLen)
		copy(block, data)
		req := &Msg{
			msgId:    messageID,
			Data:     block,
			ctx:      c,
			ClientIp: c.ClientIP(),
		}
		// 配置为0时，不做限制
		if conf.Login().LoginReqRate > 0 && conf.Login().LoginReqQueue > 0 {
			s.pushMsg(req)
		} else {
			res := s.handleLoginReq(req)
			if res.ErrCode == int32(pb.ErrorCode_Success) {
				err = req.ctx.Reply(int32(pb.Protocols_PLS2C_LoginRes), res.ErrCode, res)
			} else {
				err = req.ctx.Reply(int32(pb.Protocols_PS2C_ErrorCodeNtf), res.ErrCode, &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(res.ErrCode), Param: []string{strconv.Itoa(int(res.ErrCode))}})
			}
			if err != nil {
				logger.Error(err.Error())
			}
		}
	case pb.Protocols_PC2LS_HeartBeatReq:
		return
	default:
		logger.Warnf("invalid protocol,close conn,msgId:%d", messageID)
		c.CloseConn()
	}
}

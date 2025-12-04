package logic

import (
	"strconv"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"
	"gitlab.musadisca-games.com/wangxw/musae/framework/errorx"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"
)

func (s *LoginServer) OnTcp(c *tcpx.Context) {

	messageID, e := tcpx.MessageIDOf(c.Stream)
	if e != nil {
		logger.Warn(errorx.Wrap(e).Error())
		return
	}
	logger.Debug("OnTcp: ", c.ClientIP(), c.Network(), cmd.Protocols(messageID), len(c.Stream), c.Stream)

	data, err := tcpx.BodyBytesOf(c.Stream)
	if err != nil {
		logger.Warn("OnTcp BodyBytesOf", errorx.Wrap(err).Error())
		return
	}

	dataLen := len(data)
	// 包体大小限制
	if dataLen > baseconf.GetBaseConf().GateMsgMaxSize {
		logger.Warn("OnTcp BodyBytesOf", errorx.Wrap(err).Error())
		return
	}

	//TODO test code wangxw
	switch cmd.Protocols(messageID) {
	case cmd.Protocols_PC2LS_LoginReq:
		block := make([]byte, dataLen)
		copy(block, data)
		req := &Msg{
			msgId:    messageID,
			Data:     block,
			ctx:      c,
			ClientIp: c.ClientIP(),
		}
		// 配置为0时，不做限制
		if conf.GConf().BaseConf().LoginReqRate > 0 && conf.GConf().BaseConf().LoginReqQueue > 0 {
			s.pushMsg(req)
		} else {
			res := s.handleLoginReq(req)
			if res.ErrCode == int32(cmd.ErrorCode_Success) {
				err = req.ctx.Reply(int32(cmd.Protocols_PLS2C_LoginRes), res.ErrCode, res)
			} else {
				err = req.ctx.Reply(int32(cmd.Protocols_PS2C_ErrorCodeNtf), res.ErrCode, &cmd.S2C_ErrorCodeNtf{ErrorCode: uint32(res.ErrCode), Param: []string{strconv.Itoa(int(res.ErrCode))}})
			}
			if err != nil {
				logger.Error(err.Error())
			}
		}
	case cmd.Protocols_PC2LS_HeartBeatReq:
		return
	default:
		logger.Warnf("invalid protocol,close conn,msgId:%d", messageID)
		c.CloseConn()
	}
}

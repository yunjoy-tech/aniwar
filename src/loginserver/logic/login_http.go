package logic

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"strconv"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/utils"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/baseconf"
	"gitee.com/aniwar2/musae/framework/errorx"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/metrics"
	"gitee.com/aniwar2/musae/framework/tcpx"
	"github.com/dapr/go-sdk/service/common"
)

func (s *LoginServer) OnHttp(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("OnLogin failed, err: ", err)
		}
	}()

	if in == nil {
		return nil, fmt.Errorf("nil parameter")
	}
	logger.Infof("OnLogin [LoginStep] ContentType:%s, Verb:%s, QueryString:%s, len:%s", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	curTime := time.Now()
	// 白名单校验 TODO
	ip, err := utils.GetIP(in.Request)
	if err != nil {
		logger.Warn("OnLogin get ip failed. ", err.Error())
	}

	if len(in.Data) > baseconf.GetBaseConf().GateMsgMaxSize {
		return nil, fmt.Errorf("invalid error")
	}
	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	clientVersion := in.Request.Header.Get("client-version")
	platform := in.Request.Header.Get("platform")
	// 客户端版本验证
	if conf.GConf().Base.VersionCheck {
		logger.Infof("VersionCheck clientVersion:%s", clientVersion)
		err = s.VersionCheckExt(platform, clientVersion)
		if err != nil {
			logger.Warnf("VersionCheck error:%s", err.Error())
			out.Data = s.ErrorPack(pb.ErrorCode_VersionLimit)
			return out, nil
		}
	}

	srcData, err := tcpx.Decrypt(in.Data, "")
	if err != nil {
		logger.Warn("OnLogin Unpack Decrypt", errorx.Wrap(err, "").Error())
		return nil, err
	}

	messageId, err := tcpx.MessageIDOf(srcData)
	if err != nil {
		logger.Error("OnLogin MsgHandler parse messageId failed:", messageId, err)
		return nil, err
	}

	data, err := tcpx.BodyBytesOf(srcData)
	if err != nil {
		logger.Warn("OnLogin BodyBytesOf", errorx.Wrap(err, "").Error())
		return nil, err
	}

	res := s.handleLoginReq(&Msg{msgId: messageId, Data: data, ClientIp: ip})
	if res.ErrCode == int32(pb.ErrorCode_Success) {
		data, err = s.Pack(pb.Protocols_PLS2C_LoginRes, pb.ErrorCode_Success, res, "")
		delayTime := time.Since(curTime).Milliseconds()
		metrics.HistogramPut(metrics.LoginDelayHist, delayTime, metrics.Delay)
		logger.Debugf("===>>>OnLoginDelay, uid:%s, delay:%d, len:%v", res.AccountId, delayTime, len(data))
		logger.WarnDelayf(delayTime, "")
		taptap.LoginDelayComm(res.AccountId, nil, nil, messageId, delayTime)

	} else {
		data, err = s.Pack(pb.Protocols_PS2C_ErrorCodeNtf, pb.ErrorCode(res.ErrCode), &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(res.ErrCode), Param: []string{strconv.Itoa(int(res.ErrCode))}}, "")
	}
	if err != nil {
		logger.Warnf("OnLogin res pack err: %s", errorx.Wrap(err, "").Error())
	}
	out = &common.Content{
		Data:        data,
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}
	return out, nil
}

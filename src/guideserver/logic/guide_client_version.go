package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"time"

	"gitee.com/aniwar2/musae/framework/global"

	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/metrics"
	"github.com/dapr/go-sdk/service/common"
)

func (s *GuideServer) Version(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("/api/version failed, err: ", err)
		}
	}()

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}
	curTime := time.Now()
	var account string
	var platform int32
	out.Data, account, platform, err = s.getVersionInfo(in)
	if err != nil {
		metrics.GaugeInc(metrics.GuideFailedCount)
		logger.Errorf("getVersion error Account:%s Platform:%d err:%+v", account, platform, err)
		return out, nil
	}

	delay := time.Since(curTime).Milliseconds()
	logger.Infof("[guide] [LoginStep] ClientVersion Account:%s Platform:%d Delay:%d Data:%s", account, platform, delay, string(out.Data))
	metrics.GaugeInc(metrics.GuideSucceedCount)
	metrics.HistogramPut(metrics.GuideDelayHist, delay, metrics.Delay)
	return out, nil
}

func (s *GuideServer) getVersionInfo(in *common.InvocationEvent) ([]byte, string, int32, error) {
	if in == nil || len(in.Data) > conf.Base().GateMsgMaxSize {
		return nil, "", 0, fmt.Errorf("invocation parameter error")
	}
	// logger.Debugf("[guide] [LoginStep] /api/version - ContentType:%s, Verb:%s, QueryString:%s, data:%v", in.ContentType, in.Verb, in.QueryString, string(in.Data))

	// account check
	// account := in.Request.Header.Get("account")

	req := &server.VerReq{}
	if err := json.Unmarshal(in.Data, req); err != nil {
		logger.Warnf("[guide] /api/version, json Unmarshal fail,data: %s", string(in.Data))
		return nil, req.AccountId, req.Platform, err
	}
	version := &server.Version{}
	// 获取当前版本号
	platform := s.GetPlatform(req.Platform)
	updateVersion, err := s.GetUpdateVersionByChannel(platform)
	if err != nil {
		logger.Warnf("[guide] /api/version, version err: %v,data: %s", err, string(in.Data))
		updateVersion = conf.Base().VersionAndroid
	}
	version.UpdateVersion = updateVersion

	// 获取当前版本号对应的jenkins流水号
	updateVersionJenkins, err := s.GetJenkinsVersionByOnlineVersion(version.UpdateVersion)
	if err != nil {
		logger.Warnf("[guide] /api/version, minVersion err: %v,data: %s", err, string(in.Data))
		return nil, req.AccountId, req.Platform, err
	}
	version.DownloadPath = updateVersionJenkins

	// channel := strings.ToLower(req.Platform)
	switch platform {
	case "android":
		version.UpdateAddr = conf.GConf().SrvAddr.UpdateAddrARD
	case "ios":
		version.UpdateAddr = conf.GConf().SrvAddr.UpdateAddrIOS
	default:
		version.UpdateAddr = conf.GConf().SrvAddr.UpdateAddrARD
		logger.Warn("platform error: %s", "android" /*req.Platform*/)
	}

	/*var srvAddr string
	if req.IsHttp {
		if global.IsCloud {
			version.ServerAddr = conf.GConf().SrvAddr.HTTPAddr
			version.TcpAddr = conf.GConf().SrvAddr.TCPAddr
		} else {
			//srvAddr = global.Gateway
			version.ServerAddr = append(version.ServerAddr, global.Gateway)
			version.TcpAddr = append(version.TcpAddr, global.TcpAddr)
		}
		//version.ServerAddr = append(version.ServerAddr, srvAddr)
	} else {
		if global.IsCloud {
			version.ServerAddr = conf.GConf().SrvAddr.TCPAddr
		} else {
			if len(global.Gateway) > 0 {
				srvAddr = global.Gateway
			} else {
				localIP, err := utils.ExternalIP()
				if err != nil {
					logger.Error("ExternalIP failed")
					return nil, err
				}
				srvAddr = fmt.Sprintf("%s", localIP)
			}
			version.ServerAddr = append(version.ServerAddr, srvAddr)
			version.TcpAddr = append(version.TcpAddr, srvAddr)
		}
	}*/

	if global.IsCloud {
		version.ServerAddr = conf.GConf().SrvAddr.HTTPAddr
		version.TcpAddr = conf.GConf().SrvAddr.TCPAddr
	} else {
		version.ServerAddr = append(version.ServerAddr, global.Gateway)
		version.TcpAddr = append(version.TcpAddr, global.TcpAddr)
	}

	taptap.GetVersionComm(req.AccountId, nil, nil, taptap.ConvertStruct2Str(version), platform)

	data, err := json.Marshal(version)
	if err != nil {
		logger.Warn("[guide] ClientVersion Marshal error", err)
	}
	return data, req.AccountId, req.Platform, err
}

func (s *GuideServer) test(ctx context.Context, in *common.BindingEvent) (out []byte, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Trace("test failed, err: ", err)
		}
	}()

	var (
		srvAddr       []byte
		strAndroidVer string
		strIOSVer     string
	)

	if in == nil {
		err = fmt.Errorf("nil BindingEvent parameter")
	}
	logger.Debugf("[guide] test - in:%v", in)

	srvAddr, err = json.Marshal(conf.GConf().SrvAddr)
	if err != nil {
		logger.Errorf("[guide] /api/version, version err: %v,data: %s", err, string(in.Data))
		return nil, err
	}
	strAndroidVer, err = s.GetConfigKeyForStr(db.KeyCfgCVersionAndroid)
	if err != nil {
		logger.Warnf("[guide] /api/version, version err: %v,data: %s", err, string(in.Data))
		strAndroidVer = conf.Base().VersionAndroid
	}
	strIOSVer, err = s.GetConfigKeyForStr(db.KeyCfgCVersionIOS)
	if err != nil {
		logger.Warnf("[guide] /api/version, version err: %v,data: %s", err, string(in.Data))
		strIOSVer = conf.Base().VersionIOS
	}
	out = []byte(fmt.Sprintf("i'm ok, %s\nsrvAddr:%s\nandroid:%s\nios:%s\n",
		time.Now().Local().String(), srvAddr, strAndroidVer, strIOSVer))

	logger.Debugf("[guide] test, out: %s", string(out))
	return out, nil
}

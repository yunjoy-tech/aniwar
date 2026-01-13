package server

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/utils"
	daprCommon "github.com/dapr/go-sdk/service/common"
	"time"
)

// 启动流程3: Server
// Server Start逻辑
func (s *Server) Start() error {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Fatal("[server] Start recover, err: ", err)
		}
	}()

	// 启动流程4: 初始化service的http,网络等组件
	if err := s.InitBase(); err != nil {
		return err
	}
	logger.Info("[server] base init success")

	if err := s.OnPreInit(); err != nil {
		return err
	}

	if conf.Base().IsDebug {
		s.RegisterBindingInvocationHandler("api/status", func(ctx context.Context, in *daprCommon.BindingEvent) (out []byte, err error) {
			return []byte(s.Status()), nil
		})

		s.RegisterBindingInvocationHandler("api/excel", func(ctx context.Context, in *daprCommon.BindingEvent) (out []byte, err error) {
			return GetExcelData(string(in.Data)), nil
		})
	}

	logger.Info("[server] pre init success")

	// 启动流程5: 核心逻辑, 启动dapr service和dapr client等
	if err := s.InitCore(); err != nil {
		return err
	}

	// s.InitConfigCenter()

	if err := s.OnPostInit(); err != nil {
		return err
	}
	utils.GoSafeRunNoError(func() {
		t := time.NewTicker(time.Second * time.Duration(conf.Base().ServerHeartbeatInterval))
		defer t.Stop()
		for {
			select {
			case <-t.C:
				utils.SafeRunNoError(s.OnUpdateStatus)
			}
		}
	})

	szLog := fmt.Sprintf("server start success, appid:%s version:%s rolling:%s", global.AppID, global.APP_VERSION, global.ROLLING_VERSION)
	logger.Info(szLog)
	// todo 服务启动通知
	return nil
}

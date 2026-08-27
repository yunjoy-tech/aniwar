package main

import (
	gameConf "github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/gmeta"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/web"
	"robot/client"
	"robot/conf"
)

func main() {
	err := gameConf.LoadConf("conf/server.yaml")
	if err != nil {
		panic(err)
	}

	// 加载配置
	if err = conf.Init(); err != nil {
		panic(err)
	}

	// logger初始化
	if err = logger.Init(conf.GetConf().LogName); err != nil {
		panic(err)
	}

	if err = gmeta.GetMetaMgr().LoadAllMeta(); err != nil {
		panic(err)
	}

	// pprof
	web.PProfServerStart("0.0.0.0:20004")

	// 初始化客户端
	mgr := client.NewClientMgr(conf.GetConf().ClientNum)
	mgr.Start()

	select {}
}

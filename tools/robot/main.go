package main

import (
	gameConf "gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/gmeta"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/web"
	"gitee.com/aniwar2/robot/client"
	"gitee.com/aniwar2/robot/conf"
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

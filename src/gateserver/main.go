package main

import (
	"gitee.com/aniwar2/musae/framework/elog"
	"gitee.com/aniwar2/musae/framework/process"
	"gitee.com/bychannel/aniwar/src/gateserver/logic"
)

func main() {
	elog.RedirectStderr("gate")
	process.Start(logic.NewGateServer())
}

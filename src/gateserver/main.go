package main

import (
	"gitee.com/bychannel/aniwar/src/gateserver/logic"
	"gitee.com/bychannel/musae/framework/elog"
	"gitee.com/bychannel/musae/framework/process"
)

func main() {
	elog.RedirectStderr("gate")
	process.Start(logic.NewGateServer())
}

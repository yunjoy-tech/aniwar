package main

import (
	"gitee.com/aniwar2/aniwar/src/gateserver/logic"
	"gitee.com/aniwar2/musae/framework/elog"
	"gitee.com/aniwar2/musae/framework/process"
)

func main() {
	elog.RedirectStderr("gate")
	process.Start(logic.NewGateServer())
}

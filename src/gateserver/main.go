package main

import (
	"gitee.com/aniwar2/aniwar/src/gateserver/logic"
	"gitee.com/aniwar2/musae/elog"
	"gitee.com/aniwar2/musae/process"
)

func main() {
	elog.RedirectStderr("gate")
	process.Start(logic.NewGateServer())
}

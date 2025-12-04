package main

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/gateserver/logic"
	"gitlab.musadisca-games.com/wangxw/musae/framework/elog"
	"gitlab.musadisca-games.com/wangxw/musae/framework/process"
)

func main() {
	elog.RedirectStderr("gate")
	process.Start(logic.NewGateServer())
}

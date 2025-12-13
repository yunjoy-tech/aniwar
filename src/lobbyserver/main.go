package main

import (
	"gitee.com/bychannel/aniwar/src/lobbyserver/logic"
	"gitee.com/bychannel/musae/framework/process"
)

func main() {
	process.Start(logic.NewLobbyServer())
}

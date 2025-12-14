package main

import (
	"gitee.com/aniwar2/musae/framework/process"
	"gitee.com/bychannel/aniwar/src/lobbyserver/logic"
)

func main() {
	process.Start(logic.NewLobbyServer())
}

package main

import (
	"gitee.com/aniwar2/aniwar/src/lobbyserver/logic"
	"gitee.com/aniwar2/musae/framework/process"
)

func main() {
	process.Start(logic.NewLobbyServer())
}

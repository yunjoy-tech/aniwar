package main

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/lobbyserver/logic"
	"gitlab.musadisca-games.com/wangxw/musae/framework/process"
)

func main() {
	process.Start(logic.NewLobbyServer())
}

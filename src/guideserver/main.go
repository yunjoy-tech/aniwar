package main

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/guideserver/logic"
	"gitlab.musadisca-games.com/wangxw/musae/framework/process"
)

func main() {
	process.Start(logic.NewGuideServer())
}

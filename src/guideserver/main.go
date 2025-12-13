package main

import (
	"gitee.com/bychannel/aniwar/src/guideserver/logic"
	"gitee.com/bychannel/musae/framework/process"
)

func main() {
	process.Start(logic.NewGuideServer())
}

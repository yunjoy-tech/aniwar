package main

import (
	"gitee.com/aniwar2/musae/framework/process"
	"gitee.com/bychannel/aniwar/src/idipserver/logic"
)

func main() {
	process.Start(logic.NewIDIPServer())
}

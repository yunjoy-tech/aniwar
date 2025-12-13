package main

import (
	"gitee.com/bychannel/aniwar/src/idipserver/logic"
	"gitee.com/bychannel/musae/framework/process"
)

func main() {
	process.Start(logic.NewIDIPServer())
}

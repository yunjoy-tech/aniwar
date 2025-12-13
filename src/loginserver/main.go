package main

import (
	"gitee.com/bychannel/aniwar/src/loginserver/logic"
	"gitee.com/bychannel/musae/framework/process"
)

func main() {
	process.Start(logic.NewLoginServer())
}

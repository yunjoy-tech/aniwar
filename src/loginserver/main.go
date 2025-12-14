package main

import (
	"gitee.com/aniwar2/aniwar/src/loginserver/logic"
	"gitee.com/aniwar2/musae/framework/process"
)

func main() {
	process.Start(logic.NewLoginServer())
}

package main

import (
	"gitee.com/aniwar2/aniwar/src/gateserver/logic"
	"gitee.com/aniwar2/musae/process"
)

func main() {
	process.Start(logic.NewGateServer())
}

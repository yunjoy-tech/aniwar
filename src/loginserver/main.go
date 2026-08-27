package main

import (
	"github.com/yunjoy-tech/aniwar/src/loginserver/logic"
	"github.com/yunjoy-tech/musae/process"
)

func main() {
	process.Start(logic.NewLoginServer())
}

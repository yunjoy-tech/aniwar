package main

import (
	"github.com/yunjoy-tech/aniwar/src/idipserver/logic"
	"github.com/yunjoy-tech/musae/process"
)

func main() {
	process.Start(logic.NewIDIPServer())
}

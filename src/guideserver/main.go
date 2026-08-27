package main

import (
	"github.com/yunjoy-tech/aniwar/src/guideserver/logic"
	"github.com/yunjoy-tech/musae/process"
)

func main() {
	process.Start(logic.NewGuideServer())
}

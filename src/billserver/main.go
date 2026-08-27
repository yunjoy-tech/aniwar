package main

import (
	"github.com/yunjoy-tech/aniwar/src/billserver/logic"
	"github.com/yunjoy-tech/musae/process"
)

func main() {
	process.Start(logic.NewBillServer())
}

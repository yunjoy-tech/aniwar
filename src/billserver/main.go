package main

import (
	"gitee.com/aniwar2/aniwar/src/billserver/logic"
	"gitee.com/aniwar2/musae/process"
)

func main() {
	process.Start(logic.NewBillServer())
}

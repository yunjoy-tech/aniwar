package main

import (
	"gitee.com/aniwar2/musae/framework/process"
	"gitee.com/bychannel/aniwar/src/billserver/logic"
)

func main() {
	process.Start(logic.NewBillServer())
}

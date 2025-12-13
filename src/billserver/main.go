package main

import (
	"gitee.com/bychannel/aniwar/src/billserver/logic"
	"gitee.com/bychannel/musae/framework/process"
)

func main() {
	process.Start(logic.NewBillServer())
}

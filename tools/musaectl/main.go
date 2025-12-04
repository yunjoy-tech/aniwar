package main

import (
	"github.com/alecthomas/kingpin/v2"
)

/*
   musaectl CLI命令行工具，提供对服务资源的增删改查工作
*/
func init() {
	init_reload()
	init_logs()
}

func main() {
	cmd := kingpin.Parse()
	switch cmd {
	case "reload":
		cmd_reload()
	case "logs":
		cmd_logs()
	default:
	}
}

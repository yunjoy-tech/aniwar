package main

import (
	"bufio"
	"fmt"
	"github.com/alecthomas/kingpin/v2"
	"os"
	"strconv"
	"strings"
)

var (
	LogFile string
	LogType string
	Check   string
	Arg     string
)

func init_logs() {
	logs := kingpin.Command("logs", "delay log")
	logs.Flag("file", "log file").Default("").StringVar(&LogFile)
	logs.Flag("type", "show type").Default("delay").StringVar(&LogType)
	logs.Flag("check", "check").Default("").StringVar(&Check)
	logs.Flag("arg", "arg").Default("").StringVar(&Arg)

}

func cmd_logs() {
	file, err := os.Open(LogFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ",")
		for _, field := range fields {

			if strings.Contains(field, LogType) {
				kv := strings.Split(field, ":")
				if len(kv) == 2 {
					var bTrue bool
					switch Check {
					case "gt":
						val, e1 := strconv.Atoi(kv[1])
						arg, e2 := strconv.Atoi(Arg)
						if e1 != nil || e2 != nil {
							continue
						}
						if val > arg {
							bTrue = true
						}
					case "lt":
						val, e1 := strconv.Atoi(kv[1])
						arg, e2 := strconv.Atoi(Arg)
						if e1 != nil || e2 != nil {
							continue
						}
						if val < arg {
							bTrue = true
						}
					case "ge":
						val, e1 := strconv.Atoi(kv[1])
						arg, e2 := strconv.Atoi(Arg)
						if e1 != nil || e2 != nil {
							continue
						}
						if val >= arg {
							bTrue = true
						}
					case "le":
						val, e1 := strconv.Atoi(kv[1])
						arg, e2 := strconv.Atoi(Arg)
						if e1 != nil || e2 != nil {
							continue
						}
						if val <= arg {
							bTrue = true
						}
					case "eq":
						if kv[1] == Arg {
							bTrue = true
						}
					case "ne":
						if kv[1] != Arg {
							bTrue = true
						}
					}

					if bTrue {
						fmt.Println(field)
					}
				}
			}
		}
	}
}

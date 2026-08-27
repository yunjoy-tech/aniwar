package conf

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

type ClientConf struct {
	HttpAddr        string     `json:"httpAddr"`
	TCPAddr         string     `json:"tcpAddr"`
	ClientNum       int        `json:"clientNum"`
	NetType         string     `json:"netType"`
	Loglevel        string     `json:"loglevel"`
	LogPath         string     `json:"logPath"`
	LogName         string     `json:"logName"`
	Ticker          int        `json:"ticker"`
	AccountTag      string     `json:"accountTag"`
	AccountBase     string     `json:"accountBase"`
	AccountStartIdx int        `json:"accountStartIdx"`
	LoginTime       int        `json:"loginTime"`
	Restart         int        `json:"restart"` // 重连概率
	RangeIndexes    [][]string `json:"rangeIndexes"`
}

var clientConf ClientConf

func GetConf() *ClientConf {
	return &clientConf
}

var conf string

func Init() error {
	// run := kingpin.Command("run", "run robot client")
	// run.Flag("conf", "robot.conf").Default("./config.json").StringVar(&conf)
	// kingpin.Parse()
	flag.StringVar(&conf, "conf", "./config.json", "robot config file path")
	flag.Parse()

	fmt.Println("conf: ", conf)

	pf, err := os.Open(conf)
	if err != nil {
		return errors.New("LoadConf load failed")
	}
	defer pf.Close()

	decoder := json.NewDecoder(pf)
	if err = decoder.Decode(&clientConf); err != nil {
		fmt.Println("got err: ", err)
		return errors.New("LoadConf Decode failed")
	}

	fmt.Printf("LoadConf: %+v\n", clientConf)
	return nil
}

package client

import (
	"fmt"
	"github.com/yunjoy-tech/musae/logger"
	"robot/conf"
	"time"
)

type ClientMgr struct {
	Count   int
	Clients map[string]*Client
}

func NewClientMgr(count int) *ClientMgr {
	c := &ClientMgr{Count: count}
	c.Clients = make(map[string]*Client)
	return c
}

func (m *ClientMgr) Start() {
	start := conf.GetConf().AccountStartIdx
	for i := start; i < (start + m.Count); i++ {
		account := fmt.Sprintf("%s%s%d", conf.GetConf().AccountTag, conf.GetConf().AccountBase, i)
		m.Clients[account] = NewClient(account)
		m.Clients[account].Start()
		logger.Debug("启动机器人: ", account)
		time.Sleep(time.Millisecond * 300)
	}
}

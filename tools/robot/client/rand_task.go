package client

import (
	randutil "gitee.com/aniwar2/musae/utils/rand"
	"gitee.com/aniwar2/robot/conf"
	"strconv"
)

// 配置的协议发送结构
type RandomTask struct {
	Index    int32  // 函数索引
	Rate     int    // 概率百分比
	Times    int32  // 间隔
	Params   string // 协议参数
	CurTimes int32  // 当前读秒
}

// InitRandomTask 配置的随机请求
func (c *Client) InitRandomTask() {
	c.Debug("random task start")
	// 构建数据
	for _, each := range conf.GetConf().RangeIndexes {
		if each[3] == "0" {
			continue
		}
		index, _ := strconv.Atoi(each[0])
		rate, _ := strconv.Atoi(each[1])
		times, _ := strconv.Atoi(each[2])
		c.randomTask = append(c.randomTask, &RandomTask{
			Index:    int32(index),
			Rate:     rate,
			Times:    int32(times),
			CurTimes: 0,
			Params:   each[4],
		})
	}
}

func (c *Client) sendRandomReq() {
	// 每个任务都判定一下
	for _, task := range c.randomTask {
		task.CurTimes++
		// 时间到了
		if task.CurTimes >= task.Times {
			task.CurTimes = 0
			// 判定概率
			if randutil.InRandomProbability(task.Rate, randutil.OneHundred) {
				c.Debug("随机任务:", task.Index)
				c.handleTask(task)
			}
		}
	}
}

func (c *Client) handleTask(task *RandomTask) {
	switch task.Index {
	case 91:
		c.GmHandler.ReqTestUgcCheck(task.Params)
	case 99:
		c.LoginHandler.TryLogout()
	}
}

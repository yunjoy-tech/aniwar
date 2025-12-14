package logic

import (
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/common/conf"
	"gitee.com/bychannel/aniwar/src/proto/pb"
)

var DS = &DebugStatistics{
	StatisticsStart: time.Now(),
	MsgData:         sync.Map{},
	LoseMsg:         0,
}

// debug统计总结构
type DebugStatistics struct {
	StatisticsStart time.Time
	MsgData         sync.Map
	LoseMsg         int32
}

// MsgStatistics debug模式的消息统计数据
type MsgStatistics struct {
	Counter  int32
	ByteSize int64
	Up       bool
}

func handleMsgStatistics(msgId int32, byteSize int64, up bool) {
	if conf.GConf().Base.IsDebug {
		actual, _ := DS.MsgData.LoadOrStore(msgId, &MsgStatistics{
			Counter:  0,
			ByteSize: 0,
			Up:       false,
		})
		statistics := actual.(*MsgStatistics)
		statistics.Counter += 1
		statistics.ByteSize += byteSize
		statistics.Up = up
	}
}

func (s *GateServer) printMsgStatistics() {
	if conf.GConf().Base.IsDebug {
		logger.Debug("=====================================================================")
		logger.Debug("debug模式消息统计结果:")
		now := time.Now()
		second := now.Unix() - DS.StatisticsStart.Unix()
		num := s.userMgr.UserNum()
		up, down, upSize, downSize := int32(0), int32(0), int64(0), int64(0)
		DS.MsgData.Range(func(key, value any) bool {
			msgId := key.(int32)
			statistics := value.(*MsgStatistics)
			if statistics.Up {
				up += statistics.Counter
				upSize += statistics.ByteSize
			} else {
				down += statistics.Counter
				downSize += statistics.ByteSize
			}
			logger.Debugf("消息id：%-6d, 消息名: %-40s 处理次数: %-10d 数据大小: %-10d 上行: %v", msgId, pb.Protocols_name[msgId], statistics.Counter, statistics.ByteSize, statistics.Up)
			return true
		})

		upT := up / int32(second)
		upS := int32(upSize / second)
		downT := down / int32(second)
		downS := int32(downSize / second)

		logger.Debug("统计时长:", common.FormatStr(DS.StatisticsStart, now))
		logger.Debug("在线人数:", num)
		logger.Debugf("总消息数: %d, up: %d, down: %d", up+down, up, down)
		logger.Debugf("总字节数: %d, up: %d, down: %d", upSize+downSize, upSize, downSize)
		logger.Debugf("每秒数据统计, up总消息: %d, up总字节: %d, down总消息: %d, down总字节: %d", upT, upS, downT, downS)
		if num > 0 {
			logger.Debugf("每秒人均数据统计: up总消息: %d, up总字节: %d, down总消息: %d, down总字节: %d", upT/num, upS/num, downT/num, downS/num)
		}
		logger.Debug("丢包数量:", atomic.LoadInt32(&DS.LoseMsg))
		logger.Debug("=====================================================================")
	}
}

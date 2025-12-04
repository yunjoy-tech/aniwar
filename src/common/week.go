package common

import (
	"time"
)

// GetWeekday 获取游戏时间的星期几
func GetWeekday(t *time.Time) time.Weekday {
	hour := t.Hour()
	weekday := t.Weekday()
	if hour < GAME_DAILY_REFRESH_HOUR {
		return GetPreWeekday(weekday)
	}
	return weekday
}

// GetPreWeekday 获取weekday前一天的星期几
func GetPreWeekday(weekday time.Weekday) time.Weekday {
	switch weekday {
	case time.Sunday: // 星期天(0)的前一天是星期六(6)
		return time.Saturday
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday:
		return time.Weekday(int32(weekday) - 1)
	}

	return -1
}

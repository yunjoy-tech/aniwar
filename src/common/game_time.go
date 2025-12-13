package common

import (
	"fmt"
	"time"
	// "gitee.com/bychannel/aniwar/src/common"
)

const (
	FORMAT_DATE     = "2006/01/02"          // 日期格式
	FORMAT_DATETIME = "2006/01/02 15:04:05" // 日期+时间 [[[策划统一的配置格式]]]

	// 秒数
	TIME_SEC_1_HOUR = 3600                 // 一小时
	TIME_SEC_1_DAY  = 24 * TIME_SEC_1_HOUR // 一天
	TIME_SEC_1_YEAR = 365 * TIME_SEC_1_DAY // 一年
)

// ParseDate 字符串转时间
func ParseDate(dateStr string) (time.Time, error) {
	t, err := time.ParseInLocation(FORMAT_DATETIME, dateStr, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// ParseDate2 字符串转时间
func ParseDate2(year, month, day, hour, minute, second int) (time.Time, error) {
	dateStr := fmt.Sprintf("%04d/%02d/%02d %02d:%02d:%02d", year, month, day, hour, minute, second)
	// fmt.Println(dateStr)
	return ParseDate(dateStr)
}

// FormatDateByUnix 时间戳转日期字符串
func FormatDateByUnix(unix int64) string {
	timeVal := time.Unix(unix, 0)
	return timeVal.Format(FORMAT_DATETIME)
}

// IsSameDay 判断是否同一天
func IsSameDay(d1, d2 time.Time) bool {
	return IsSameDayByOffset(d1, d2, 0)
}

// IsSameDayByOffset 是否同一天，带小时偏移值
func IsSameDayByOffset(d1, d2 time.Time, offset int32) bool {
	d1 = d1.Add(-time.Hour * time.Duration(offset))
	d2 = d2.Add(-time.Hour * time.Duration(offset))

	return d1.Year() == d2.Year() && d1.Month() == d2.Month() && d1.Day() == d2.Day()
}

// IsSameWeek 判断是否同一周
func IsSameWeek(d1, d2 time.Time) bool {
	return IsSameWeekByOffset(d1, d2, 0)
}

// IsSameWeekByOffset 判断是否同一周，带小时偏移值
func IsSameWeekByOffset(d1, d2 time.Time, offset int32) bool {
	d1 = d1.Add(-time.Hour * time.Duration(offset))
	d2 = d2.Add(-time.Hour * time.Duration(offset))

	year1, week1 := d1.ISOWeek()
	year2, week2 := d2.ISOWeek()

	return year1 == year2 && week1 == week2
}

// IsSameMonth 判断是否同一个月
func IsSameMonth(d1, d2 time.Time) bool {
	return IsSameMonthByOffset(d1, d2, 0)
}

// IsSameMonthByOffset 判断是否同一个月，带小时偏移值
func IsSameMonthByOffset(d1, d2 time.Time, offset int32) bool {
	d1 = d1.Add(-time.Hour * time.Duration(offset))
	d2 = d2.Add(-time.Hour * time.Duration(offset))

	return d1.Year() == d2.Year() && d1.Month() == d2.Month()
}

// GetNextDailyRefreshTime 获取下一次刷新的时间
func GetNextDailyRefreshTime() int64 {
	return GetNextNDailyRefreshTime(1)
}

// GetNextNDailyRefreshTime 获取下N天刷新的时间
func GetNextNDailyRefreshTime(n int32) int64 {
	now := time.Now()
	refreshTime := GetTodayRefreshTime(now)
	if now.Before(refreshTime) {
		return refreshTime.Unix()
	}
	return refreshTime.AddDate(0, 0, int(1*n)).Unix()
}

// ToGameTime 转为游戏时间
func ToGameTime(t time.Time) time.Time {
	return t.Add(-1 * time.Hour * time.Duration(GAME_DAILY_REFRESH_HOUR)) // 回拨到当天的日期
}

// GetTodayRefreshTime 获取当天刷新的时间
func GetTodayRefreshTime(t time.Time) time.Time {
	offsetT := ToGameTime(t)
	return time.Date(offsetT.Year(), offsetT.Month(), offsetT.Day(), GAME_DAILY_REFRESH_HOUR, 0, 0, 0, offsetT.Location())
}

// GetMondayOfWeek 获取本周周一的日期
func GetMondayOfWeek(t time.Time) (dayStr string) {
	dayObj := GetTodayRefreshTime(t)
	if t.Weekday() == time.Monday {
		dayStr = dayObj.Format(FORMAT_DATETIME)
	} else {
		offset := int(time.Monday - t.Weekday())
		if offset > 0 {
			offset = -6
		}
		dayStr = dayObj.AddDate(0, 0, offset).Format(FORMAT_DATETIME)
	}
	return
}

// GetLastNWeekMonday 获取上N周周一日期
func GetLastNWeekMonday(t time.Time, n int) (ret time.Time, err error) {
	monday := GetMondayOfWeek(t)
	dayObj, err := time.Parse(FORMAT_DATETIME, monday)
	if err != nil {
		return
	}

	return dayObj.AddDate(0, 0, -7*n), nil
}

// GetNextNWeekMonday 获取下N周周一日期
func GetNextNWeekMonday(t time.Time, n int) (ret time.Time, err error) {
	monday := GetMondayOfWeek(t)
	dayObj, err := time.Parse(FORMAT_DATETIME, monday)
	if err != nil {
		return
	}
	return dayObj.AddDate(0, 0, 7*n), nil
}

// GetMonth1RefreshTime 当月1号刷新时间
func GetMonth1RefreshTime(t time.Time) (ret time.Time, err error) {
	dayObj := GetTodayRefreshTime(t)

	// 本月1号
	return time.Date(dayObj.Year(), dayObj.Month(), 1,
		dayObj.Hour(), dayObj.Minute(), dayObj.Second(), dayObj.Nanosecond(),
		dayObj.Location()), nil
}

// GetLastNMonth1RefreshTime 前N月1号的刷新时间
func GetLastNMonth1RefreshTime(t time.Time, n int) (ret time.Time, err error) {
	// 本月1号
	currMonthOne, err := GetMonth1RefreshTime(t)
	if err != nil {
		return
	}

	ret = currMonthOne.AddDate(0, -1*n, 0)
	return
}

// GetNextNMonth1RefreshTime 后N月1号刷新时间
func GetNextNMonth1RefreshTime(t time.Time, n int) (ret time.Time, err error) {
	// 本月1号
	currMonthOne, err := GetMonth1RefreshTime(t)
	if err != nil {
		return
	}

	ret = currMonthOne.AddDate(0, 1*n, 0)
	return
}

func FormatStr(start, end time.Time) string {
	if start.After(end) {
		return ""
	}
	// 经过的秒数
	total := end.Unix() - start.Unix()
	day := total / (24 * 60 * 60)
	hour := (total - 24*60*60*day) / (60 * 60)
	minute := (total - 24*60*60*day - 60*60*hour) / 60
	second := total - 24*60*60*day - 60*60*hour - 60*minute
	return fmt.Sprintf("%d日%d时%d分%d秒", day, hour, minute, second)
}

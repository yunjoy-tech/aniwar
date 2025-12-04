package common

import (
	"fmt"
	"testing"
	"time"
)

func Test_IsSameDayByOffset(t *testing.T) {
	dTime := time.Now()
	isToday := IsSameDayByOffset(dTime, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间：%s, 当前的时间%s === IsSameDayByOffset 结果：%t", dTime, time.Now(), isToday))

	println("#################")
	year, month, day := time.Now().Date()
	hour := 11
	minute := 59
	second := 59
	date_hour235959, _ := ParseDate2(year, int(month), day, hour, minute, second)
	isToday = IsSameDayByOffset(date_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_hour235959, time.Now(), isToday))

	year1 := year + 1
	date_year_hour235959, _ := ParseDate2(year1, int(month), day, hour, minute, second)
	isToday = IsSameDayByOffset(date_year_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_year_hour235959, time.Now(), isToday))

	year2 := year - 1
	date_year1_hour235959, _ := ParseDate2(year2, int(month), day, hour, minute, second)
	isToday = IsSameDayByOffset(date_year1_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_year1_hour235959, time.Now(), isToday))

	month1 := month - 1
	date_month1_hour235959, _ := ParseDate2(year, int(month1), day, hour, minute, second)
	isToday = IsSameDayByOffset(date_month1_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_month1_hour235959, time.Now(), isToday))

	month2 := month + 1
	date_month2_hour235959, _ := ParseDate2(year, int(month2), day, hour, minute, second)
	isToday = IsSameDayByOffset(date_month2_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_month2_hour235959, time.Now(), isToday))

	day1 := day - 1
	date_day1_hour235959, _ := ParseDate2(year, int(month), day1, hour, minute, second)
	isToday = IsSameDayByOffset(date_day1_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_day1_hour235959, time.Now(), isToday))

	day2 := day + 1
	date_day2_hour235959, _ := ParseDate2(year, int(month), day2, hour, minute, second)
	isToday = IsSameDayByOffset(date_day2_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_day2_hour235959, time.Now(), isToday))

	hour1 := 1
	date_hour1_hour235959, _ := ParseDate2(year, int(month), day, hour1, minute, second)
	isToday = IsSameDayByOffset(date_hour1_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_hour1_hour235959, time.Now(), isToday))

	hour4 := 4
	date_hour4_hour235959, _ := ParseDate2(year, int(month), day, hour4, minute, second)
	isToday = IsSameDayByOffset(date_hour4_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_hour4_hour235959, time.Now(), isToday))

	hour50000 := 5
	date_hour50000_hour235959, _ := ParseDate2(year, int(month), day, hour50000, minute, second)
	isToday = IsSameDayByOffset(date_hour50000_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_hour50000_hour235959, time.Now(), isToday))

	hour6 := 6
	date_hour6_hour235959, _ := ParseDate2(year, int(month), day, hour6, minute, second)
	isToday = IsSameDayByOffset(date_hour6_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", date_hour6_hour235959, time.Now(), isToday))

	d1, _ := ParseDate2(2022, 8, 30, 3, 0, 0)
	d2, _ := ParseDate2(2022, 8, 30, 4, 0, 0)
	isToday = IsSameDayByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameDayByOffset 结果：%t", d1, d2, isToday))

}
func Test_IsSameWeekByOffset(t *testing.T) {
	dTime := time.Now()
	r := IsSameWeekByOffset(dTime, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间：%s, 当前的时间%s === IsSameWeekByOffset 结果：%t", dTime, time.Now(), r))

	println("#################")
	year, month, day := time.Now().Date()
	hour := 11
	minute := 59
	second := 59
	date_hour235959, _ := ParseDate2(year, int(month), day, hour, minute, second)
	r = IsSameWeekByOffset(date_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", date_hour235959, time.Now(), r))

	year1 := year + 1
	date_year_hour235959, _ := ParseDate2(year1, int(month), day, hour, minute, second)
	r = IsSameWeekByOffset(date_year_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", date_year_hour235959, time.Now(), r))

	month2 := month + 1
	date_month2_hour235959, _ := ParseDate2(year, int(month2), day, hour, minute, second)
	r = IsSameWeekByOffset(date_month2_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", date_month2_hour235959, time.Now(), r))

	day2 := day + 1
	date_day2_hour235959, _ := ParseDate2(year, int(month), day2, hour, minute, second)
	r = IsSameWeekByOffset(date_day2_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", date_day2_hour235959, time.Now(), r))

	d1, _ := ParseDate2(2022, 8, 27, 11, 0, 0)
	d2, _ := ParseDate2(2022, 8, 28, 6, 0, 0)
	r = IsSameWeekByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", d1, d2, r))

	d1, _ = ParseDate2(2022, 8, 28, 11, 0, 0)
	d2, _ = ParseDate2(2022, 8, 29, 6, 0, 0)
	r = IsSameWeekByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", d1, d2, r))

	d1, _ = ParseDate2(2022, 8, 29, 4, 0, 0)
	d2, _ = ParseDate2(2022, 8, 29, 6, 0, 0)
	r = IsSameWeekByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", d1, d2, r))

	d1, _ = ParseDate2(2022, 8, 29, 5, 0, 0)
	d2, _ = ParseDate2(2022, 8, 29, 6, 0, 0)
	r = IsSameWeekByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameWeekByOffset 结果：%t", d1, d2, r))

}

func Test_IsSameMonthByOffset(t *testing.T) {
	dTime := time.Now()
	r := IsSameMonthByOffset(dTime, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间：%s, 当前的时间%s === IsSameMonthByOffset 结果：%t", dTime, time.Now(), r))

	println("#################")
	year, month, day := time.Now().Date()
	hour := 11
	minute := 59
	second := 59
	date_hour235959, _ := ParseDate2(year, int(month), day, hour, minute, second)
	r = IsSameMonthByOffset(date_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", date_hour235959, time.Now(), r))

	year1 := year + 1
	date_year_hour235959, _ := ParseDate2(year1, int(month), day, hour, minute, second)
	r = IsSameMonthByOffset(date_year_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", date_year_hour235959, time.Now(), r))

	month2 := month + 1
	date_month2_hour235959, _ := ParseDate2(year, int(month2), day, hour, minute, second)
	r = IsSameMonthByOffset(date_month2_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", date_month2_hour235959, time.Now(), r))

	day2 := day + 1
	date_day2_hour235959, _ := ParseDate2(year, int(month), day2, hour, minute, second)
	r = IsSameMonthByOffset(date_day2_hour235959, time.Now(), GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", date_day2_hour235959, time.Now(), r))

	d1, _ := ParseDate2(2022, 8, 31, 11, 0, 0)
	d2, _ := ParseDate2(2022, 9, 1, 6, 0, 0)
	r = IsSameMonthByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", d1, d2, r))

	d1, _ = ParseDate2(2022, 9, 1, 4, 0, 0)
	d2, _ = ParseDate2(2022, 9, 1, 6, 0, 0)
	r = IsSameMonthByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", d1, d2, r))

	d1, _ = ParseDate2(2022, 9, 1, 5, 0, 0)
	d2, _ = ParseDate2(2022, 9, 1, 6, 0, 0)
	r = IsSameMonthByOffset(d1, d2, GAME_DAILY_REFRESH_HOUR)
	fmt.Println(fmt.Sprintf("验证的时间:%s, 当前的时间:%s === IsSameMonthByOffset 结果：%t", d1, d2, r))

}

func TestGetDailyRefreshTime(t *testing.T) {
	//now := time.Now()
	d1, _ := ParseDate2(2023, 3, 16, 4, 0, 0)

	refreshTime := GetTodayRefreshTime(d1)
	fmt.Println("now: ", d1, " refreshTime: ", refreshTime) //now:  2023-03-16 04:00:00 +0800 CST  refreshTime:  2023-03-15 05:00:00 +0800 CST
}

func TestGetMondayOfWeek(t *testing.T) {
	d1, _ := ParseDate2(2022, 8, 21, 4, 0, 0)
	d2, _ := ParseDate2(2022, 8, 22, 5, 0, 0)
	d3, _ := ParseDate2(2022, 8, 23, 6, 0, 0)
	d4, _ := ParseDate2(2022, 8, 28, 7, 0, 0)

	fmt.Println("d1: ", d1, " GetMondayOfWeek(d1): ", GetMondayOfWeek(d1))
	fmt.Println("d2: ", d2, " GetMondayOfWeek(d2): ", GetMondayOfWeek(d2))
	fmt.Println("d3: ", d3, " GetMondayOfWeek(d3): ", GetMondayOfWeek(d3))
	fmt.Println("d4: ", d4, " GetMondayOfWeek(d4): ", GetMondayOfWeek(d4))
}

func TestGetLastNWeekMonday(t *testing.T) {
	now := time.Now()
	fmt.Println("now: ", now)

	ret0, _ := GetLastNWeekMonday(now, 0)
	fmt.Println("ret0: ", ret0)
	ret1, _ := GetLastNWeekMonday(now, 1)
	fmt.Println("ret1: ", ret1)
	ret2, _ := GetLastNWeekMonday(now, 2)
	fmt.Println("ret2: ", ret2)
}

func TestGetNextNWeekMonday(t *testing.T) {
	now := time.Now()
	fmt.Println("now: ", now)

	ret0, _ := GetNextNWeekMonday(now, 0)
	fmt.Println("ret0: ", ret0)
	ret1, _ := GetNextNWeekMonday(now, 1)
	fmt.Println("ret1: ", ret1)
	ret2, _ := GetNextNWeekMonday(now, 2)
	fmt.Println("ret2: ", ret2)
}

func Test_GetMonth1RefreshTime(t *testing.T) {
	now := time.Now()
	fmt.Println("now: ", now)

	ret0, _ := GetMonth1RefreshTime(now)
	fmt.Println("ret0: ", ret0)
}

func Test_GetLastNMonth1RefreshTime(t *testing.T) {
	now := time.Now()
	fmt.Println("now: ", now)

	ret0, _ := GetLastNMonth1RefreshTime(now, 0)
	fmt.Println("ret0: ", ret0)
	ret1, _ := GetLastNMonth1RefreshTime(now, 1)
	fmt.Println("ret1: ", ret1)
	ret2, _ := GetLastNMonth1RefreshTime(now, 10)
	fmt.Println("ret2: ", ret2)
}

func Test_GetNextNMonth1RefreshTime(t *testing.T) {
	now := time.Now()
	fmt.Println("now: ", now)

	ret0, _ := GetNextNMonth1RefreshTime(now, 0)
	fmt.Println("ret0: ", ret0)
	ret1, _ := GetNextNMonth1RefreshTime(now, 1)
	fmt.Println("ret1: ", ret1)
	ret2, _ := GetNextNMonth1RefreshTime(now, 2)
	fmt.Println("ret2: ", ret2)
	ret5, _ := GetNextNMonth1RefreshTime(now, 5)
	fmt.Println("ret5: ", ret5)

}

func TestFormatStr(t *testing.T) {
	d1, _ := ParseDate2(2022, 8, 21, 4, 1, 0)
	d2, _ := ParseDate2(2022, 8, 22, 3, 0, 0)
	fmt.Println(FormatStr(d1, d2))
}

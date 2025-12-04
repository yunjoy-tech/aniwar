package common

import (
	"fmt"
	"testing"
)

// GetWeekday 获取游戏时间的星期几
func Test_GetWeekday(t *testing.T) {
	//now := time.Now()
	d1, _ := ParseDate2(2023, 1, 4, 5, 0, 0)
	weekday := GetWeekday(&d1)

	showStr := fmt.Sprintf("日期时间:%s, 是星期%s, 昨天是星期%s", d1.Format(FORMAT_DATETIME), weekday.String(), GetPreWeekday(weekday))
	fmt.Println(showStr)

}

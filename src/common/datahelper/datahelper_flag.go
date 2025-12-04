package datahelper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	STORY_FLAG_V         = 0   // 配表默认值
	STORY_FLAG_SEPARATOR = ":" // 配表分隔符
)

// 获取flag 和 其值
func GetFlagVal(flagStr string) (error, string, int32) {
	var (
		err  error
		flag = ""
		val  = STORY_FLAG_V // 默认值
	)

	splits := strings.Split(flagStr, STORY_FLAG_SEPARATOR)
	flag = splits[0]
	if len(splits) > 1 {
		val, err = strconv.Atoi(splits[1])
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("flag:%s, 获取值失败, 不能转成数字", flagStr))
			return err, flag, int32(val)
		}
	}

	return nil, flag, int32(val)
}

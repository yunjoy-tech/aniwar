package datahelper

import (
	"fmt"
	"time"

	"gitee.com/bychannel/aniwar/src/common"

	"gitee.com/bychannel/aniwar/src/excel/data"
	"github.com/pkg/errors"
)

func GetActivityCfgTime(activityId int32) (time.Time, time.Time, time.Time, time.Time, error) {
	var timeNil time.Time

	activityCfg := data.GetActivityMgr().GetById(activityId)
	if activityCfg == nil {
		return timeNil, timeNil, timeNil, timeNil, errors.New(fmt.Sprintf("cfg is nil, activityId=%d", activityId))
	}

	beginTime, err := common.ParseDate(activityCfg.Begintime)
	if err != nil {
		return timeNil, timeNil, timeNil, timeNil, err
	}

	endTime, err := common.ParseDate(activityCfg.Endtime)
	if err != nil {
		return timeNil, timeNil, timeNil, timeNil, err
	}

	// showTime
	LastTime, err := common.ParseDate(activityCfg.Lasttime)
	if err != nil {
		return timeNil, timeNil, timeNil, timeNil, err
	}

	ShowLastTime, err := common.ParseDate(activityCfg.Showlasttime)
	if err != nil {
		return timeNil, timeNil, timeNil, timeNil, err
	}

	return LastTime, beginTime, endTime, ShowLastTime, nil
}

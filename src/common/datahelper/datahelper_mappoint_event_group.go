package datahelper

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// GetEventGroupList 根据事件获取事件组列表
func GetEventGroupList(events []*cmd.MappointEvent) []*cmd.MappointEventGroupInfo {
	var (
		eventGroupMap  = make(map[int32]*cmd.MappointEventGroupInfo, 0)
		eventGroupList = make([]*cmd.MappointEventGroupInfo, 0)
	)

	for _, event := range events {
		mappointEventCfg := data.GetMappointEventMgr().GetById(event.EventId)
		if mappointEventCfg == nil {
			continue
		}

		eventGroupId := mappointEventCfg.GroupId
		if _, ok := eventGroupMap[eventGroupId]; !ok { // 事件组信息无需重复
			eventGroupMap[eventGroupId] = &cmd.MappointEventGroupInfo{
				GroupId:       eventGroupId,
				NextUpdateSec: common.NIWA_EVENT_GROUP_CD, // 初始值
			}
		}
	}

	for _, each := range eventGroupMap {
		eventGroupList = append(eventGroupList, each)
	}

	return eventGroupList
}

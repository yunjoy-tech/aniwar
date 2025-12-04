package datahelper

//// GetMappointEventFrozen 获取地图固定事件
//func GetMappointEventFrozen(niwaId int32, mapEvents []*cmd.MappointEvent) ([]*cmd.MappointEvent, []*cmd.MappointEventGroupInfo) {
//	var (
//		events = make([]*cmd.MappointEvent, 0)
//	)
//
//	data.GetDungeonentranceMgr().Foreach(func(cfg *data.DungeonentranceCfg) bool {
//		if cfg.NiwaId != niwaId {
//			return false
//		}
//
//		// 重复的事件
//		hadFound := false
//		for _, mapEvent := range mapEvents {
//			if mapEvent.EventId == cfg.Id {
//				hadFound = true
//				break
//			}
//		}
//		if hadFound {
//			logger.Infof("地图:%d, 地图上已经存在该事件:%d", niwaId, cfg.Id)
//			return false
//		}
//
//		eventData := &cmd.MappointEvent{
//			EventId: cfg.Id,
//			PosIdx:  cfg.BornHierarchy,
//		}
//		events = append(events, eventData)
//
//		return true
//
//	}, true)
//
//	logger.Infof("地图%d, 固定事件:%v", niwaId, events)
//
//	return events, GetEventGroupList(events)
//}

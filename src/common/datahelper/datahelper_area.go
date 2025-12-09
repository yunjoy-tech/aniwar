package datahelper

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
)

// CheckEventIdCanExistInArea 检查地图中是否有可能出现指定事件
func CheckEventIdCanExistInArea(niwaId, eventId int32) (error, cmd.ErrorCode) {
	// hadFound := false
	// data.GetAreaMgr().Foreach(func(cfg *data.AreaCfg) bool {
	// 	if niwaId == cfg.GetNiwaId() {
	// 		for eachEventId, weight := range cfg.BattleId {
	// 			if eachEventId == eventId && weight > 0 {
	// 				hadFound = true
	// 				return false // 不继续执行遍历
	// 			}
	// 		}
	// 	}
	// 	return true
	// }, false)
	//
	// if !hadFound {
	// 	return fmt.Errorf("无效的事件, 当前地图=%d, %s中不存在该事件id=%d",
	// 			niwaId, data.GetAreaMgr().GetSheetName(), eventId),
	// 		cmd.ErrorCode_InvalidParam
	// }

	return nil, cmd.ErrorCode_Success
}

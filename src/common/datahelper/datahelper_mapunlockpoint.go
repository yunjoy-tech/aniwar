package datahelper

import "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"

// GetDefaultUnlockCfgs 获取默认解锁的点
func GetDefaultUnlockCfgs(currLevelId int32) []*data.MapunlockpointCfg {
	var (
		cfgs = make([]*data.MapunlockpointCfg, 0)
	)

	data.GetMapunlockpointMgr().Foreach(func(cfg *data.MapunlockpointCfg) bool {
		if cfg.StageId != currLevelId {
			return false
		}
		if cfg.DefaultUnlock == 0 {
			return false
		}

		cfgs = append(cfgs, cfg)
		return true

	}, true)

	return cfgs
}

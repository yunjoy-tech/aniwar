package useractor

import excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"

// GetCampExchange 获取家具转换道具
func GetCampExchange(quality int32) *excel.KeyVal {
	if int32(len(excel.GetConfigMgr().GetCfg().GACHA_CAMP_EXCHANGE)) < quality {
		return nil
	}
	return excel.GetConfigMgr().GetCfg().GACHA_CAMP_EXCHANGE[quality-1]
}

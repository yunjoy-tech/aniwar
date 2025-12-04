package useractor

import (
	"fmt"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"time"
)

// LightingComposeTree 光和树
type LightingComposeTree struct {
	BaseBuilding
}

func NewLightingComposeTree() IBuilding {
	return &LightingComposeTree{
		BaseBuilding: BaseBuilding{},
	}
}

func (lt *LightingComposeTree) Build(commonParams *OutputParams, h *CampHandler, req *cmd.C2LS_PlayerCampMakeFunctionBuildingReq, buildLevelConfig *excel.BuildingLevelCfg) (*cmd.PPlayerCampCommonBuilding, error, int32) {
	//创建
	building := h.NewPPlayerCampCommonBuilding(req.GetX(), req.GetY(), req.GetParentId(), req.GetParentGridId(), req.GetEdge(), req.GetFlip(), nil)
	building.Building = h.NewPPlayerCampDecorationBuilding(req.ItemId, 1, h.actor.Data.Camp.CurrentCampId)
	if commonParams.layout.Building == nil {
		commonParams.layout.Building = map[int64]*cmd.PPlayerCampCommonBuilding{}
	}

	formulaCfg := h.getFormulaByCfg(buildLevelConfig, false)
	if len(formulaCfg) == 0 {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}
	endTimeStamp := make([]*cmd.ComposeTreeProductEndTime, 0)
	for _, v := range formulaCfg {
		for _, p := range v.ItemProduct {
			endTimeStamp = append(endTimeStamp, &cmd.ComposeTreeProductEndTime{
				ItemId:       p.ItemId,
				EndTimestamp: time.Now().Unix() + int64(v.TimeCost),
			})
		}
	}
	commonParams.camp.LightingComposeTree = &cmd.PPlayerCampLightingComposeTree{
		BuildingId:       building.Building.BuildingId,
		Level:            1,
		EndTimestampList: endTimeStamp, // time.Now().Unix() + timeCost
	}
	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{LightingComposeTree: commonParams.camp.LightingComposeTree})

	commonParams.layout.Building[building.Building.BuildingId] = building
	h.actor.GetCampData().DecorationBuilding[building.Building.BuildingId] = building.Building
	return building, nil, int32(cmd.ErrorCode_Success)
}
func (lt *LightingComposeTree) LevelUp(commonParams *OutputParams, campRet *cmd.PPlayerCamp, h *CampHandler) (error, int32) {
	curLv := commonParams.buildLevelConfig.Id - 1
	curLvCfg := excel.GetBuildingLevelMgr().GetById(curLv)
	if curLvCfg == nil {
		return fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}
	preLvFormula := h.getFormulaByCfg(curLvCfg, false)
	nextLvFormula := h.getFormulaByCfg(commonParams.buildLevelConfig, false)
	if len(preLvFormula) == 0 || len(nextLvFormula) == 0 {
		return fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	var curLimit, nextLimit, totalTimeCost int64
	now := time.Now().Unix()
	endTime := make([]*cmd.ComposeTreeProductEndTime, 0)
	for _, v := range commonParams.camp.LightingComposeTree.EndTimestampList {
		itemId := v.GetItemId()
		curLimit, totalTimeCost = lt.GetMaxNum(preLvFormula, itemId)
		nextLimit, _ = lt.GetMaxNum(nextLvFormula, itemId)
		curLvTimestamp := v.GetEndTimestamp()
		var remainReward int64
		var seconds int64
		if now >= curLvTimestamp { // 已经过了当前营地的时间
			// 获取超过的部分
			remainReward = nextLimit - curLimit
		} else {
			// 已经消耗时间
			tm := totalTimeCost - (curLvTimestamp - now)
			// 已经获得的奖励
			curPerPointTimeCost := float64(totalTimeCost) / float64(curLimit) // 一个花费的时间
			rewardNum := int64(float64(tm) / curPerPointTimeCost)             // 已经获得的奖励

			//多出来的秒数
			tmp := float64(tm) - float64(rewardNum)*curPerPointTimeCost
			seconds = int64(tmp * float64(curLimit) / float64(nextLimit))
			remainReward = nextLimit - rewardNum
		}
		//计算剩余奖励恢复所需时间
		remainCostTime := totalTimeCost * remainReward / nextLimit
		//不足一秒，按一秒算
		if totalTimeCost*remainReward%nextLimit != 0 {
			remainCostTime++
		}
		//减去已经消费的时间，但该时间不足恢复一点
		remainCostTime -= seconds
		endTime = append(endTime, &cmd.ComposeTreeProductEndTime{ItemId: v.GetItemId(), EndTimestamp: now + remainCostTime})
	}
	commonParams.camp.LightingComposeTree.EndTimestampList = endTime
	commonParams.camp.LightingComposeTree.Level++
	commonParams.building.BuildingLevel++
	campRet.LightingComposeTree = commonParams.camp.LightingComposeTree
	return nil, int32(cmd.ErrorCode_Success)
}

func (lt *LightingComposeTree) GetMaxNum(formula map[int32]*excel.ItemSynthesisCfg, itemId int32) (int64, int64) {
	for _, v := range formula {
		for _, p := range v.ItemProduct {
			if p.ItemId == itemId {
				return int64(p.GetNum()), int64(v.TimeCost)
			}
		}
	}
	return 0, 0
}

//func Old(preLvFormula *excel.ItemSynthesisCfg, nextLvFormula *excel.ItemSynthesisCfg) {
//	//获取第一个产物的最大数量
//	//for _, v := range preLvFormula {
//	//	if len(v.ItemProduct) == 0 {
//	//		return fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
//	//	}
//	//	totalTimeCost = int64(v.TimeCost)
//	//	curLimit = int64(v.ItemProduct[0].Num)
//	//	break
//	//}
//	//for _, v := range nextLvFormula {
//	//	if len(v.ItemProduct) == 0 {
//	//		return fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
//	//	}
//	//	nextLimit = int64(v.ItemProduct[0].Num)
//	//	break
//	//}
//
//	h.Debugf("pre level reward limit %d:%d, next level reward limit %d:%d", curLv, curLimit, curLv+1, nextLimit)
//
//	if curLimit > nextLimit {
//		return fmt.Errorf("invalid param"), int32(cmd.ErrorCode_InvalidParam)
//	}
//
//	now := time.Now().Unix()
//	//当前等级光合树恢复结束时间
//	curLvTimestamp := commonParams.camp.LightingComposeTree.EndTimestamp
//	fmt.Println(curLvTimestamp)
//	//剩余待恢复的数值
//	var remainReward int64
//	var seconds int64
//	//上一等级已经恢复满了
//	//
//	if now >= curLvTimestamp { // 已经过了当前营地的时间
//		// 获取超过的部分
//		remainReward = nextLimit - curLimit
//	} else {
//		// 已经消耗时间
//		tm := totalTimeCost - (curLvTimestamp - now)
//		// 已经获得的奖励
//		curPerPointTimeCost := float64(totalTimeCost) / float64(curLimit) // 一个花费的时间
//		rewardNum := int64(float64(tm) / curPerPointTimeCost)             // 已经获得的奖励
//
//		//多出来的秒数
//		tmp := float64(tm) - float64(rewardNum)*curPerPointTimeCost
//		seconds = int64(tmp * float64(curLimit) / float64(nextLimit))
//		remainReward = nextLimit - rewardNum
//	}
//	//计算剩余奖励恢复所需时间
//	remainCostTime := totalTimeCost * remainReward / nextLimit
//	//不足一秒，按一秒算
//	if totalTimeCost*remainReward%nextLimit != 0 {
//		remainCostTime++
//	}
//	//减去已经消费的时间，但该时间不足恢复一点
//	remainCostTime -= seconds
//}

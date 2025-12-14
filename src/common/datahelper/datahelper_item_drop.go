package datahelper

import (
	"math/rand"

	"gitee.com/aniwar2/aniwar/src/excel/data"
	"gitee.com/aniwar2/musae/framework/logger"
)

// 根据掉落id掉落奖励
func GetRewardsByDropId(dropId int32) []*data.ItemReward {
	var (
		ret = make([]*data.ItemReward, 0)
	)

	if dropId == 0 {
		return ret
	}

	// resourceCfg := data.GetResourceMgr().GetById(resourceId)
	dropConfig := data.GetDropMgr().GetById(dropId)
	if dropConfig == nil {
		logger.Warnf("无效的掉落id, dropId=%d", dropId)
		return ret
	}

	// 遍历每个掉落组
	for _, eachGroup := range dropConfig.GetGroup() {
		// 遍历次数
		for i := int32(0); i < eachGroup.Times; i++ {
			rate := rand.Int31n(100) + 1 // 概率为百分比
			// if eachGroup.Type < rate {
			//	// 未命中
			//	continue
			// }
			if eachGroup.Type >= rate {
				weightRewards := GroupRandomByWeight(eachGroup)
				ret = append(ret, weightRewards...)
			}
		}
	}

	logger.Warnf("掉落组随机-->> 最终奖励, dropId=%d, rewards=%+v",
		dropId, ret)

	return ret
}

// // 根据概率随机奖励
// func GroupRandomByRate(dropInfo *data.DropInfo) []*data.ItemReward {
//	itemRewards := make([]*data.ItemReward, 0)
//
//	/*	weightVos := make([]*data.WeightVo, 0)
//		groupCfgs := GetGroupCfgsByGroupId(dropInfo.GroupId)
//
//		// 掉落组多次掉落
//		for i := int32(0); i < dropInfo.Times; i++ {
//			//groupCfg := RandomByGroupId(dropInfo.GroupId)
//			RandomByRate(dropInfo.GroupId)
//			if groupCfg == nil {
//				continue
//			}
//
//			ret = append(ret, &data.ItemReward{
//				ItemId: groupCfg.ItemId,
//				Num:    groupCfg.ItemNum,
//			})
//		}*/
//
//	return itemRewards
// }

// 根据groupId, 权重随机出道具
func GroupRandomByWeight(dropInfo *data.DropInfo) []*data.ItemReward {
	itemRewards := make([]*data.ItemReward, 0)

	weightVos := make([]*data.WeightVo, 0)
	groupCfgs := GetGroupCfgsByGroupId(dropInfo.GroupId)
	for _, cfg := range groupCfgs {
		weightVos = append(weightVos, &data.WeightVo{
			Weight: cfg.ItemWeight,
			VoId:   cfg.Id,
		})
	}

	// // 多次掉落
	// for i := int32(0); i < dropInfo.Times; i++ {
	// 根据权重随机
	randomVo, err := RandomByWeightVo(weightVos)
	if err != nil {
		return nil
	}

	randomGroupCfg := data.GetGroupMgr().GetById(randomVo.GetVoId())
	itemRewards = append(itemRewards, &data.ItemReward{
		ItemId: randomGroupCfg.ItemId,
		Num:    randomGroupCfg.ItemNum,
	})
	// }

	return itemRewards
}

// 根据groupId, 获取配置列表
func GetGroupCfgsByGroupId(groupId int32) []*data.GroupCfg {
	groupCfgs := make([]*data.GroupCfg, 0)
	data.GetGroupMgr().Foreach(func(cfg *data.GroupCfg) bool {
		if cfg.ItemGroupId != groupId {
			return false
		}

		// weightVos = append(weightVos, &data.WeightVo{
		//	Weight: cfg.ItemWeight,
		//	VoId:   cfg.Id,
		// })
		groupCfgs = append(groupCfgs, cfg)

		return true

	}, true)

	return groupCfgs
}

// // 根据权重随机奖励
// func doGetDropRewardsByWeight(dropId int32, dropInfo *data.DropInfo, roleLv uint32, levelLv int32) []*data.ItemReward {
//	var (
//		ret = make([]*data.ItemReward, 0)
//	)
//
//	dropGroupCfg := data.GetGroupMgr().GetById(dropInfo.GetGroupId())
//	if dropGroupCfg == nil {
//		return ret
//	}
//
//	if !checkGroupCfgCondition(dropGroupCfg, dropId, roleLv, levelLv) {
//		// 不满足掉落条件
//		return ret
//	}
//
//	// 掉落组多次掉落
//	for i := int32(0); i < dropInfo.Times; i++ {
//		// 掉落物品
//		vo, err := RandomByWeightVo2(dropGroupCfg.GetContents())
//		if err != nil {
//			logger.Debugf("RandomByWeightVo2, 掉落物品报错, err:%+v", err)
//			return make([]*data.ItemReward, 0)
//		}
//
//		ret = append(ret, &data.ItemReward{
//			ItemId: vo.VoId,
//			Num:    vo.Num,
//		})
//	}
//
//	return ret
// }

// // group表判断掉落奖励条件
// func checkGroupCfgCondition(dropGroupCfg *data.GroupCfg, dropId int32, roleLv uint32, levelLv int32) bool {
//	if dropGroupCfg == nil {
//		return false
//	}
//
//	// 判断是否满足掉落条件
//	for _, condition := range dropGroupCfg.GetCondition() {
//		switch condition.GetConditionId() {
//		case 1: // 玩家等级限制
//			if roleLv < uint32(condition.GetMinValue()) || roleLv > uint32(condition.GetMaxValue()) {
//				logger.Warnf("掉落组随机-->> 玩家等级限制, dropId=%d, roleLv=%d, levelLv=%d, condition=%+v",
//					dropId, roleLv, levelLv, condition)
//				return false // 本次掉落中断
//			}
//		case 2: // 关卡限制
//			if levelLv < int32(condition.GetMinValue()) || levelLv > int32(condition.GetMaxValue()) {
//				logger.Warnf("掉落组随机-->> 关卡限制, dropId=%d, roleLv=%d, levelLv=%d, condition=%+v",
//					dropId, roleLv, levelLv, condition)
//				return false // 本次掉落中断
//			}
//		default:
//			logger.Errorf("未支持的条件类型, dropId:%+v, dropGroupId:%v, condition:%+v", dropId, dropGroupCfg.GetId(), condition)
//		}
//	}
//
//	return true
// }

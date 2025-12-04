package useractor

import (
	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"strconv"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/safe"
)

func (x *UserData) GetUserItems() *cmd.PCommonItemInfos {
	itemData := x.Data.ItemData
	if itemData.Items == nil {
		itemData.Items = make(map[uint64]*cmd.PCommonItemInfo)
	}

	return x.Data.ItemData
}

// AddItems 添加道具
func (x *UserData) AddItems(uid string, reason common.ChangeReason, itemInfos ...*cmd.PCommonItemInfo) ([]*cmd.ItemReward, []*cmd.PCommonItemInfo, map[int32]int32, error) {
	var (
		changeItems = make([]*cmd.ItemReward, 0)      // 变化量
		finalItems  = make([]*cmd.PCommonItemInfo, 0) // 最终量
		limitItems  = make(map[int32]int32)           // 超上限数据
	)
	for _, item := range itemInfos {
		if item.ItemNum <= 0 {
			continue
		}

		itemCfg := excel.GetItemMgr().GetById(int32(item.BaseId))
		if itemCfg == nil {
			continue
		}
		var beforeNum, afterNum uint32
		limitNum := int32(0)
		target, exist := x.GetUserItems().Items[item.UniqueId]
		if exist && target != nil {
			beforeNum = target.ItemNum
			// 防止越界
			addedNum, err := safe.AddUint(beforeNum, item.ItemNum)
			if err != nil {
				return changeItems, finalItems, limitItems, errors.WithStack(err)
			}
			target.ItemNum = utils.Min(addedNum, uint32(itemCfg.GetNumLimit()))

			afterNum = target.ItemNum
			finalItems = append(finalItems, target)
			limitNum = int32(addedNum - uint32(itemCfg.GetNumLimit()))
		} else {
			// 原来数据中未找到，直接合并
			limitNum = int32(item.ItemNum - uint32(itemCfg.GetNumLimit()))
			item.ItemNum = utils.Min(item.ItemNum, uint32(itemCfg.GetNumLimit()))
			x.GetUserItems().Items[item.UniqueId] = item
			afterNum = item.GetItemNum()
			finalItems = append(finalItems, item)
		}

		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.ItemFlow{
		//		HeadInfo:   lilith.BuildHeadInfo(lilith.LogType_ItemFlow, uid, device),
		//		RoleId:     strconv.FormatUint(x.Data.Base.Common.RoleId, 10),
		//		ItemFlow:   "in",
		//		ItemId:     strconv.FormatUint(item.UniqueId, 10),
		//		ItemCount:  int32(item.ItemNum),
		//		Level:      int32(x.Data.Base.Common.RoleLevel),
		//		VipLevel:   0,
		//		Action:     strconv.Itoa(int(reason)),
		//		ItemBefore: int64(beforeNum),
		//		ItemAfter:  int64(afterNum),
		//		Recharge:   0,
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.ItemFlow{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(x.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(x.Data.Base.Common.RoleId, 10),
				ItemFlow:          "in",
				ItemId:            strconv.FormatUint(item.UniqueId, 10),
				ItemCount:         int32(item.ItemNum),
				Level:             int32(x.Data.Base.Common.RoleLevel),
				VipLevel:          0,
				Action:            strconv.Itoa(int(reason)),
				ItemBefore:        int64(beforeNum),
				ItemAfter:         int64(afterNum),
				Recharge:          0,
			}
			taptap.WriteDataLog(taptap.LogType_ItemFlow, uid, x.Account.TapUserInfo, e)
		})

		changeItems = append(changeItems, &cmd.ItemReward{ItemId: item.BaseId, Num: item.ItemNum})
		if limitNum > 0 {
			limitItems[int32(item.BaseId)] = limitNum
		}
	}

	return changeItems, finalItems, limitItems, nil
}

// SubItems 扣除道具
func (x *UserData) SubItems(costItemUniqueId uint64, costNum uint32, uid string, reason common.ChangeReason) (*cmd.PCommonItemInfo, []*excel.ItemReward, error) {
	var (
		ret             = &cmd.PCommonItemInfo{}
		exchangeRewards = make([]*excel.ItemReward, 0)
	)

	var beforeNum uint32

	target, exist := x.GetUserItems().Items[costItemUniqueId]
	if exist && target != nil {
		beforeNum = target.ItemNum
		target.ItemNum -= costNum
		if target.ItemNum <= 0 {
			delete(x.GetUserItems().Items, costItemUniqueId)
		}

		// 下发客户端数据
		ret = target

		// 道具消耗转换的奖励
		itemCfg := excel.GetItemMgr().GetById(int32(target.BaseId))
		if itemCfg.Change.ItemId > 0 && itemCfg.Change.Num > 0 {
			exchangeRewards = append(exchangeRewards, &excel.ItemReward{
				ItemId: itemCfg.Change.ItemId,
				Num:    itemCfg.Change.Num * int32(costNum),
			})
		}

		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.ItemFlow{
		//		HeadInfo:   lilith.BuildHeadInfo(lilith.LogType_ItemFlow, uid, device),
		//		RoleId:     strconv.FormatUint(x.Data.Base.Common.RoleId, 10),
		//		ItemFlow:   "out",
		//		ItemId:     strconv.FormatUint(costItemUniqueId, 10),
		//		ItemCount:  int32(costNum),
		//		Level:      int32(x.Data.Base.Common.RoleLevel),
		//		VipLevel:   0,
		//		Action:     strconv.Itoa(int(reason)),
		//		ItemBefore: int64(beforeNum),
		//		ItemAfter:  int64(target.GetItemNum()),
		//		Recharge:   0,
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.ItemFlow{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(x.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(x.Data.Base.Common.RoleId, 10),
				ItemFlow:          "out",
				ItemId:            strconv.FormatUint(costItemUniqueId, 10),
				ItemCount:         int32(costNum),
				Level:             int32(x.Data.Base.Common.RoleLevel),
				VipLevel:          0,
				Action:            strconv.Itoa(int(reason)),
				ItemBefore:        int64(beforeNum),
				ItemAfter:         int64(target.GetItemNum()),
				Recharge:          0,
			}
			taptap.WriteDataLog(taptap.LogType_ItemFlow, uid, x.Account.TapUserInfo, e)
		})
		// 埋点log
		if reason == common.CR_Destroy_EXP_ITEM {
			//threading.RunSafe(func() {
			//	lilith.WriteDataLog(&lilith.DestroyExpireItem{
			//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_DestroyExpireItem, uid, x.Account.CliDeviceInfo),
			//		Id:             int64(costItemUniqueId),
			//		ItemId:         int32(target.BaseId),
			//		ItemNum:        int32(costNum),
			//		Expire:         int64(target.ExpirationTimestamp),
			//		Exchange:       lilith.ConvertMap2Str(map[int32]int32{itemCfg.Change.ItemId: itemCfg.Change.Num * int32(costNum)}),
			//	})
			//})
			threading.RunSafe(func() {
				e := &taptap.DestroyExpireItem{
					PropertyFieldInfo: taptap.BuildPropertyFieldInfo(x.Account.CliDeviceInfo),
					Id:                int64(costItemUniqueId),
					ItemId:            int32(target.BaseId),
					ItemNum:           int32(costNum),
					Expire:            target.ExpirationTimestamp,
					Exchange:          taptap.ConvertMap2Str(map[int32]int32{itemCfg.Change.ItemId: itemCfg.Change.Num * int32(costNum)}),
				}
				taptap.WriteDataLog(taptap.LogType_DestroyExpireItem, uid, x.Account.TapUserInfo, e)
			})
		}
	} else {
		return ret, exchangeRewards, errors.New("扣除失败")
	}

	logger.Debugf("扣除道具成功 %d %d", costItemUniqueId, costNum)
	return ret, exchangeRewards, nil
}

package useractor

import (
	"errors"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/meta"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/clidto"

	"gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
)

type ConsumeMgr struct {
	actor           *UserActor
	ExchangeRewards []*meta.ItemReward // 道具消耗转换的奖励(item表配置)
}

func newConsumeMgr(userActor *UserActor) *ConsumeMgr {
	return &ConsumeMgr{
		actor:           userActor,
		ExchangeRewards: make([]*meta.ItemReward, 0),
	}
}

func GetConsumeMgr(userActor *UserActor) *ConsumeMgr {
	return newConsumeMgr(userActor)
}

// 已经过期了
func hasExpiration(item *pb.PCommonItemInfo) bool {
	if item.ExpirationTimestamp == 0 {
		return false
	}
	nowSec := time.Now().Unix()
	return item.ExpirationTimestamp > 0 && item.ExpirationTimestamp <= nowSec
}

// --------------------------- 统一扣除资源处理逻辑 ---------------------------

// ConsumeAll 全部消耗
func (m *ConsumeMgr) ConsumeAll(itemUniqueIds []uint64, commonData *clidto.Comdata, reason common.ChangeReason) error {
	costItems := make(map[uint64]uint32)
	for _, eachCostUniqueId := range itemUniqueIds {
		target := m.actor.GetUserItems().Items[eachCostUniqueId]
		if target == nil {
			return fmt.Errorf("道具不足, uniqueId=%d", eachCostUniqueId)
		}

		costItems[eachCostUniqueId] = target.ItemNum
	}

	return m.ConsumeListByUniqueId(costItems, commonData, reason)
}

func (m *ConsumeMgr) ConsumeListByUniqueId(items map[uint64]uint32, commonData *clidto.Comdata, reason common.ChangeReason) error {

	for itemId, itemNum := range items {
		err := m.ConsumeOneByUniqueId(itemId, itemNum, commonData, reason)
		if err != nil {
			m.actor.Errorf("ConsumeMgr.ConsumeList get err, itemId=%d, itemNum=%d, err:%+v", itemId, itemNum, err)
			return err
		}
	}

	return nil
}

func (m *ConsumeMgr) ConsumeOneByUniqueId(itemUniqueId uint64, itemNum uint32, commonData *clidto.Comdata, reason common.ChangeReason) error {

	if itemUniqueId <= 0 {
		m.actor.Errorf("无效的item unique id, itemUniqueId=%d", itemUniqueId)
		return errors.New("扣除失败")
	}

	itemInfo := m.actor.GetUserItems().Items[itemUniqueId]
	if itemInfo == nil {
		return errors.New("扣除失败")
	}

	err := m.doConsumeByUniqueId(itemUniqueId, itemNum, commonData, reason)
	if err != nil {
		return err
	}

	return nil
}

func (m *ConsumeMgr) doConsumeByUniqueId(uniqueId uint64, costNum uint32, commonData *clidto.Comdata, reason common.ChangeReason) error {

	costItem, exchangeRewards, err := m.actor.SubItems(uniqueId, costNum, m.actor.uid, reason)
	if err == nil {
		commonData.Data.Items = append(commonData.Data.Items, costItem)
		err = m.actor.BagHandler.SaveDB()
		m.ExchangeRewards = append(m.ExchangeRewards, exchangeRewards...)
	}

	return err
}

// func (m *ConsumeMgr) ConsumeKeyValItemList(items []*pb.KeyValueItem, commonData *clidto.Comdata, reason common.ChangeReason) error {
// 	itemList := utils.ConvertItem(items)
// 	return m.ConsumeList(itemList, commonData, reason)
// }

func (m *ConsumeMgr) ConsumeKeyValList(items []*meta.KeyVal, commonData *clidto.Comdata, reason common.ChangeReason) error {
	// itemList := datahelper.ConvertItem3(items)
	return m.ConsumeList(nil, commonData, reason)
}

func (m *ConsumeMgr) ConsumeList(items map[int32]int32, commonData *clidto.Comdata, reason common.ChangeReason) error {
	for itemId, itemNum := range items {
		if itemNum < 0 {
			return errors.New(fmt.Sprintf("扣除数量为负数, itemId=%d, itemNum=%d", itemId, itemNum))
		}

		var itemCfg *meta.ItemPkgItemMeta
		// itemCfg := excel.GetItemMgr().GetById(itemId)
		if itemCfg == nil {
			return fmt.Errorf("item not found %d", itemId)
		}

		err := m.doConsume(itemCfg, uint32(itemNum), commonData, reason)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *ConsumeMgr) doConsume(itemCfg *meta.ItemPkgItemMeta, costNum uint32, commonData *clidto.Comdata, reason common.ChangeReason) error {

	var (
		err error
	)

	switch pb.ItemType(itemCfg.Type) {
	case pb.ItemType_Currency:
		err = m.actor.CurrencyHandler.SubCurrency(itemCfg.ItemId, int64(costNum), commonData, reason)

	case pb.ItemType_Consumable,
		pb.ItemType_Material,
		pb.ItemType_Food,
		pb.ItemType_Gift,
		pb.ItemType_Quest:
		costItem, exchangeRewards, errx := m.actor.SubItems(uint64(itemCfg.ItemId), costNum, m.actor.uid, reason)
		err = errx
		if err == nil {
			commonData.Data.Items = append(commonData.Data.Items, costItem)
			err = m.actor.BagHandler.SaveDB()
			m.ExchangeRewards = append(m.ExchangeRewards, exchangeRewards...)
		}
	case pb.ItemType_Stamina:
		// err = m.actor.PlayerLevelHandler.SubStamina(int32(costNum), commonData, reason)

	default:
		m.actor.Warnf("doConsume ===>>> 未支持的itemType=%d\n", itemCfg.Type)
	}

	return err
}

// --------------------------- 统一check消耗资源的逻辑 ---------------------------

// CheckMapEnoughByUniqueId
//
//	@Description: 批量检查过期类型道具数量是否足够
//	@receiver m
//	@param items 扣除资源列表
//	@return bool 足够返回true，否则返回false
func (m *ConsumeMgr) CheckMapEnoughByUniqueId(items map[uint64]uint32) bool {
	for k, v := range items {
		if !m.CheckEnoughByUniqueId(k, v) {
			return false
		}
	}

	return true
}

// CheckEnoughByUniqueId 检查指定道具的数量是否足够,足够则返回true
func (m *ConsumeMgr) CheckEnoughByUniqueId(uniqueId uint64, itemNum uint32) bool {
	itemInfo := m.actor.GetUserItems().Items[uniqueId]
	if itemInfo != nil && !hasExpiration(itemInfo) && itemInfo.ItemNum >= itemNum {
		return true
	}

	return false
}

// CheckMapEnough
//
//	@Description: 批量检查扣除资源数量是否足够
//	@receiver m
//	@param items 扣除资源列表
//	@return bool 足够返回true，否则返回false
func (m *ConsumeMgr) CheckMapEnough(items map[int32]int32) bool {
	for k, v := range items {
		if !m.CheckEnough(k, v) {
			return false
		}
	}

	return true
}

// CheckKeyValEnough
//
//	@Description: 批量检查扣除资源数量是否足够
//	@receiver m
//	@param items 扣除资源列表
//	@return bool 足够返回true，否则返回false
func (m *ConsumeMgr) CheckKeyValEnough(items []*meta.KeyVal) bool {
	for _, kv := range items {
		if !m.CheckEnough(kv.Key, kv.Val) {
			return false
		}
	}

	return true
}

func (m *ConsumeMgr) CheckKeyValItemEnough(items []*pb.KeyValueItem) bool {
	for _, kv := range items {
		if !m.CheckEnough(kv.Key, kv.Value) {
			return false
		}
	}

	return true
}

// CheckEnough 检查指定道具的数量是否足够,足够则返回true
func (m *ConsumeMgr) CheckEnough(costId, costNum int32) bool {
	if costNum == 0 {
		return true
	}

	if costNum < 0 {
		m.actor.Warnf("CheckEnough ===>>> costId=%d, costNum=%d 为负数", costId, costNum)
		return false
	}

	var cfg *meta.ItemPkgItemMeta
	// cfg := excel.GetItemMgr().GetById(costId)
	if cfg == nil {
		m.actor.Warnf("CheckEnough ===>>> 不存在的道具costId=%d, costNum=%d, ", costId, costNum)
		return false
	}

	switch cfg.Type {
	case int32(pb.ItemType_Currency):
		return m.actor.CurrencyHandler.CheckEnough(cfg.ItemId, int64(costNum))

	case int32(pb.ItemType_Consumable),
		int32(pb.ItemType_Material),
		int32(pb.ItemType_Food),
		int32(pb.ItemType_Gift),
		int32(pb.ItemType_Quest):
		itemInfo := m.actor.GetUserItems().Items[uint64(costId)]
		if itemInfo != nil && !hasExpiration(itemInfo) && int32(itemInfo.ItemNum) >= costNum {
			return true
		}

	case int32(pb.ItemType_Stamina):
		// return m.actor.PlayerLevelHandler.CheckStaminaEnough(costNum)
		return false
	default:
		m.actor.Warnf("CheckEnough ===>>> 未支持的itemType=%d\n", cfg.Type)
	}

	return false
}

// CheckItemType
//
//	@Description: 检查道具是否为指定类型
//	@receiver m
//	@param itemId
//	@param fType 父类型
//	@param sType 子类型 填0表示不校验
//	@return bool
func (m *ConsumeMgr) CheckItemType(itemId int32, fType, sType int32) bool {
	// itemCfg := excel.GetItemMgr().GetById(itemId)
	// if itemCfg == nil {
	// 	return false
	// }
	// if itemCfg.Type != fType {
	// 	return false
	// }
	// if sType > 0 && itemCfg.SubType != sType {
	// 	return false
	// }
	return true
}

// CheckItemNumAndType
//
//	@Description: 检查给定道具是否指定的道具类型，并且数量是否足够
//	@receiver m
//	@param itemId 道具id
//	@param itemNum 数量
//	@param fType 父类型
//	@param sType 子类型 填0表示不校验
//	@return bool 条件均满足返回true，否则返回false
func (m *ConsumeMgr) CheckItemNumAndType(itemId, itemNum, fType, sType int32) bool {
	return m.CheckItemType(itemId, fType, sType) && m.CheckEnough(itemId, itemNum)
}

// CheckMapItemNumAndType
//
//	@Description: 检查给定道具集是否为指定的道具类型，并且数量是否足够
//	@receiver m
//	@param items 道具集合
//	@param fType 父类型
//	@param sType 子类型 填0表示不校验
//	@return bool 条件均满足返回true，否则返回false
func (m *ConsumeMgr) CheckMapItemNumAndType(items map[int32]int32, fType, sType int32) bool {
	for k, v := range items {
		if !m.CheckItemNumAndType(k, v, fType, sType) {
			return false
		}
	}
	return true
}

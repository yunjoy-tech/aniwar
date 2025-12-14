package useractor

import (
	"fmt"
	"time"

	"gitee.com/aniwar2/aniwar/src/actorserver/useractor/event"

	"gitee.com/aniwar2/aniwar/src/common/clidto"

	"gitee.com/aniwar2/aniwar/src/common"
	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"
	excel "gitee.com/aniwar2/aniwar/src/excel/data"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/utils"
)

type DropMgr struct {
	actor      *UserActor
	limitItems map[int32]int32 // 超出上限记录
}

func newDropMgr(actor *UserActor) *DropMgr {
	return &DropMgr{actor: actor, limitItems: make(map[int32]int32)}
}

func GetDropMgr(actor *UserActor) *DropMgr {
	return newDropMgr(actor)
}

// @param overlying 是否合并奖励
func mergeDropChange(source, newChange *pb.DropChange, overlying ...bool) {
	if source == nil || newChange == nil {
		// m.actor.Warnf("mergeDropChange err, source:%v, newChange:%v", source, newChange)
		return
	}

	// 玩家经验
	if newChange.RoleExp != nil {
		if source.RoleExp == nil {
			source.RoleExp = &pb.PRoleBattleSettlement{}
		}

		source.RoleExp.RoleLevel = newChange.RoleExp.RoleLevel
		source.RoleExp.RoleExp += newChange.RoleExp.RoleExp
	}

	if len(newChange.Items) > 0 {
		source.Items = append(source.Items, newChange.Items...)
		if myUtils.GotBoolParam(overlying...) {
			source.Items = mergeItems(source.Items)
		}
	}
	// TODO 根据id merge 数量
	if len(newChange.CardExpInfos) > 0 {
		source.CardExpInfos = append(source.CardExpInfos, newChange.CardExpInfos...)
	}
}

func mergeItems(items []*pb.ItemReward) []*pb.ItemReward {
	tempMap := make(map[uint32]uint32, len(items))
	for _, v := range items {
		// TODO 根据id merge 数量
		tempMap[v.ItemId] += v.Num
	}

	ret := make([]*pb.ItemReward, 0, len(tempMap))
	for k, v := range tempMap {
		ret = append(ret, &pb.ItemReward{ItemId: k, Num: v})
	}

	return ret
}

// --------------------------- 统一掉落资源的处理逻辑 ---------------------------

func (m *DropMgr) DropListByItems(dropItemRewards []*excel.ItemReward, isFullMail bool, params []int32,
	commonData *clidto.Comdata, reason common.ChangeReason) (*pb.DropChange, error) {

	items := make(map[uint32]uint32, 0)
	for _, eachItemReward := range dropItemRewards {
		if _, ok := items[uint32(eachItemReward.ItemId)]; ok {
			items[uint32(eachItemReward.ItemId)] += uint32(eachItemReward.Num)
		} else {
			items[uint32(eachItemReward.ItemId)] = uint32(eachItemReward.Num)
		}
	}

	if len(items) <= 0 {
		return &pb.DropChange{}, nil
	}

	return m.DropList(items, isFullMail, params, commonData, reason)
}

func (m *DropMgr) DropList2(items map[int32]int32, isFullMail bool, params []int32, commonData *clidto.Comdata, reason common.ChangeReason) (*pb.DropChange, error) {

	items2 := make(map[uint32]uint32, 0)
	for key, val := range items {
		items2[uint32(key)] = uint32(val)
	}

	return m.DropList(items2, isFullMail, params, commonData, reason)
}

func (m *DropMgr) DropListByPCommonItemInfo(items map[int32]*pb.PCommonItemInfo, isFullMail bool, params []int32, commonData *clidto.Comdata, reason common.ChangeReason) (*pb.DropChange, error) {

	items2 := make(map[uint32]uint32, 0)
	for _, val := range items {
		items2[val.BaseId] = val.ItemNum
	}

	return m.DropList(items2, isFullMail, params, commonData, reason)
}

func (m *DropMgr) DropList(items map[uint32]uint32, isFullMail bool, params []int32, commonData *clidto.Comdata, reason common.ChangeReason) (*pb.DropChange, error) {
	var (
		dropChange = &pb.DropChange{}
	)

	if len(items) <= 0 {
		return dropChange, nil
	}

	m.actor.Infof("掉落奖励道具 reason: %v, 内容：%v", reason, items)
	for itemId, itemNum := range items {
		itemCfg := excel.GetItemMgr().GetById(int32(itemId))
		if itemCfg == nil {
			m.actor.Warnf("item not found %d", itemId)
			return nil, fmt.Errorf("item config not found %d", itemId)
		}

		// eachChange, err := m.DropOne(itemId, itemNum, false, params, commonData, reason)
		eachChange, err := m.doDrop(itemCfg, itemNum, params, commonData, reason)
		if err != nil {
			m.actor.Warnf("_dropMgr.DropList get err, itemId=%d, itemNum=%d, err:%+v", itemId, itemNum, err)
			return nil, err
		}

		// dropChange.RoleExp
		// dropChange.Items = append(dropChange.Items, eachChange.Items...)
		// dropChange.CardExpInfos = append(dropChange.CardExpInfos, eachChange.CardExpInfos...)

		// 合并奖励
		mergeDropChange(dropChange, eachChange)
	}
	// 发送上限邮件
	if isFullMail && len(m.limitItems) > 0 {
		m.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_2, m.limitItems, commonData)
	}
	// 发布道具掉落事件
	errx := m.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_ITEM_DROP, []int32{}, nil))
	if errx != nil {
		m.actor.Error(errx)
	}

	return dropChange, nil
}

// 奖励物品掉落
// @param DropChange 给前端的改变量-界面显示
// @param commonData 给前端最终量-修改前端的缓存数据
func (m *DropMgr) doDrop(itemCfg *excel.ItemCfg, itemNum uint32, params []int32, commonData *clidto.Comdata, reason common.ChangeReason) (*pb.DropChange, error) {

	var (
		err        error
		dropChange = &pb.DropChange{}
	)

	switch pb.ItemType(itemCfg.Type) {
	case pb.ItemType_Ability:
		_, err = m.handleItemTypeAbility(itemCfg, itemNum, params, commonData, dropChange)
		if err != nil {
			m.actor.Warnf("doDrop ===>>> handleItemTypeAbility=%v", err)
		}
		// if err == nil {
		//	dropChange.CardExpInfos = append(dropChange.CardExpInfos, expRewards...)
		//	//dropChange.Items = append(dropChange.Items, &pb.ItemReward{ItemId: uint32(itemCfg.ItemId), Num: itemNum})
		// }

	case pb.ItemType_Currency:
		err = m.actor.CurrencyHandler.AddCurrency(itemCfg.ItemId, int64(itemNum), commonData, reason)
		dropChange.Items = append(dropChange.Items, &pb.ItemReward{ItemId: uint32(itemCfg.ItemId), Num: itemNum})

	case pb.ItemType_Consumable,
		pb.ItemType_Material,
		pb.ItemType_Food,
		pb.ItemType_Gift,
		pb.ItemType_Quest:
		eachChangeItems, eachFinalItems, limitItems, errx := m.handleItemTypeBag(itemCfg, itemNum, reason)
		err = errx
		if errx == nil {
			dropChange.Items = append(dropChange.Items, eachChangeItems...)
			commonData.Data.Items = append(commonData.Data.Items, eachFinalItems...)
			myUtils.MergeItems(m.limitItems, limitItems)
		}

	case pb.ItemType_Card:
		// drop, errx := m.actor.CardHandler.AddCard(itemCfg, itemNum, commonData)
		// err = errx
		// if drop != nil {
		// 	dropChange.Items = append(dropChange.Items, drop.Items...)
		// }

	case pb.ItemType_CardSkin:
		// drop, errx := m.actor.SkinHandler.AddCardSkin(itemCfg, itemNum, commonData)
		// err = errx
		// if drop != nil {
		// 	dropChange.Items = append(dropChange.Items, drop.Items...)
		// }

	case pb.ItemType_Equip:
		// err = m.actor.EquipHandler.CreateAndAddEquip(itemCfg.ItemId, int32(itemNum), commonData)
		// if err == nil {
		// 	dropChange.Items = append(dropChange.Items, &pb.ItemReward{ItemId: uint32(itemCfg.ItemId), Num: itemNum})
		// }
	case pb.ItemType_Stamina: // 更新玩家体力
		// err = m.actor.PlayerLevelHandler.AddStamina(int32(itemNum), commonData, reason)
		// if err == nil {
		// 	dropChange.Items = append(dropChange.Items, &pb.ItemReward{ItemId: uint32(itemCfg.ItemId), Num: itemNum})
		// }

	default:
		m.actor.Warnf("doDrop ===>>> 未支持的itemType=%d \n", itemCfg.Type)
	}

	m.actor.Debugf("获得奖励: %+v, reason:%d", commonData.Data.Items, reason)
	return dropChange, err
}

// 掉落道具子类型1处理
func (m *DropMgr) handleItemTypeAbility(itemCfg *excel.ItemCfg, itemNum uint32, params []int32,
	commonData *clidto.Comdata, dropChange *pb.DropChange) ([]*pb.CommonCardExpReward, error) {

	switch itemCfg.SubType {
	case int32(pb.ItemSubType_1_Ability_RoleExp):
		dropChange.RoleExp = &pb.PRoleBattleSettlement{
			RoleLevel: m.actor.GetUserData().Common.RoleLevel, // 变化前的等级
			RoleExp:   itemNum,                                // 这次增加的经验值
		}

		_, err := m.actor.LoginHandler.AddRoleExp(uint64(itemNum), commonData)
		return nil, err

	case int32(pb.ItemSubType_1_Ability_CardExp):
		// expRewards, cards := m.actor.CardHandler.AddCardExpByIdList(params, int32(itemNum))
		// dropChange.CardExpInfos = append(dropChange.CardExpInfos, expRewards...)
		// for _, v := range cards {
		// 	commonData.Data.Card = append(commonData.Data.Card, m.actor.CardHandler.ToClientData(v))
		// }
		// return expRewards, nil

	case int32(pb.ItemSubType_1_Ability_CardFavoriteExp):
		// cardId := m.actor.DutyHandler.GetCurDutyCard()
		// card, errCode := m.actor.CardHandler.AddFavoriteExpById(cardId, itemNum)
		// if card == nil || errCode != pb.ErrorCode_Success {
		// 	break
		// }
		// commonData.Data.Card = append(commonData.Data.Card, card)

	default:
		m.actor.Debugf("unrealized item type ability %d", itemCfg.SubType)
	}

	return nil, nil
}

// 直接掉落背包的处理
func (m *DropMgr) handleItemTypeBag(itemCfg *excel.ItemCfg, itemNum uint32, reason common.ChangeReason) ([]*pb.ItemReward, []*pb.PCommonItemInfo, map[int32]int32, error) {

	itemInfo := m.CreatePCommonItemInfo(uint32(itemCfg.Id), itemNum)
	changeItems, finalItems, limitItems, err := m.actor.UserData.AddItems(m.actor.uid, reason, itemInfo)
	if err != nil {
		return nil, nil, nil, err
	}

	m.actor.Debugf("doDrop ===>>> ItemData:%+v", m.actor.Data.GetItemData())

	err = m.actor.BagHandler.SaveDB()
	if err != nil {
		return nil, nil, nil, err
	}

	return changeItems, finalItems, limitItems, err
}

func (m *DropMgr) CreatePCommonItemInfo(itemId, itemNum uint32) *pb.PCommonItemInfo {
	// 到期时间
	expireSec := int64(0)
	itemCfg := excel.GetItemMgr().GetById(int32(itemId))
	if itemCfg.GetTimeLimit() > 0 {
		expireSec = time.Now().Unix() + int64(itemCfg.GetTimeLimit())
	}

	// 主键id
	uniqueId := uint64(0)
	if itemCfg.GetTimeLimit() > 0 {
		uniqueId = uint64(utils.GenIntUUID()) // 有过期时间，就重新生成一个主键id
	} else {
		uniqueId = uint64(itemId)
	}

	m.actor.Debugf("道具%d, 数量%d, 主键%d, 到期时间%d", itemId, itemNum, uniqueId, expireSec)

	return &pb.PCommonItemInfo{
		UniqueId:            uniqueId,
		BaseId:              itemId,
		ItemNum:             itemNum,
		ExpirationTimestamp: expireSec,
	}
}

// CheckMapLimit 道具是否到达背包的数量上限
func (m *DropMgr) CheckMapLimit(items map[int32]int32) bool {
	for k, v := range items {
		if m.CheckLimit(k, v) {
			return true
		}
	}
	return false
}

// CheckLimit 指定道具是否超过上限
func (m *DropMgr) CheckLimit(itemId, itemNum int32) bool {
	itemCfg := excel.GetItemMgr().GetById(itemId)
	if itemCfg == nil {
		return true
	}

	// 按类型检查
	switch pb.ItemType(itemCfg.Type) {
	case pb.ItemType_Ability:
		return false

	case pb.ItemType_Currency:
		return m.actor.CurrencyHandler.CheckLimit(itemCfg.ItemId, int64(itemNum))

	case pb.ItemType_Consumable,
		pb.ItemType_Material,
		pb.ItemType_Food,
		pb.ItemType_Gift,
		pb.ItemType_Quest:
		cur := int32(0)
		for _, item := range m.actor.GetUserItems().Items {
			if item.BaseId == uint32(itemId) {
				cur += int32(item.ItemNum)
			}
		}
		if int64(cur+itemNum) > itemCfg.GetNumLimit() {
			return true
		}
		return false

	case pb.ItemType_Card:
		return false

	case pb.ItemType_CardSkin:
		return false

	case pb.ItemType_Equip:
		// return m.actor.EquipHandler.CheckLimit(itemNum)

	case pb.ItemType_Stamina:
		// return m.actor.PlayerLevelHandler.CheckLimit(itemNum)

	default:
		m.actor.Warnf("CheckLimit ===>>> 未支持的itemType=%d \n", itemCfg.Type)
	}

	return true
}

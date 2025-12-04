package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

// @author yitie
// @module 装备系统
type EquipHandler struct {
	*UABaseHandler
}

func NewEquipHandler(actor *UserActor) *EquipHandler {
	h := &EquipHandler{UABaseHandler: NewUABaseHandler(actor, "EquipHandler")}
	h.ChildHandler = h

	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_EquipLevelUpReq), h.EquipLevelUpReq) // 装备升级
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_EquipWearReq), h.EquipWearReq)       // 装备穿戴
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_EquipUnWearReq), h.EquipUnWearReq)   // 装备卸下

	return h
}

// Init 初始化模块数据
func (h *EquipHandler) Init() error {
	// 初始化
	h.actor.Data.EquipData = &cmd.PEquipData{
		Createtime: time.Now().Unix(),
		Equips:     make(map[uint64]*cmd.PCommonEquipInfo),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init equip data success.")
	return nil
}

func (h *EquipHandler) EnterGame() error {
	return nil
}

func (h *EquipHandler) DailyRefresh() error {
	return nil
}

func (h *EquipHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PEquipData); ok {
		h.actor.Data.EquipData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *EquipHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserEquipInfo(h.actor.ID()), h.actor.Data.EquipData
}

// 装备升级
func (h *EquipHandler) EquipLevelUpReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_EquipLevelUpReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 获取装备
	equip, err := h.GetEquip(req.EquipId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_EquipNotExist)
	}

	// 是否满级
	if equip.Level >= getEquipLevelLimit(equip.ConfigId) {
		return nil, fmt.Errorf("equip is max level"), int32(cmd.ErrorCode_EquipIsMaxLevel)
	}
	// 消耗check
	code := h.equipCostCheck(req.Costs)
	if code != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("equip cost check failed %v", code), int32(code)
	}
	// 计算电力消耗
	addExp := h.calEquipCostExp(req.Costs)
	currencyCost := calCurrencyCost(addExp)
	if !GetConsumeMgr(h.actor).CheckEnough(common.CURRENCY_ITEM_ID_2001, currencyCost) {
		return nil, fmt.Errorf("currency not enough %d", currencyCost), int32(cmd.ErrorCode_CurrencyNotEnough)
	}

	// 处理逻辑
	err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{common.CURRENCY_ITEM_ID_2001: currencyCost}, h.actor.comData, common.CR_EQUIP_LEVEL_UP)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	err = h.DelEquip(req.Costs)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 升级前等级
	beforeLevel := equip.Level
	h.addEquipExp(equip, addExp)
	// 穿戴中，尝试刷新血量加成
	if equip.CardId > 0 {
		card, _ := h.actor.CardHandler.GetCard(uint32(equip.CardId))
		if card != nil {
			h.actor.CardHandler.TrySupplementMaxHp(card)
		}
	}
	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 升级后等级
	afterLevel := equip.Level
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_EQUIP_LEVELUP, []int32{TASK_TYPE_211}, nil))
	if errx != nil {
		h.Error(errx)
	}

	// 消息返回
	h.actor.comData.Data.Equip = append(h.actor.comData.Data.Equip, equip)
	rsp := &cmd.LS2C_EquipLevelUpRes{
		CommonData: h.actor.comData.FixDownComData(),
		Costs:      req.Costs,
	}

	// 装备升级埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.EquipLevelUp{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogEquipLevelUp, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		EquipId:        req.EquipId,                       // 升级的装备唯一id
	//		ConfigId:       equip.ConfigId,                    // 装备配表id
	//		EquipDelIds:    lilith.ConvertList2Str(req.Costs), // 待合并的装备id列表
	//		AddExp:         addExp,                            // 增加的经验值
	//		BeforeLv:       beforeLevel,                       // 升级前等级
	//		AfterLv:        afterLevel,                        // 升级后等级
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.EquipLevelUp{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			EquipId:           req.EquipId,                       // 升级的装备唯一id
			ConfigId:          equip.ConfigId,                    // 装备配表id
			EquipDelIds:       taptap.ConvertList2Str(req.Costs), // 待合并的装备id列表
			AddExp:            addExp,                            // 增加的经验值
			BeforeLv:          beforeLevel,                       // 升级前等级
			AfterLv:           afterLevel,                        // 升级后等级
		}
		taptap.WriteDataLog(taptap.LogEquipLevelUp, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, 0
}

// 装备穿戴
func (h *EquipHandler) EquipWearReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_EquipWearReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 获取装备
	equip, err := h.GetEquip(req.EquipId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_EquipNotExist)
	}

	// 卡牌check
	card, err := h.actor.CardHandler.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_CardNotExist)
	}

	// 类型匹配
	cfg := excel.GetEquipmentMgr().GetById(equip.ConfigId)
	if cfg == nil {
		return nil, fmt.Errorf("cfg not found %d", equip.ConfigId), int32(cmd.ErrorCode_NotFoundConfig)
	}
	beastarCfg := excel.GetBeastarMgr().GetById(int32(req.CardId))
	if beastarCfg == nil {
		return nil, fmt.Errorf("cfg not found %d", req.CardId), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// 处理逻辑
	equipInfos, cardInfos, code := h.handleEquipWear(card, equip)
	if code != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("handleEquipWear failed %d", int32(code)), int32(code)
	}

	err = h.actor.CardHandler.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 尝试刷新血量加成
	h.actor.CardHandler.TrySupplementMaxHp(card)

	// 消息返回
	cards := make([]*cmd.PClientCardInfo, 0)
	for _, t := range cardInfos {
		cards = append(cards, h.actor.CardHandler.ToClientData(t))
	}

	// 装备穿戴
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.EquipWear{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogEquipWear, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		EquipId:        req.EquipId,    // 装备唯一id
	//		ConfigId:       equip.ConfigId, // 装备配表id
	//		CardId:         req.CardId,     // 穿戴的目标卡牌id
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.EquipWear{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			EquipId:           req.EquipId,    // 装备唯一id
			ConfigId:          equip.ConfigId, // 装备配表id
			CardId:            req.CardId,     // 穿戴的目标卡牌id
		}
		taptap.WriteDataLog(taptap.LogEquipWear, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	rsp := &cmd.LS2C_EquipWearRes{
		CommonData: &cmd.CliComData{Equip: equipInfos, Card: cards},
	}

	return rsp, nil, 0
}

// 穿戴或者交换逻辑
func (h *EquipHandler) handleEquipWear(card *cmd.CardData, equip *cmd.PCommonEquipInfo) ([]*cmd.PCommonEquipInfo, []*cmd.CardData, cmd.ErrorCode) {
	equipInfos := make([]*cmd.PCommonEquipInfo, 0)
	cardInfos := make([]*cmd.CardData, 0)
	var (
		errx error
		err  cmd.ErrorCode
	)
	// 装备孔位
	cfg := excel.GetEquipmentMgr().GetById(equip.ConfigId)
	if cfg == nil {
		return nil, nil, cmd.ErrorCode_NotFoundConfig
	}
	// 是否解锁孔位
	limit := excel.GetConfigMgr().GetCfg().EQUIP_UNLOCK_LIMIT[cfg.Type-1]
	if int32(card.BreakthroughLevel) < limit {
		return nil, nil, cmd.ErrorCode_CharacterNotBreak
	}

	// 装备穿戴中
	oldCardId := equip.CardId
	var oldCard *cmd.CardData
	if oldCardId > 0 {
		oldCard, errx = h.actor.CardHandler.GetCard(uint32(oldCardId))
		if errx != nil {
			return nil, nil, cmd.ErrorCode_CardNotExist
		}
		_, err = h.handleEquipUnWear(oldCard, uint32(cfg.Type))
		if err != cmd.ErrorCode_Success {
			return nil, nil, err
		}
		cardInfos = append(cardInfos, oldCard)
	}

	// 位置上有老装备
	oldEquipId := card.EquipId[uint32(cfg.Type)]
	oldEquip := &cmd.PCommonEquipInfo{}
	if oldEquipId > 0 {
		oldEquip, err = h.handleEquipUnWear(card, uint32(cfg.Type))
		if err != cmd.ErrorCode_Success {
			return nil, nil, err
		}
		equipInfos = append(equipInfos, oldEquip)
	}

	// 交换逻辑
	if oldCardId > 0 && oldEquipId > 0 {
		oldEquip.CardId = int32(oldCard.BaseId)
		oldCard.EquipId[uint32(cfg.Type)] = uint64(oldEquip.EquipId)
	}

	// 穿戴逻辑
	equip.CardId = int32(card.BaseId)
	card.EquipId[uint32(cfg.Type)] = uint64(equip.EquipId)
	equipInfos = append(equipInfos, equip)
	cardInfos = append(cardInfos, card)

	return equipInfos, cardInfos, cmd.ErrorCode_Success
}

func (h *EquipHandler) handleEquipUnWear(card *cmd.CardData, typ uint32) (*cmd.PCommonEquipInfo, cmd.ErrorCode) {
	// 修正已穿戴装备的数据
	equip, err := h.GetEquip(card.EquipId[typ])
	if err != nil {
		return nil, cmd.ErrorCode_EquipNotExist
	}

	equip.CardId = 0

	// 修正卡牌数据
	card.EquipId[typ] = 0

	return equip, cmd.ErrorCode_Success
}

// 装备卸下
func (h *EquipHandler) EquipUnWearReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.C2LS_EquipUnWearReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 获取卡牌
	card, err := h.actor.CardHandler.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_CardNotExist)
	}

	// 是否穿戴了
	if card.EquipId[req.Typ] == 0 {
		return nil, err, int32(cmd.ErrorCode_EquipIsNotWear)
	}

	// 处理逻辑
	equip, code := h.handleEquipUnWear(card, req.Typ)
	if code != cmd.ErrorCode_Success {
		return nil, fmt.Errorf("handleEquipUnWear failed %d", int32(code)), int32(code)
	}

	err = h.actor.CardHandler.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 尝试刷新血量加成
	h.actor.CardHandler.TrySupplementMaxHp(card)

	// 装备卸下
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.EquipUnWear{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogEquipunWear, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		EquipId:        equip.EquipId,  // 装备唯一id
	//		ConfigId:       equip.ConfigId, // 装备配表id
	//		CardId:         req.CardId,     // 卸下的目标卡牌id
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.EquipUnWear{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			EquipId:           equip.EquipId,  // 装备唯一id
			ConfigId:          equip.ConfigId, // 装备配表id
			CardId:            req.CardId,     // 卸下的目标卡牌id
		}
		taptap.WriteDataLog(taptap.LogEquipunWear, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 消息返回
	rsp := &cmd.LS2C_EquipUnWearRes{
		CommonData: &cmd.CliComData{Equip: []*cmd.PCommonEquipInfo{equip}, Card: []*cmd.PClientCardInfo{h.actor.CardHandler.ToClientData(card)}},
	}

	return rsp, nil, 0
}

func (h *EquipHandler) buildEquipList() []*cmd.PCommonEquipInfo {
	equipData := make([]*cmd.PCommonEquipInfo, 0)
	for _, info := range h.actor.GetEquipData().Equips {
		equipData = append(equipData, &cmd.PCommonEquipInfo{
			EquipId:    info.EquipId,
			ConfigId:   info.ConfigId,
			Level:      info.Level,
			Exp:        info.Exp,
			SkillNumId: info.SkillNumId,
			CardId:     info.CardId,
			MainAttr:   info.MainAttr,
			SubAttr:    info.SubAttr,
		})
	}

	return equipData
}

// CreateAndAddEquip 增加指定装备
func (h *EquipHandler) CreateAndAddEquip(baseId, num int32, commonData *clidto.Comdata) error {
	equips, err := h.CreateEquip(baseId, num)
	if err != nil {
		return err
	}

	err = h.AddEquip(equips, commonData)
	if err != nil {
		return err
	}

	h.Debug("CreateAndAddEquip success")
	return nil
}

// AddEquip 增加指定装备
func (h *EquipHandler) AddEquip(adds []*cmd.PCommonEquipInfo, commonData *clidto.Comdata) error {
	limit := make(map[int32]int32)
	ret := make([]*cmd.PCommonEquipInfo, 0)
	quality := make(map[int32]int32)
	for _, equip := range adds {
		// 上限判定
		if len(h.actor.GetEquipData().Equips) >= int(excel.GetConfigMgr().GetCfg().EQUIPMENT_PACKAGE_LIMIT) {
			limit[equip.ConfigId] += 1
			continue
		}

		cfg := excel.GetItemMgr().GetById(equip.ConfigId)
		if cfg == nil {
			continue
		}

		if h.actor.GetEquipData().Equips == nil {
			h.actor.GetEquipData().Equips = make(map[uint64]*cmd.PCommonEquipInfo)
		}

		h.actor.GetEquipData().Equips[uint64(equip.EquipId)] = equip
		ret = append(ret, equip)
		quality[cfg.Quality] += 1

		// 装备获得埋点
		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.EquipCreate{
		//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogEquipCreate, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		EquipId:        equip.EquipId,    // 装备唯一id
		//		ConfigId:       equip.ConfigId,   // 装备配表id
		//		Lv:             equip.Level,      // 当前等级
		//		Exp:            equip.Exp,        // 当前经验
		//		SkillId:        equip.SkillNumId, // 技能id
		//		MainAttr:       equip.MainAttr,   // 主属性类型
		//		SubAttr:        equip.SubAttr,    // 副属性类型
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.EquipCreate{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				EquipId:           equip.EquipId,    // 装备唯一id
				ConfigId:          equip.ConfigId,   // 装备配表id
				Lv:                equip.Level,      // 当前等级
				Exp:               equip.Exp,        // 当前经验
				SkillId:           equip.SkillNumId, // 技能id
				MainAttr:          equip.MainAttr,   // 主属性类型
				SubAttr:           equip.SubAttr,    // 副属性类型
			}
			taptap.WriteDataLog(taptap.LogEquipCreate, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	commonData.Data.Equip = append(commonData.Data.Equip, ret...) // 批量添加道具
	// 是否需要发邮件
	if len(limit) > 0 {
		if err := h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_2, limit, commonData); err != nil {
			return err
		}
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	// 发布事件
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_EQUIP_CREATE, []int32{TASK_TYPE_201, TASK_TYPE_202}, map[string]interface{}{
		"count":   int32(len(ret)),
		"quality": quality,
	}))
	if errx != nil {
		h.Error(errx)
	}

	h.Debug("add equip success")
	return nil
}

func (h *EquipHandler) CreateEquip(baseId, num int32) ([]*cmd.PCommonEquipInfo, error) {
	if num <= 0 {
		return nil, fmt.Errorf("create equip param err %d", num)
	}

	adds := make([]*cmd.PCommonEquipInfo, 0)
	for i := int32(0); i < num; i++ {
		cfg := excel.GetEquipmentMgr().GetById(baseId)
		if cfg == nil {
			return nil, fmt.Errorf("equip cfg not found %d", baseId)
		}

		skillNumId := utils.RandomList(cfg.AbilityId)

		atrlinkCfg := excel.GetEquipmentAtrlinkMgr().GetById(cfg.AtrType*100 + cfg.Type)
		if atrlinkCfg == nil {
			return nil, fmt.Errorf("atrlinkCfg not found %d", cfg.AtrType*100+cfg.Type)
		}
		mainAttr := utils.RandomList(atrlinkCfg.MainAtr)
		subAttrs := checkAttrConflict(mainAttr, atrlinkCfg.SubAtr)
		subAttr := utils.RandomList(subAttrs)

		newEquip := &cmd.PCommonEquipInfo{
			EquipId:    int64(h.actor.Srv.GenGUID(guid.GUID_EQUIP)),
			ConfigId:   cfg.Id,
			Level:      0,
			Exp:        0,
			SkillNumId: skillNumId,
			CardId:     0,
			MainAttr:   mainAttr,
			SubAttr:    subAttr,
		}
		adds = append(adds, newEquip)
	}

	h.Debugf("create equip success. baseId:%d num:%d", baseId, num)
	return adds, nil
}

// 主属性和副属性互斥
func checkAttrConflict(mainAttr int32, subAttrs []int32) []int32 {
	var f bool
	switch mainAttr {
	case common.EQUIP_ATTR_8, common.EQUIP_ATTR_9, common.EQUIP_ATTR_10, common.EQUIP_ATTR_11:
		f = true
	}

	// 副属性配置id和属性类型的映射
	tempMap := map[int32]int32{
		3:  common.EQUIP_ATTR_8,
		4:  common.EQUIP_ATTR_9,
		9:  common.EQUIP_ATTR_10,
		10: common.EQUIP_ATTR_11,
	}

	newAttrs := make([]int32, 0)
	for _, attr := range subAttrs {
		if f && tempMap[attr] == mainAttr {
			continue
		}
		newAttrs = append(newAttrs, attr)
	}

	return newAttrs
}

func (h *EquipHandler) CheckLimit(adds int32) bool {
	if int32(len(h.actor.GetEquipData().Equips))+adds > (excel.GetConfigMgr().GetCfg().EQUIPMENT_PACKAGE_LIMIT) {
		return true
	}

	return false
}

// DelEquip 删除指定装备
func (h *EquipHandler) DelEquip(delIds []uint64) error {

	hadDel := make([]uint64, 0)
	for _, delId := range delIds {
		if h.IsExistEquip(delId) {
			// 删除指定设备埋点
			// 获取装备
			equip, err := h.GetEquip(delId)
			if err != nil {
				continue
			}
			// 装备删除埋点
			//threading.RunSafe(func() {
			//	lilith.WriteDataLog(&lilith.EquipDelete{
			//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogEquipDelete, h.actor.uid, h.actor.Account.CliDeviceInfo),
			//		EquipId:        delId,            // 删除的装备唯一id
			//		ConfigId:       equip.ConfigId,   // 装备配表id
			//		Lv:             equip.Level,      // 当前等级
			//		Exp:            equip.Exp,        // 当前经验
			//		SkillId:        equip.SkillNumId, // 技能id
			//		MainAttr:       equip.MainAttr,   // 主属性类型
			//		SubAttr:        equip.SubAttr,    // 副属性类型
			//	})
			//})
			threading.RunSafe(func() {
				e := &taptap.EquipDelete{
					PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
					EquipId:           delId,            // 删除的装备唯一id
					ConfigId:          equip.ConfigId,   // 装备配表id
					Lv:                equip.Level,      // 当前等级
					Exp:               equip.Exp,        // 当前经验
					SkillId:           equip.SkillNumId, // 技能id
					MainAttr:          equip.MainAttr,   // 主属性类型
					SubAttr:           equip.SubAttr,    // 副属性类型
				}
				taptap.WriteDataLog(taptap.LogEquipDelete, h.actor.uid, h.actor.Account.TapUserInfo, e)
			})

			delete(h.actor.GetEquipData().Equips, delId)

			hadDel = append(hadDel, delId)
		}
	}

	err := h.SaveDB()
	if err != nil {
		return err
	}

	h.Debug("del equip success %v", hadDel)
	return nil
}

func (h *EquipHandler) IsExistEquip(equipId uint64) bool {
	_, ok := h.actor.GetEquipData().Equips[equipId]
	return ok
}

func (h *EquipHandler) GetEquip(equipId uint64) (*cmd.PCommonEquipInfo, error) {
	if !h.IsExistEquip(equipId) {
		return nil, fmt.Errorf("not found equip: %d", equipId)
	}

	return h.actor.GetEquipData().Equips[equipId], nil
}

// 装备喂养check
func (h *EquipHandler) equipCostCheck(costIds []uint64) cmd.ErrorCode {

	for _, id := range costIds {
		// 是否存在
		equip, err := h.GetEquip(id)
		if err != nil {
			return cmd.ErrorCode_EquipNotExist
		}

		// 是否穿戴中
		if equip.CardId != 0 {
			return cmd.ErrorCode_EquipIsWear
		}
	}

	return cmd.ErrorCode_Success
}

// 计算装备喂养exp
func (h *EquipHandler) calEquipCostExp(costIds []uint64) int32 {

	total := int32(0)
	for _, id := range costIds {
		equip, err := h.GetEquip(id)
		if err != nil {
			h.Debug("calEquipCostExp err:", err)
			continue
		}
		_, exp := h.getEquipExp(equip.ConfigId, equip.Level)
		total += exp
	}

	return total
}

// 按比例计算出电力消耗
func calCurrencyCost(addExp int32) int32 {
	cfg := excel.GetConfigMgr().GetCfg().EQUIP_EXP_ELE
	return int32(float32(addExp*cfg[1]) / float32(cfg[0]))
}

func (h *EquipHandler) getEquipExp(equipId, level int32) (int32, int32) {
	cfg := excel.GetItemMgr().GetById(equipId)
	if cfg == nil {
		h.Debug("calEquipCostExp cfg not found ", equipId)
		return 0, 0
	}

	expCfg := excel.GetEquipmentUpgradeMgr().GetById(level)
	if expCfg.EquipmentSSR == nil {
		h.Debug("calEquipCostExp expCfg not found ", level)
		return 0, 0
	}

	switch cfg.Quality {
	case common.ITEM_QUALITY_1:
		return expCfg.EquipmentN[0], expCfg.EquipmentN[1]
	case common.ITEM_QUALITY_4:
		return expCfg.EquipmentR[0], expCfg.EquipmentR[1]
	case common.ITEM_QUALITY_5:
		return expCfg.EquipmentSR[0], expCfg.EquipmentSR[1]
	case common.ITEM_QUALITY_6:
		return expCfg.EquipmentSSR[0], expCfg.EquipmentSSR[1]
	}
	return 0, 0
}

// 是否可以增加经验
func canAddEquipExp(equip *cmd.PCommonEquipInfo) bool {
	// 满级了
	if equip.GetLevel() >= getEquipLevelLimit(equip.ConfigId) {
		return false
	}

	return true
}

// 增加装备exp
func (h *EquipHandler) addEquipExp(equip *cmd.PCommonEquipInfo, addExp int32) {
	if !canAddEquipExp(equip) {
		return
	}

	equip.Exp += addExp

	for {
		if !canAddEquipExp(equip) {
			break
		}

		// 下一级的exp
		needExp, _ := h.getEquipExp(equip.ConfigId, equip.Level)
		if equip.Exp >= needExp {
			equip.Level += 1
			equip.Exp -= needExp
		} else {
			break
		}
	}

	h.Debugf("addEquipExp Id:%d level:%d Exp:%d", equip.ConfigId, equip.Level, addExp)
}

func getEquipLevelLimit(equipId int32) int32 {
	cfg := excel.GetItemMgr().GetById(equipId)
	if cfg == nil {
		return 0
	}
	level := excel.GetConfigMgr().GetCfg().EQUIP_MAX_LEVEL

	switch cfg.Quality {
	case common.ITEM_QUALITY_1:
		return level[0]
	case common.ITEM_QUALITY_4:
		return level[1]
	case common.ITEM_QUALITY_5:
		return level[2]
	case common.ITEM_QUALITY_6:
		return level[3]
	}

	return 0
}

func (h *EquipHandler) wearEquipByGM(cardId, equipId int32, commonData *clidto.Comdata) error {

	// 获取装备
	equip := &cmd.PCommonEquipInfo{}
	for _, info := range h.actor.GetEquipData().Equips {
		if equipId == info.ConfigId && equip.CardId == 0 {
			equip = info
			break
		}
	}
	if equip == nil {
		return fmt.Errorf("equip not found")
	}

	// 卡牌check
	card, err := h.actor.CardHandler.GetCard(uint32(cardId))
	if err != nil {
		return err
	}

	// 类型匹配
	cfg := excel.GetEquipmentMgr().GetById(int32(equip.ConfigId))
	if cfg == nil {
		return fmt.Errorf("type not match")
	}
	beastarCfg := excel.GetBeastarMgr().GetById(int32(cardId))
	if beastarCfg == nil {
		return fmt.Errorf("type not match")
	}

	// 处理逻辑
	_, cards, code := h.handleEquipWear(card, equip)
	if code != cmd.ErrorCode_Success {
		return fmt.Errorf("handleEquipWear %v", code)
	}

	err = h.actor.CardHandler.SaveDB()
	if err != nil {
		return err
	}

	err = h.SaveDB()
	if err != nil {
		return err
	}

	for _, v := range cards {
		commonData.Data.Card = append(commonData.Data.Card, h.actor.CardHandler.ToClientData(v))
	}

	commonData.Data.Equip = h.buildEquipList()

	h.Debug("wearEquipByGM success")
	return nil
}

// GetEquipList
//
//	@Description: 获取卡牌的装备数据
//	@receiver h
//	@param cardId 给定卡牌id
//	@return []*cmd.PCommonEquipInfo 装备数据列表
//	@return error
func (h *EquipHandler) GetEquipList(cardId int32) ([]*cmd.PCommonEquipInfo, error) {
	ret := make([]*cmd.PCommonEquipInfo, 0)
	card, err := h.actor.CardHandler.GetCard(uint32(cardId))
	if err != nil {
		return ret, nil // 此处返回nil，对新创建卡牌逻辑容错
	}
	for _, equipId := range card.EquipId {
		if equipId == 0 {
			continue
		}

		if equip, err := h.GetEquip(equipId); err != nil {
			return ret, err
		} else {
			ret = append(ret, equip)
		}
	}

	return ret, nil
}

// 获取主属性血量加成(绝对值)
func (h *EquipHandler) GetEquipMainAttrHp(equip *cmd.PCommonEquipInfo) int32 {
	// 不是生命属性
	if equip.MainAttr != 1 {
		return 0
	}

	cfg := excel.GetEquipmentMgr().GetById(equip.ConfigId)
	if cfg == nil {
		return 0
	}

	mainAttrCfg := excel.GetEquipmentMainattributeMgr().GetById(cfg.AtrType*100 + equip.Level)
	if mainAttrCfg == nil {
		return 0
	}
	return mainAttrCfg.MainHealth
}

// 获取副属性血量加成(百分比+绝对值)
func (h *EquipHandler) GetEquipSubAttrHp(equip *cmd.PCommonEquipInfo) map[int32]int32 {
	// 不是生命属性
	if equip.SubAttr != 5 && equip.SubAttr != 6 {
		return nil
	}
	cfg := excel.GetEquipmentMgr().GetById(equip.ConfigId)
	if cfg == nil {
		return nil
	}

	subAttrCfg := excel.GetEquipmentSubattributeMgr().GetById(cfg.AtrType*100 + equip.Level)
	if subAttrCfg == nil {
		return nil
	}
	// 百分比生命(2)
	if equip.SubAttr == 5 {
		return map[int32]int32{subAttrCfg.Sub5[1]: subAttrCfg.Sub5[2]}
	}
	// 绝对值生命(1)
	if equip.SubAttr == 6 {
		return map[int32]int32{subAttrCfg.Sub6[1]: subAttrCfg.Sub6[2]}
	}
	return nil
}

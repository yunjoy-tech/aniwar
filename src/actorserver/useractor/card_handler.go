package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"math"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type CardHandler struct {
	*UABaseHandler
}

func NewCardHandler(actor *UserActor) *CardHandler {
	h := &CardHandler{UABaseHandler: NewUABaseHandler(actor, "CardHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardBreakthroughReq), h.CardBreakthroughReq)       // 突破 (觉醒)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardSkillUpgradeReq), h.CardSkillUpgradeReq)       // 技能升级
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardCompoundReq), h.CardCompoundReq)               // 潜力升级
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardCharacterBreakReq), h.CardCharacterBreakReq)   // 性格突破
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardCharacterUnlockReq), h.CardCharacterUnlockReq) // 性格解锁
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardCharacterChangeReq), h.CardCharacterChangeReq) // 性格切换
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardLevelUpReq), h.CardLevelUpReq)                 // 使用经验道具升级

	return h
}

// Init 初始化模块数据
func (h *CardHandler) Init() error {
	// 初始化
	h.actor.Data.Cards = &pb.PCardData{
		Createtime: time.Now().Unix(),
		Card:       make(map[uint32]*pb.CardData),
	}

	if err := h.SaveDB(); err != nil {
		return err
	}
	h.Debug("init cards data success. player: %s", h.actor.ID())
	return nil
}

func (h *CardHandler) EnterGame() error {
	return h.tryRefreshCardInfo()
}

func (h *CardHandler) DailyRefresh() error {
	return nil
}

func (h *CardHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PCardData); ok {
		h.actor.Data.Cards = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *CardHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCard(h.actor.ID()), h.actor.Data.Cards
}

func NewCard(cardCfg *excel.BeastarCfg) *pb.CardData {
	baseId := uint32(cardCfg.GetId())

	skill := make(map[uint32]uint32)
	skill[0] = uint32(cardCfg.GetSkillID()[0])
	skill[1] = uint32(cardCfg.GetSkillID()[1])
	skill[2] = uint32(cardCfg.GetSkillID()[2])

	equip := make(map[uint32]uint64)
	equip[1] = 0
	equip[2] = 0
	equip[3] = 0

	card := &pb.CardData{
		BaseId:            baseId,
		CardLevel:         1,
		Hp:                uint32(cardCfg.GetHp()),
		SkillCfgId:        skill,
		CardExp:           0,
		CreateTimestamp:   time.Now().Unix(),
		SkinId:            uint32(cardCfg.GetSkin0()),
		BreakthroughLevel: 0,
		AwakenLevel:       0,
		CharacterLevel:    0,
		OldMaxHp:          0,
		EquipId:           equip,
		FavoriteLevel:     0,
		FavoriteExp:       0,
		IsNew:             true,
		Character:         []int32{int32(pb.CharacterType_CharacterType_Human)},
		CurCharacter:      int32(pb.CharacterType_CharacterType_Human),
		AddNum:            1,
	}

	return card
}

// 登录尝试刷新卡牌数据
func (h *CardHandler) tryRefreshCardInfo() error {
	// 最大血量
	data := h.actor.GetUserCardData()
	for _, card := range data.Card {
		_, err := h.TrySupplementMaxHp(card)
		if err != nil {
			h.Errorf("update card maxHp failed. err: %v", err)
			continue
		}
	}
	return h.SaveDB()
}

// 卡牌升级
func (h *CardHandler) CardLevelUpReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardLevelUpReq{}
	if err = in.UnmarshalData(req); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 取对应卡牌数据
	card, err := h.GetCard(uint32(req.CardId))
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}
	// 满级了？
	maxLevel, err := h.CheckCardLevelUp(card)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	if card.CardLevel >= maxLevel {
		return nil, fmt.Errorf("card level is max"), int32(pb.ErrorCode_CardLevelIsMax)
	}

	// 道具check
	costs := utils.ConvertItem(req.Items)

	var sumExp int32
	for k, v := range costs {
		if v <= 0 {
			return nil, fmt.Errorf("param error"), int32(pb.ErrorCode_ParamError)
		}
		cfg := excel.GetItemMgr().GetById(k)
		if cfg == nil {
			return nil, fmt.Errorf("config not found"), int32(pb.ErrorCode_NotFoundConfig)
		}
		if !(cfg.Type == int32(pb.ItemType_Material) && cfg.SubType == int32(pb.ItemMaterialType_ItemMaterialType_Card_Exp)) { // 8
			return nil, fmt.Errorf("param error"), int32(pb.ErrorCode_ParamError)
		}
		sumExp += cfg.UseEffectShow * v
	}
	costs[common.CURRENCY_ITEM_ID_2001] += sumExp * excel.GetConfigMgr().GetCfg().ROLE_UPGRADE_ELE_COST
	if !GetConsumeMgr(h.actor).CheckMapEnough(costs) {
		return nil, fmt.Errorf("item not enough"), int32(pb.ErrorCode_NotEnoughItem)
	}

	err = GetConsumeMgr(h.actor).ConsumeList(costs, h.actor.comData, common.CR_Card_Level_Up)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 升级前等级
	beforeCardLevel := card.CardLevel
	// 处理卡牌升级
	if _, err = h.AddExp(card, sumExp, true); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	// 升级后等级
	afterCardLevel := card.CardLevel

	// 埋点
	threading.RunSafe(func() {
		e := &taptap.CardLevelUp{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CardId:            req.CardId,                              // 卡牌id
			Items:             taptap.ConvertListStruct2Str(req.Items), // 使用的经验道具列表
			AddExp:            sumExp,                                  // 本次增加的经验值
			BeforeLv:          beforeCardLevel,                         // 升级前等级
			AfterLv:           afterCardLevel,                          // 升级后等级
		}
		taptap.WriteDataLog(taptap.LogType_CardLevelUp, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(card))
	rsp := &pb.LS2C_CardLevelUpRes{CommonData: h.actor.comData.FixDownComData()}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

// 卡牌突破
func (h *CardHandler) CardBreakthroughReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardBreakthroughReq{}
	if err = in.UnmarshalData(req); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 取对应卡牌数据
	card, err := h.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}

	// 返回数据
	rsp := &pb.LS2C_CardBreakthroughRes{}

	// 处理卡牌突破
	errorCode, isAwaken := h.handleCardBreakthrough(card, rsp)
	if errorCode != pb.ErrorCode_Success {
		return nil, err, int32(errorCode)
	}

	// 卡牌突破埋点
	threading.RunSafe(func() {
		e := &taptap.CardBreakThrough{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CardId:            req.CardId,
			BeforeLv:          card.BreakthroughLevel - 1,
			AfterLv:           card.BreakthroughLevel,
		}
		taptap.WriteDataLog(taptap.LogType_CardEreakThrough, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 发布事件
	e := event.NewBasicEvent(TASK_EVENT_BREAKTHROUGH, []int32{TASK_TYPE_104, TASK_TYPE_105}, nil)
	e.Set("card_id", int32(req.CardId))
	e.Set("level", int32(card.BreakthroughLevel))
	e.Set("is_awaken", isAwaken)
	if err = h.actor.eventManager.SyncPublish(e); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	rsp.CommonData = h.actor.comData.FixDownComData()
	return rsp, nil, int32(pb.ErrorCode_Success)
}

// 技能升级
func (h *CardHandler) CardSkillUpgradeReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardSkillUpgradeReq{}
	err = in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 取对应卡牌数据
	card, err := h.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}

	// 处理升级逻辑
	index := req.GetIndex()
	errorCode, beforeSkillLv, nextSkillLv := h.handleSkillUpgrade(card, index, h.actor.comData)
	if errorCode != pb.ErrorCode_Success {
		return nil, err, int32(errorCode)
	}

	// 返回数据
	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(card))
	rsp := &pb.LS2C_CardSkillUpgradeRes{CommonData: h.actor.comData.FixDownComData()}

	// 埋点
	threading.RunSafe(func() {
		e := &taptap.CardSkillUpgrade{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CardId:            req.CardId,    // 卡牌id
			Index:             index,         // 升级的技能位
			BeforeLv:          beforeSkillLv, // 升级前的技能等级
			AfterLv:           nextSkillLv,   // 升级后的技能等级
		}
		taptap.WriteDataLog(taptap.LogType_CardSkillUpgrade, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 发布事件
	e := event.NewBasicEvent(TASK_EVENT_SKILL_UPGRADE, []int32{TASK_TYPE_101, TASK_TYPE_105}, nil)
	e.Set("card_id", req.CardId)
	if err = h.actor.eventManager.SyncPublish(e); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	rsp.CommonData = h.actor.comData.FixDownComData()
	return rsp, nil, 0
}

// 卡牌觉醒
func (h *CardHandler) CardCompoundReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardCompoundReq{}
	err = in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 取卡牌
	card, err := h.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}

	// 逻辑处理
	errorCode := h.handleCardCompound(card, h.actor.comData)
	if errorCode != pb.ErrorCode_Success {
		return nil, err, int32(errorCode)
	}

	// 消息返回
	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(card))
	rsp := &pb.LS2C_CardCompoundRes{CommonData: h.actor.comData.FixDownComData()}

	// 埋点
	threading.RunSafe(func() {
		e := &taptap.CardCompound{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CardId:            req.CardId,           // 卡牌id
			BeforeLv:          card.AwakenLevel - 1, // 觉醒前等级
			AfterLv:           card.AwakenLevel,     // 觉醒后等级
		}
		taptap.WriteDataLog(taptap.LogType_CardCompound, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 发布事件
	e := event.NewBasicEvent(TASK_EVENT_COMPOUND, []int32{TASK_TYPE_102, TASK_TYPE_105}, nil)
	e.Set("card_id", req.CardId)
	e.Set("level", card.AwakenLevel)
	if err = h.actor.eventManager.SyncPublish(e); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	rsp.CommonData = h.actor.comData.FixDownComData()
	return rsp, nil, 0
}

// 性格突破
func (h *CardHandler) CardCharacterBreakReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardCharacterBreakReq{}
	err = in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 取卡牌
	card, err := h.GetCard(req.CardId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}
	cardCfg := excel.GetBeastarMgr().GetById(int32(req.CardId))
	if cardCfg == nil {
		return nil, fmt.Errorf("not found card: %d config", req.CardId), int32(pb.ErrorCode_NotFoundConfig)
	}

	// 等级检查
	limit := GetCharacterBreakLimit(card)
	if card.CardLevel < uint32(limit) {
		return nil, fmt.Errorf("card level not enough"), int32(pb.ErrorCode_CardLevelNotEnough)
	}

	// 道具扣除
	cfg := excel.GetCharacterMgr().GetById(int32(card.BaseId*100 + card.CharacterLevel + 1))
	if !GetConsumeMgr(h.actor).CheckMapEnough(cfg.CharacterCost) {
		return nil, fmt.Errorf("item not enough"), int32(pb.ErrorCode_NotEnoughItem)
	}

	if err = GetConsumeMgr(h.actor).ConsumeList(cfg.CharacterCost, h.actor.comData, common.CR_Card_Character_Break); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 突破
	card.CharacterLevel += 1

	if _, err = h.TrySupplementMaxHp(card); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 消息返回
	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(card))
	rsp := &pb.LS2C_CardCharacterBreakRes{CommonData: h.actor.comData.FixDownComData()}

	// 埋点
	threading.RunSafe(func() {
		e := &taptap.CardCharacterBreak{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CardId:            req.CardId,              // 卡牌id
			BeforeLv:          card.CharacterLevel - 1, // 升级前等级
			AfterLv:           card.CharacterLevel,     // 升级后等级
		}
		taptap.WriteDataLog(taptap.LogType_CardCharacterBreak, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 发布事件
	e := event.NewBasicEvent(TASK_EVENT_CHARACTER_BREAK, []int32{TASK_TYPE_103, TASK_TYPE_105}, nil)
	e.Set("card_id", req.CardId)
	e.Set("level", card.CharacterLevel)
	if err = h.actor.eventManager.SyncPublish(e); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	rsp.CommonData = h.actor.comData.FixDownComData()
	h.Debug("CardCharacterBreakReq handle:", in.UserId)
	return rsp, nil, 0
}

func (h *CardHandler) CardCharacterUnlockReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardCharacterUnlockReq{}
	err = in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 参数校验
	if req.CharacterId <= pb.CharacterType_CharacterType_None || req.CharacterId >= pb.CharacterType_CharacterType_Max {
		return nil, fmt.Errorf("param error %d", req.CharacterId), int32(pb.ErrorCode_ParamError)
	}

	// 取卡牌
	card, err := h.GetCard(uint32(req.CardId))
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}

	// 已经解锁了
	for _, v := range card.Character {
		if v == int32(req.CharacterId) {
			return nil, fmt.Errorf("character has unlock %d", req.CharacterId), int32(pb.ErrorCode_ParamError)
		}
	}

	// 品质校验
	rarity, err := GetCardRarityById(req.CardId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_ParamError)
	}
	if rarity <= common.POTENTIAL_SR {
		return nil, fmt.Errorf("card rarity limit %d", rarity), int32(pb.ErrorCode_ParamError)
	}

	// 解锁消耗
	cost := getCharacterUnlockCost(rarity)
	if cost != nil {
		if !GetConsumeMgr(h.actor).CheckEnough(cost.Key, cost.Val) {
			return nil, fmt.Errorf("item not enough"), int32(pb.ErrorCode_NotEnoughItem)
		}
		if err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{cost.Key: cost.Val}, h.actor.comData, common.CR_CHARACTER_UNLOCK); err != nil {
			return nil, err, int32(pb.ErrorCode_InternalError)
		}
	}

	// 解锁
	card.Character = append(card.Character, int32(req.CharacterId))
	card.CurCharacter = int32(req.CharacterId)

	if _, err = h.TrySupplementMaxHp(card); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 消息返回
	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(card))
	return &pb.LS2C_CardCharacterUnlockRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func getCharacterUnlockCost(rarity int32) *excel.KeyVal {
	cost := excel.GetConfigMgr().GetCfg().CHARACTER_UNLOCK_COST
	if rarity == common.POTENTIAL_SSR && len(cost) >= 1 {
		return cost[0]
	}
	if rarity == common.POTENTIAL_SP && len(cost) >= 2 {
		return cost[1]
	}

	return nil
}

func (h *CardHandler) CardCharacterChangeReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD)
	if err != nil {
		return nil, err, int32(code)
	}
	req := &pb.C2LS_CardCharacterChangeReq{}
	err = in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 参数校验
	if req.CharacterId <= pb.CharacterType_CharacterType_None || req.CharacterId >= pb.CharacterType_CharacterType_Max {
		return nil, fmt.Errorf("param error %d", req.CharacterId), int32(pb.ErrorCode_ParamError)
	}

	// 是否在副本中
	// if h.actor.ChapterHandler.IsInSubLevel() {
	// 	return nil, fmt.Errorf("illegal operation"), int32(pb.ErrorCode_IllegalOperationError)
	// }

	// 取卡牌
	card, err := h.GetCard(uint32(req.CardId))
	if err != nil {
		return nil, err, int32(pb.ErrorCode_CardNotExist)
	}

	// 是否解锁
	var b bool
	for _, v := range card.Character {
		if v == int32(req.CharacterId) {
			b = true
			break
		}
	}
	if !b {
		return nil, fmt.Errorf("character not unlock %d", req.CharacterId), int32(pb.ErrorCode_ParamError)
	}

	// 品质校验
	rarity, err := GetCardRarityById(req.CardId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_ParamError)
	}
	if rarity <= common.POTENTIAL_SR {
		return nil, fmt.Errorf("card rarity limit %d", rarity), int32(pb.ErrorCode_ParamError)
	}

	// 切换
	card.CurCharacter = int32(req.CharacterId)

	if _, err = h.TrySupplementMaxHp(card); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 消息返回
	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(card))
	return &pb.LS2C_CardCharacterChangeRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// 卡牌突破处理逻辑
func (h *CardHandler) handleCardBreakthrough(c *pb.CardData, rsp *pb.LS2C_CardBreakthroughRes) (pb.ErrorCode, bool) {
	var isAwaken bool
	cardId := c.BaseId
	oldBreakthroughLevel := c.BreakthroughLevel
	newBreakthroughLevel := oldBreakthroughLevel + 1
	newBreakthroughLevelId := cardId*100 + newBreakthroughLevel
	cfg := excel.GetEvolutionMgr().GetById(int32(newBreakthroughLevelId))

	if cfg == nil {
		h.Warnf("card: %d not found breakthrough level: %d, %d config", cardId, newBreakthroughLevel, newBreakthroughLevelId)
		return pb.ErrorCode_NotFoundConfig, isAwaken
	}

	if len(cfg.NextRequire) == 0 {
		h.Warnf("card: %d breakthrough level: %d consumable materials empty", cardId, newBreakthroughLevelId)
		return pb.ErrorCode_ConfigError, isAwaken
	}
	if cfg.Evolution != int32(c.CardLevel) {
		return pb.ErrorCode_CardLevelNotEnough, isAwaken
	}

	// 道具消耗check
	if !GetConsumeMgr(h.actor).CheckMapEnough(cfg.NextRequire) {
		return pb.ErrorCode_NotEnoughItem, isAwaken
	}

	// 扣除道具
	err := GetConsumeMgr(h.actor).ConsumeList(cfg.NextRequire, h.actor.comData, common.CR_Card_Breakthrough_Upgrade)
	if err != nil {
		h.Error("ConsumeList err:", err)
		return pb.ErrorCode_InternalError, isAwaken
	}

	// 突破奖励
	if len(cfg.Reward) > 0 {
		rsp.DropChange, err = GetDropMgr(h.actor).DropList2(cfg.Reward, true, nil, h.actor.comData, common.CR_Card_Breakthrough_Upgrade)
		if err != nil {
			return pb.ErrorCode_InternalError, isAwaken
		}
		// 尝试穿戴皮肤（无皮肤跳过）
		for id := range cfg.Reward {
			itemCfg := excel.GetItemMgr().GetById(id)
			if itemCfg == nil {
				continue
			}
			if itemCfg.Type == int32(pb.ItemType_CardSkin) {
				c.SkinId = uint32(itemCfg.SystemId)
				break
			}
		}
	}
	c.BreakthroughLevel = newBreakthroughLevel

	_, err = h.AddExp(c, 0, false)
	if err != nil {
		return pb.ErrorCode_InternalError, isAwaken
	}

	if err = h.SaveDB(); err != nil {
		h.Error("handleCardBreakthrough save err:", err)
		return pb.ErrorCode_InternalError, isAwaken
	}

	_, err = h.TrySupplementMaxHp(c)
	if err != nil {
		return pb.ErrorCode_InternalError, isAwaken
	}

	h.actor.comData.Data.Card = append(h.actor.comData.Data.Card, h.ToClientData(c))
	rsp.CommonData = h.actor.comData.FixDownComData()
	if c.BreakthroughLevel == uint32(excel.GetConfigMgr().GetCfg().AWAKEN_PROFILE_GET) {
		isAwaken = true
	}
	return pb.ErrorCode_Success, isAwaken
}

// 技能升级逻辑处理
func (h *CardHandler) handleSkillUpgrade(c *pb.CardData, index uint32, commonData *clidto.Comdata) (pb.ErrorCode, int32, int32) {
	// if _, ok := c.SkillCfgId[index]; !ok {
	// 	return pb.ErrorCode_InvalidParam, 0, 0
	// }
	//
	// skillCfgId := c.SkillCfgId[index]
	// skillCfg := excel.GetSkillMgr().GetById(int32(skillCfgId))
	// if skillCfg == nil {
	// 	h.Errorf("skill config not found: %d", skillCfgId)
	// 	return pb.ErrorCode_NotFoundConfig, 0, 0
	// }
	//
	// // 等级限制
	// unlockLimit := excel.GetConfigMgr().GetCfg().SKILL_UNLOCK_LIMIT
	// if skillCfg.Lv > int32(len(unlockLimit)) {
	// 	return pb.ErrorCode_ConfigError, 0, 0
	// }
	//
	// lv := skillCfg.Lv - 1
	// if lv >= 0 && lv < int32(len(unlockLimit)) && c.CardLevel < uint32(unlockLimit[skillCfg.Lv-1]) {
	// 	return pb.ErrorCode_CardLevelNotEnough, 0, 0
	// }
	//
	// // 满级了
	// if 0 >= skillCfg.GetNextLevelId() {
	// 	h.Errorf("skill: %d has invalid next level id: %d", skillCfgId, skillCfg.GetNextLevelId())
	// 	return pb.ErrorCode_ConfigError, 0, 0
	// }
	//
	// // 升级前的技能等级
	// beforeSkillLv := skillCfg.Lv
	//
	// // 道具消耗check
	// if !GetConsumeMgr(h.actor).CheckMapEnough(skillCfg.UpgradeCost) {
	// 	return pb.ErrorCode_NotEnoughItem, 0, 0
	// }
	//
	// err := GetConsumeMgr(h.actor).ConsumeList(skillCfg.UpgradeCost, commonData, common.CR_Card_Skill_Upgrade)
	// if err != nil {
	// 	return pb.ErrorCode_InternalError, 0, 0
	// }
	//
	// // 下一技能等级索引
	// nextSkillIndex := uint32(skillCfg.GetNextLevelId())
	// c.SkillCfgId[index] = nextSkillIndex
	// err = h.SaveDB()
	// if err != nil {
	// 	return pb.ErrorCode_InternalError, 0, 0
	// }
	//
	// // 返回下一技能等级
	// nextSkillCfgId := c.SkillCfgId[index]
	// nextSkillCfg := excel.GetSkillMgr().GetById(int32(nextSkillCfgId))
	//
	// return pb.ErrorCode_Success, beforeSkillLv, nextSkillCfg.Lv // 返回升级后的技能等级
	return pb.ErrorCode_Success, 0, 0
}

// 卡牌觉醒处理逻辑
func (h *CardHandler) handleCardCompound(c *pb.CardData, commonData *clidto.Comdata) pb.ErrorCode {
	cardCfg := excel.GetBeastarMgr().GetById(int32(c.BaseId))
	if cardCfg == nil {
		h.Errorf("not found card: %d config", c.BaseId)
		return pb.ErrorCode_NotFoundConfig
	}

	newAwakenLevel := c.AwakenLevel + 1
	cardAwakenId := cardCfg.Potential*100 + int32(newAwakenLevel)
	cardAwakenCfg := excel.GetPotentialMgr().GetById(cardAwakenId)
	// 觉醒上限
	if cardAwakenCfg == nil {
		h.Warnf("not found card: %d compoud: %d config", c.BaseId, cardAwakenId)
		return pb.ErrorCode_NotFoundConfig
	}

	// 碎片消耗check
	costValue, err := GetAwakenCostValue(cardCfg.GetRarity(), c.AwakenLevel)
	if err != nil {
		h.Warnf("handleCardCompound err: ", err)
		return pb.ErrorCode_ConfigError
	}
	if !GetConsumeMgr(h.actor).CheckEnough(cardCfg.GetPotentialCost(), int32(costValue)) {
		return pb.ErrorCode_NotEnoughItem
	}

	// 消耗扣除
	err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{cardCfg.GetPotentialCost(): int32(costValue)}, commonData, common.CR_Card_Awaken_Upgrade)
	if err != nil {
		return pb.ErrorCode_InternalError
	}

	c.AwakenLevel = newAwakenLevel
	err = h.SaveDB()
	if err != nil {
		return pb.ErrorCode_InternalError
	}

	_, err = h.TrySupplementMaxHp(c)
	if err != nil {
		return pb.ErrorCode_InternalError
	}
	return pb.ErrorCode_Success
}

func (h *CardHandler) buildCardList() []*pb.PClientCardInfo {
	cards := make([]*pb.PClientCardInfo, 0)
	for _, card := range h.actor.GetUserCardData().Card {
		cards = append(cards, h.ToClientData(card))
	}

	return cards
}

// GetCard 根据id取卡牌数据
func (h *CardHandler) GetCard(cardId uint32) (*pb.CardData, error) {
	if !h.IsExistCard(cardId) {
		return nil, fmt.Errorf("not found card: %d", cardId)
	}

	return h.actor.GetUserCardData().Card[cardId], nil
}

func (h *CardHandler) GetCardSkillLvSum(cardId uint32) int32 {
	// card, err := h.GetCard(cardId)
	// if err != nil {
	// 	return 0
	// }
	//
	// var sum int32 = 0
	// for _, v := range card.SkillCfgId {
	// 	skillCfg := excel.GetSkillMgr().GetById(int32(v))
	// 	if skillCfg != nil {
	// 		sum += skillCfg.Lv
	// 	}
	// }
	// return sum
	return 0
}

// 对外方法

// IsExistCard 是否存在指定卡牌,如果存在返回true
func (h *CardHandler) IsExistCard(cardId uint32) bool {
	_, ok := h.actor.GetUserCardData().Card[cardId]
	return ok
}

func (h *CardHandler) AddCard(itemCfg *excel.ItemCfg, addNum uint32, commonData *clidto.Comdata) (*pb.DropChange, error) {
	if addNum <= 0 {
		return nil, fmt.Errorf("param err: %d", addNum)
	}
	cardId := itemCfg.SystemId
	cardCfg := excel.GetBeastarMgr().GetById(cardId)
	if cardCfg == nil {
		return nil, fmt.Errorf("not found card: %d config", cardId)
	}

	var (
		dropChange = &pb.DropChange{}
	)

	exist := h.IsExistCard(uint32(cardId))

	// 发整卡
	if !exist {
		card := NewCard(cardCfg)
		// 初始化血量标记
		// maxHp, err := h.CalcMaxHp(card)
		// if err != nil {
		// 	return nil, err
		// }
		card.Hp = 0       /*maxHp*/
		card.OldMaxHp = 0 /*maxHp*/
		// 初始化皮肤
		_, err := h.actor.SkinHandler.tryInitSkin(cardId)
		if err != nil {
			return nil, err
		}
		h.actor.GetUserCardData().Card[uint32(cardId)] = card

		if err = h.SaveDB(); err != nil {
			h.Error("AddCard save err:", err)
			return nil, err
		}

		commonData.Data.Card = append(commonData.Data.Card, h.ToClientData(card))

		// 发布事件
		errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_CARD_CREATE, []int32{TASK_TYPE_401, TASK_TYPE_402, TASK_TYPE_403, TASK_TYPE_505}, map[string]interface{}{
			"cardId": cardId,
			"rarity": cardCfg.Rarity,
			"total":  int32(len(h.actor.GetUserCardData().Card)),
		}))
		if errx != nil {
			h.Error(errx)
		}

		dropChange.Items = append(dropChange.Items, &pb.ItemReward{
			ItemId: uint32(itemCfg.ItemId),
			Num:    1,
		})

		// 初始化羁绊
		// h.actor.UserRelationHandler.InitCardRelation(cardId, commonData)
		// 初始化通话信号
		// h.actor.UserCallSysHandler.AddCardInitSignalLevel(cardId)

		exist = true
		addNum -= 1
		h.Debugf("add new card, player: %s, cardId:%d", h.actor.ID(), cardId)
	}

	// 存在则转换成其他道具
	if exist && addNum > 0 {
		// 记录获得次数
		card, err := h.GetCard(uint32(cardId))
		if err != nil {
			return nil, err
		}
		before := card.AddNum
		card.AddNum += addNum
		if err = h.SaveDB(); err != nil {
			return nil, err
		}

		// 计算奖励
		reward := make(map[uint32]uint32)

		var sum int32
		for _, v := range GetAwakenCostCfg(cardCfg.GetRarity()) {
			sum += v
		}
		chipNum := getDuplicateCardChip(cardCfg.GetRarity())
		exchange := uint32(math.Ceil(float64(sum)/float64(chipNum)) + 1)

		for i := uint32(1); i <= addNum; i++ {
			if before+i <= exchange {
				// 给角色碎片
				reward[getCardChipId(cardId)] += chipNum
			} else {
				// 给兑换货币
				reward[common.CURRENCY_ITEM_ID_2009] += getDuplicateCardMoney(cardCfg.GetRarity())
			}
		}

		// 加背包
		_dropChange, err := GetDropMgr(h.actor).DropList(reward, true, nil, commonData, common.CR_Add_Card)
		if err != nil {
			return nil, err
		}

		mergeDropChange(dropChange, _dropChange)
		h.Debugf("卡牌转材料 cardId: %d, reward: %+v, change:%+v ", cardId, reward, _dropChange)
	}
	return dropChange, nil
}

func (h *CardHandler) SetCardHp(cardId int32, curHp int32) (error, *pb.PClientCardInfo) {
	// 获取卡牌
	card, err := h.GetCard(uint32(cardId))
	if err != nil {
		return err, nil
	}
	// 判定hp值
	if curHp < 0 {
		return fmt.Errorf("invalid hp value %d", curHp), h.ToClientData(card)
	}
	card.Hp = uint32(curHp)
	err = h.SaveDB()
	if err != nil {
		return err, nil
	}

	return nil, h.ToClientData(card)
}

func getCardChipId(cardId int32) uint32 {
	return uint32((60000+cardId)*100 + 1)
}

func getDuplicateCardChip(rarity int32) uint32 {
	values := excel.GetConfigMgr().GetCfg().RECRUIT_SAME_HERO_SCRIPT

	var v int32
	if rarity == common.POTENTIAL_R {
		v = values[0]
	} else if rarity == common.POTENTIAL_SR {
		v = values[1]
	} else if rarity == common.POTENTIAL_SSR {
		v = values[2]
	} else if rarity == common.POTENTIAL_SP {
		v = values[3]
	}
	return uint32(v)
}

func getDuplicateCardMoney(rarity int32) uint32 {
	values := excel.GetConfigMgr().GetCfg().RECRUIT_SCRIPT_EXCHANGE

	var v int32
	if rarity == common.POTENTIAL_R {
		v = values[0]
	} else if rarity == common.POTENTIAL_SR {
		v = values[1]
	} else if rarity == common.POTENTIAL_SSR {
		v = values[2]
	} else if rarity == common.POTENTIAL_SP {
		v = values[3]
	}

	return uint32(v)
}

// GetCards 获取卡牌列表
func (h *CardHandler) GetCards(cardIds []int32) []*pb.CardData {
	var (
		cards = make([]*pb.CardData, 0)
	)

	for _, each := range cardIds {
		card, err := h.GetCard(uint32(each))
		if err != nil {
			h.Debugf("不存在的card数据, cardId=%d", each)
			continue
		}

		cards = append(cards, card)
	}

	return cards
}

// TrySupplementMaxHp
//
//	@Description: 尝试补齐最大血量值,内部调用,没有推送卡牌数据变更
//	@receiver h
//	@param card
//	@return error
func (h *CardHandler) TrySupplementMaxHp(card *pb.CardData) (bool, error) {
	var change bool
	// newMaxHp, err := h.CalcMaxHp(card)
	// if err != nil {
	// 	return change, err
	// }
	var newMaxHp uint32
	oldMaxHp := card.GetOldMaxHp()
	if newMaxHp != oldMaxHp {
		change = true
	}
	card.OldMaxHp = newMaxHp

	// 血量上限提高,血量补齐
	if newMaxHp > oldMaxHp {
		err, _ := h.SetCardHp(int32(card.BaseId), int32(card.Hp+newMaxHp-oldMaxHp))
		if err != nil {
			return change, err
		}
	} else {
		// 上限降低
		//	如果当前血量高于新的最大血量，血量降低到最大血量。
		//	如果本身就低于新的最大血量，不处理
		if card.Hp > newMaxHp {
			err, _ := h.SetCardHp(int32(card.BaseId), int32(newMaxHp))
			if err != nil {
				return change, err
			}
		}
	}
	// 血量变化事件
	if change {
		errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_CHANGE_HP, []int32{}, map[string]interface{}{
			"card_id": card.BaseId,
			"hp":      newMaxHp,
		}))
		if errx != nil {
			h.Error(errx)
		}
	}
	h.Debugf("尝试补齐卡牌最大血量 cardId: %d", card.BaseId)
	return change, nil
}

// 是否可以增加好感度经验
func GetMaxFavoriteLevel() uint32 {
	// 满级了
	maxLevel := int32(0)
	excel.GetHeroLevelMgr().Foreach(func(cfg *excel.HeroLevelCfg) bool {
		if cfg.Favor > 0 && cfg.GetId() > maxLevel {
			maxLevel = cfg.GetId()
		}
		return true
	}, true)

	return uint32(maxLevel)
}

func (h *CardHandler) AddFavoriteExpById(cardId int32, addExp uint32) (*pb.PClientCardInfo, pb.ErrorCode) {
	card, err := h.GetCard(uint32(cardId))
	if err != nil {
		h.Debugf("AddFavoriteExpById, err %v", err)
		return nil, pb.ErrorCode_InternalError
	}

	errCode := h.AddFavoriteExp(card, addExp)
	if errCode != pb.ErrorCode_Success {
		h.Warnf("AddFavoriteExpById : AddFavoriteExp err currlevel : %v, addExp :%v, maxLevel:%v", card.GetFavoriteLevel(), addExp)
		return nil, errCode
	}

	if err = h.SaveDB(); err != nil {
		h.Debugf("AddFavoriteExpById, err %v", err)
		return nil, pb.ErrorCode_SaveDBError
	}

	return h.ToClientData(card), pb.ErrorCode_Success
}

// AddFavoriteExp 计算增加好感度值
func (h *CardHandler) AddFavoriteExp(card *pb.CardData, addExp uint32) pb.ErrorCode {
	maxLevel := GetMaxFavoriteLevel() // 配置最大等级
	totalFavorExp := card.FavoriteExp + addExp
	var targetLevel uint32
	var isLevelUp bool
	// 模拟升级 获得 target level, 防止升级过程中出错
	for targetLevel = card.GetFavoriteLevel(); targetLevel < maxLevel; targetLevel++ {

		// 下一级的exp
		needExp := excel.GetHeroLevelMgr().GetById(int32(targetLevel + 1))
		if needExp == nil {
			h.Errorf("AddFavoriteExp: GetHeroLevelCfg err level :%v,get exp :%v", targetLevel, addExp)
			return pb.ErrorCode_InternalError
		}
		if int32(totalFavorExp) < needExp.GetFavor() {
			break
		}
		totalFavorExp -= uint32(needExp.GetFavor())
		isLevelUp = true
	}

	// 好感度升级触发通话
	if card.FavoriteLevel != targetLevel {
		for i := card.FavoriteLevel + 1; i <= targetLevel; i++ {
			h.Debugf("卡片[%d]好感度升级[%d]->[%d]触发通话", card.BaseId, card.FavoriteLevel, i)
			// h.actor.UserCallSysHandler.FavorUpTriggerCall(int32(card.BaseId), int32(i))
		}
	}
	// 据俊杰说升满，抹掉超出部分经验
	if targetLevel >= maxLevel {
		h.Infof("AddFavoriteExp : levelMax card Level :%v max Level %v get exp %v", card.FavoriteLevel, maxLevel, addExp)
		card.FavoriteLevel = maxLevel
		card.FavoriteExp = 0
	} else {
		card.FavoriteLevel = targetLevel
		card.FavoriteExp = totalFavorExp
	}

	if err := h.SaveDB(); err != nil {
		return pb.ErrorCode_SaveDBError
	}

	h.Debugf("AddFavoriteExp Id:%d level:%d Exp:%d", card.BaseId, card.FavoriteLevel, card.FavoriteExp)
	if isLevelUp {
		_, err := h.TrySupplementMaxHp(card)
		if err != nil {
			h.Error()
		}
	}
	return pb.ErrorCode_Success
}

// 是否可以增加经验
func (h *CardHandler) CheckCardLevelUp(c *pb.CardData) (uint32, error) {
	i := int32(c.GetBaseId()*100 + c.GetBreakthroughLevel() + 1)
	cfg := excel.GetEvolutionMgr().GetById(i)
	if cfg == nil {
		return 0, fmt.Errorf("config not found %d", i)
	}
	maxLevel := uint32(cfg.Evolution)
	if c.CardLevel >= maxLevel {
		h.Warnf("CheckCardLevelUp: Card Level Max Card Level :%v, maxLevel:%v", c.CardLevel, maxLevel)
		return maxLevel, fmt.Errorf("Card Level Max Card Level :%v, maxLevel:%v", c.CardLevel, maxLevel)
	}

	return maxLevel, nil
}

// AddExp 计算增加经验值
// 返回值：真实加成的经验值,error
func (h *CardHandler) AddExp(c *pb.CardData, value int32, triggerTask bool) (int32, error) {
	if value < 0 {
		h.Errorf("AddExp:err add exp <= 0")
		return 0, fmt.Errorf("AddExp:err add exp <= 0")
	}

	maxCardLevel, err := h.CheckCardLevelUp(c)
	if err != nil {
		h.Warnf("AddExp:CheckCardLevelUp err :card baseId %v level %v BreakthroughLevel %v ", c.GetBaseId(), c.GetCardLevel(), c.GetBreakthroughLevel())
		return 0, fmt.Errorf("CheckCardLevelUp is false")
	}

	var totalAddExp = value + int32(c.GetCardExp())
	var realAddExp = 0 - int32(c.GetCardExp())
	var targetLevel uint32
	var isLevelUp = false
	for targetLevel = c.GetCardLevel(); targetLevel < maxCardLevel; targetLevel++ {
		cardLevelCfg := excel.GetHeroLevelMgr().GetById(int32(targetLevel))
		if cardLevelCfg == nil {
			h.Errorf("AddExp: GetHeroLevelCfg err level :%v", targetLevel)
			return 0, fmt.Errorf("Err Config HeroLevelCfg card Level: %d ", c.GetCardLevel())
		}

		if cardLevelCfg.GetExp() > totalAddExp {
			realAddExp += totalAddExp
			break
		}

		totalAddExp -= cardLevelCfg.GetExp()
		realAddExp += cardLevelCfg.GetExp()
		isLevelUp = true
	}
	defer func() {
		if isLevelUp && triggerTask {
			errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_LEVEL_UPGRADE, []int32{TASK_TYPE_121, TASK_TYPE_105, TASK_TYPE_505}, map[string]interface{}{
				"card_id": c.BaseId,
				"level":   targetLevel,
			}))
			if errx != nil {
				h.Error(errx)
			}
		}

		h.Debugf("AddExp cardId:%d level:%d readAddExp:%d", c.BaseId, c.GetCardLevel(), realAddExp)
		_, err = h.TrySupplementMaxHp(c)
		if err != nil {
			h.Error()
		}

	}()
	// 升级到最大等级抹掉多余经验
	// if targetLevel >= maxCardLevel {
	//	h.Infof("AddExp : levelMax card Level :%v max Level %v", c.CardLevel, maxCardLevel)
	//
	//	c.CardLevel = targetLevel
	//	c.CardExp = 0
	//	return realAddExp, nil
	// }

	c.CardLevel = targetLevel
	c.CardExp = uint32(totalAddExp)

	return realAddExp, nil
}

// 增加卡牌经验值
func (h *CardHandler) AddExpByTroop(troopType pb.CardTroopType, troopId int32, addExp uint64) []*pb.PCardBattleSettlement {
	var (
		changeCards = make([]*pb.PCardBattleSettlement, 0)
	)
	if addExp <= 0 {
		return changeCards
	}

	cardIds := h.actor.TroopHandler.GetTroopCardIds(int32(troopType), troopId)
	if len(cardIds) <= 0 {
		h.Errorf("未找到队伍信息, troopType=%d, troopId=%d", pb.ErrorCode_Chapter_empty_troop, troopId)
		return changeCards
	}

	cards := h.actor.CardHandler.GetCards(cardIds)
	for _, card := range cards {
		realAddExp, err := h.AddExp(card, int32(addExp), false)
		if err != nil {
			h.Errorf("cardHandler.addExp got error, troopType=%d, troopId=%d, cardId=%d, err:%+v",
				pb.ErrorCode_Chapter_empty_troop, troopId, card.BaseId, err)
		}

		eachChangeCard := &pb.PCardBattleSettlement{
			CardId:    card.BaseId,
			CardLevel: card.CardLevel,
			CardExp:   uint32(realAddExp),
			CardHp:    card.Hp,
		}

		changeCards = append(changeCards, eachChangeCard)
	}

	return changeCards
}

// 卡牌增加经验
func (h *CardHandler) AddCardExpByIdList(cardList []int32, exp int32) ([]*pb.CommonCardExpReward, []*pb.CardData) {
	expRewards := make([]*pb.CommonCardExpReward, 0)
	cards := make([]*pb.CardData, 0)
	for _, roleId := range cardList {
		if card, err := h.actor.CardHandler.GetCard(uint32(roleId)); err == nil {
			addExp, err := h.actor.CardHandler.AddExp(card, exp, false) // 掉落奖励加经验不触发任务
			if err != nil {
				h.Warnf("AddCardExpByIdList failed, err: %v", err)
			}
			expRewards = append(expRewards, &pb.CommonCardExpReward{
				RoleId: roleId,
				Exp:    addExp,
			})
			cards = append(cards, card)
		}
	}
	err := h.actor.CardHandler.SaveDB()
	if err != nil {
		h.Warn(err)
	}

	return expRewards, cards
}

func (h *CardHandler) GetCardCount() int {
	return len(h.actor.GetUserCardData().Card)
}

// GetAllCardIds 获取所有的卡片Id
func (h *CardHandler) GetAllCardIds() []int32 {
	cardIds := make([]int32, 0, len(h.actor.GetUserCardData().Card))
	// 获取所有的卡牌Id
	for id := range h.actor.GetUserCardData().Card {
		cardIds = append(cardIds, int32(id))
	}
	return cardIds
}

func GetCardRarityByItemId(cardItemId int32) (int32, error) {
	// 道具表
	itemCfg := excel.GetItemMgr().GetById(cardItemId)
	if itemCfg == nil {
		return 0, fmt.Errorf("item config not found")
	}
	return GetCardRarityById(itemCfg.SystemId)
}

func GetCardRarityById(cardId int32) (int32, error) {
	// 角色表
	beastarCfg := excel.GetBeastarMgr().GetById(cardId)
	if beastarCfg == nil {
		return 0, fmt.Errorf("card config not found")
	}
	return beastarCfg.GetRarity(), nil
}

func (h *CardHandler) GetCardCountByQuality(rarity int32) int32 {
	var sum int32
	for _, card := range h.actor.GetUserCardData().Card {
		cfg := excel.GetBeastarMgr().GetById(int32(card.BaseId))
		if cfg == nil {
			continue
		}
		if cfg.Rarity == rarity {
			sum++
		}
	}
	return sum
}

func (h *CardHandler) GetCardCountByLevel(level int32) int32 {
	var sum int32
	for _, card := range h.actor.GetUserCardData().Card {
		if card.CardLevel >= uint32(level) {
			sum++
		}
	}
	return sum
}

func (h *CardHandler) buildCards(cards []int32) []*pb.PClientCardInfo {
	ret := make([]*pb.PClientCardInfo, 0)
	cardsList := h.GetCards(cards)
	for _, card := range cardsList {
		ret = append(ret, h.ToClientData(card))
	}

	return ret
}

// 处理卡牌红点已读
func (h *CardHandler) handleRedPoint(commonData *clidto.Comdata, param []int64) error {
	b := false
	for _, id := range param {
		card, err := h.GetCard(uint32(id))
		if err != nil {
			continue
		}
		card.IsNew = false
		commonData.Data.Card = append(commonData.Data.Card, h.ToClientData(card))
		b = true
	}

	if b {
		if err := h.SaveDB(); err != nil {
			return err
		}
	}

	return nil
}

// 计算好感度道具经验值
func calFavorExp(cardId int32, costs map[int32]int32) uint32 {
	var likeSum float32
	var unLikeSum float32
	var normalSum uint32

	cardCfg := excel.GetBeastarMgr().GetById(cardId)
	if cardCfg == nil {
		return 0
	}
	tempLike := make(map[int32]int32)
	for _, v := range cardCfg.FavorItem {
		tempLike[v] = 0
	}
	tempUnLike := make(map[int32]int32)
	for _, v := range cardCfg.DisfavorItem {
		tempUnLike[v] = 0
	}

	fixNum := excel.GetConfigMgr().GetCfg().HERO_LIKEITEM_PRO
	if len(fixNum) < 2 {
		return 0
	}

	for itemId, itemNum := range costs {
		itemCfg := excel.GetItemMgr().GetById(itemId)
		if itemCfg == nil {
			continue
		}
		// 喜欢礼物
		if _, ok := tempLike[itemId]; ok {
			likeSum += float32(itemCfg.UseEffectShow) * (1 + float32(fixNum[0])/100) * float32(itemNum)
			continue
		}
		// 不喜欢礼物
		if _, ok := tempUnLike[itemId]; ok {
			unLikeSum += float32(itemCfg.UseEffectShow) * (1 - float32(fixNum[1])/100) * float32(itemNum)
			continue
		}
		// 一般礼物
		normalSum += uint32(itemCfg.UseEffectShow * itemNum)
	}
	return uint32(math.Ceil(float64(likeSum+unLikeSum))) + normalSum
}

// 卡牌强化gm (1=等级,2=技能,3=突破等级,4=觉醒等级,5=性格等级,6=好感度等级)
func (h *CardHandler) SetSuperCardByGM(id int, typ int, commonData *clidto.Comdata) error {
	card, err := h.GetCard(uint32(id))
	if err != nil {
		return err
	}

	switch typ {
	case 1:
		card.CardLevel = getMaxLvTempalte(1)
	case 2:
		fixCard2(card)
	case 3:
		fixCard3(card)
	case 4:
		fixCard4(card)
	case 5:
		card.CharacterLevel = getMaxLvTempalte(5)
	case 6:
		card.FavoriteLevel = getMaxLvTempalte(6)
	default:
		card.CardLevel = getMaxLvTempalte(1)
		fixCard2(card)
		fixCard3(card)
		fixCard4(card)
		card.CharacterLevel = getMaxLvTempalte(5)
		card.FavoriteLevel = getMaxLvTempalte(6)
	}

	h.TrySupplementMaxHp(card)
	if err = h.SaveDB(); err != nil {
		return err
	}

	commonData.Data.Card = append(commonData.Data.Card, h.ToClientData(card))
	return nil
}

func fixCard2(card *pb.CardData) {
	m := make(map[uint32]uint32)
	// for k, skillCfgId := range card.SkillCfgId {
	// 	var max = skillCfgId
	// 	for i := 0; i < 200; i++ {
	// 		skillCfg := excel.GetSkillMgr().GetById(int32(max))
	// 		if skillCfg == nil {
	// 			break
	// 		}
	//
	// 		// 满级了
	// 		if 0 >= skillCfg.GetNextLevelId() {
	// 			break
	// 		}
	// 		max = uint32(skillCfg.GetNextLevelId())
	// 	}
	// 	m[k] = max
	// }
	card.SkillCfgId = m
}

func fixCard3(card *pb.CardData) {
	for i := 0; i < 200; i++ {
		newBreakthroughLevel := card.BreakthroughLevel + 1
		newBreakthroughLevelId := card.BaseId*100 + newBreakthroughLevel
		cfg := excel.GetEvolutionMgr().GetById(int32(newBreakthroughLevelId))
		if cfg == nil {
			return
		}
		card.BreakthroughLevel++
	}
}

func fixCard4(card *pb.CardData) {
	cardCfg := excel.GetBeastarMgr().GetById(int32(card.BaseId))
	if cardCfg == nil {
		return
	}

	for i := 0; i < 200; i++ {
		newAwakenLevel := card.AwakenLevel + 1
		cardAwakenId := cardCfg.Potential*100 + int32(newAwakenLevel)
		cardAwakenCfg := excel.GetPotentialMgr().GetById(cardAwakenId)
		// 觉醒上限
		if cardAwakenCfg == nil {
			return
		}
		card.AwakenLevel++
	}
}

// 获取最大等级 1=角色等级 5=性格等级 6=好感度等级
func getMaxLvTempalte(typ int32) uint32 {
	var max int32
	excel.GetHeroLevelMgr().Foreach(func(cfg *excel.HeroLevelCfg) bool {
		if cfg.Id < max {
			return true
		}
		if typ == 1 {
			if cfg.Exp == 0 {
				return true
			}
		} else if typ == 5 {
			if cfg.CharacterExp == 0 {
				return true
			}
		} else if typ == 6 {
			if cfg.Favor == 0 {
				return true
			}
		} else {
			return true
		}
		max = cfg.Id
		return true
	}, true)
	return uint32(max)
}

func (h *CardHandler) GetCardCfg(cardId int32) *excel.BeastarCfg {
	cardCfg := excel.GetBeastarMgr().GetById(cardId)
	if cardCfg == nil {
		return nil
	}
	return cardCfg
}

// 获取性格突破次数的等级限制
func GetCharacterBreakLimit(c *pb.CardData) int32 {
	limit := excel.GetConfigMgr().GetCfg().CHARACTER_UNLOCK_LIMIT
	level := int32(0)
	for _, each := range limit {
		if c.CharacterLevel+1 >= uint32(each.Key) {
			level = each.Val
		}
	}

	return level
}

func GetAwakenCostValue(rarity int32, level uint32) (uint32, error) {
	values := GetAwakenCostCfg(rarity)
	if len(values) == 0 {
		return 0, fmt.Errorf("unrealized card rarity type %d", rarity)
	}
	return uint32(values[level]), nil
}

func GetAwakenCostCfg(rarity int32) []int32 {
	var values []int32
	if rarity == common.POTENTIAL_R {
		values = excel.GetConfigMgr().GetCfg().POTENTIAL_COST_R
	} else if rarity == common.POTENTIAL_SR {
		values = excel.GetConfigMgr().GetCfg().POTENTIAL_COST_SR
	} else if rarity == common.POTENTIAL_SSR {
		values = excel.GetConfigMgr().GetCfg().POTENTIAL_COST_SSR
	} else if rarity == common.POTENTIAL_SP {
		values = excel.GetConfigMgr().GetCfg().POTENTIAL_COST_SP
	}
	return values
}

// 转换成客户端卡牌数据
func (h *CardHandler) ToClientData(c *pb.CardData) *pb.PClientCardInfo {
	cardId := c.BaseId
	cardCfg := excel.GetBeastarMgr().GetById(int32(cardId))
	if cardCfg == nil {
		return nil
	}

	temp := &pb.PCommonCardInfo{}

	temp.CardId = cardId
	temp.CardLevel = c.GetCardLevel()
	temp.CardExp = uint64(c.GetCardExp())
	temp.Hp = c.GetHp()
	temp.CreateTimestamp = c.GetCreateTimestamp()

	skill := make([]uint32, len(c.SkillCfgId), len(c.SkillCfgId))
	for k, v := range c.SkillCfgId {
		skill[k] = v
	}
	temp.SkillId = skill

	temp.SkinId = c.GetSkinId()
	temp.BreakthroughLevel = c.GetBreakthroughLevel()
	temp.AwakenCfgId = uint32(cardCfg.Potential)*100 + c.AwakenLevel
	temp.CharacterLevel = c.GetCharacterLevel()

	equip := make([]uint64, len(c.EquipId), len(c.EquipId))
	for k, v := range c.EquipId {
		equip[k-1] = v
	}
	temp.EquipId = equip

	temp.FavoriteLevel = c.GetFavoriteLevel()
	temp.FavoriteExp = c.GetFavoriteExp()
	temp.Skins = h.actor.SkinHandler.buildSkinList(int32(cardId))
	temp.IsNew = c.GetIsNew()
	temp.FavoriteReward = c.GetFavoriteReward()
	temp.Character = c.GetCharacter()
	temp.CurCharacter = c.GetCurCharacter()

	return &pb.PClientCardInfo{Common: temp}
}

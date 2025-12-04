package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type SkinHandler struct {
	*UABaseHandler
}

func NewSkinHandler(actor *UserActor) *SkinHandler {
	h := &SkinHandler{UABaseHandler: NewUABaseHandler(actor, "SkinHandler")}
	h.ChildHandler = h

	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CardDressSkinReq), h.DressSkinReq)

	return h
}

// Init 初始化模块数据
func (h *SkinHandler) Init() error {
	// 初始化
	h.actor.Data.SkinData = &cmd.PSkinData{
		Createtime: time.Now().Unix(),
		Skins:      make(map[int32]*cmd.CardSkinData),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debugf("init card skin data success. player: %s", h.actor.ID())
	return nil
}

func (h *SkinHandler) EnterGame() error {
	return nil
}

func (h *SkinHandler) DailyRefresh() error {
	return nil
}

func (h *SkinHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PSkinData); ok {
		h.actor.Data.SkinData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *SkinHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCardSkin(h.actor.ID()), h.actor.Data.SkinData
}

func (h *SkinHandler) buildSkinList(cardId int32) []*cmd.PCommonSkinInfo {
	data := h.actor.GetCardSkinData()
	skinData, ok := data.Skins[cardId]
	if !ok {
		return nil
	}
	return skinData.Skins
}

// 处理卡牌红点已读
func (h *SkinHandler) handleRedPoint(commonData *clidto.Comdata, param []int64) error {
	b := false
	skinData := h.actor.GetCardSkinData()
	for _, id := range param {
		// 皮肤配置
		cfg := excel.GetSkinMgr().GetById(int32(id))
		if cfg == nil {
			continue
		}

		// 红点
		skin, ok := skinData.Skins[cfg.HeroId]
		if !ok {
			continue
		}
		for _, v := range skin.Skins {
			if v.SkinId == int32(id) {
				v.IsNew = false
			}
		}

		// 更新comdata
		card, err := h.actor.CardHandler.GetCard(uint32(cfg.HeroId))
		if err != nil {
			continue
		}
		commonData.Data.Card = append(commonData.Data.Card, h.actor.CardHandler.ToClientData(card))
		b = true
	}

	if b {
		if err := h.SaveDB(); err != nil {
			return err
		}
	}

	return nil
}

func (h *SkinHandler) DressSkinReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_CardDressSkinReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	card, err := h.actor.CardHandler.GetCard(uint32(req.CardId))
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_CardNotExist)
	}

	if !h.IsExistSkin(req.CardId, req.SkinId) {
		return nil, fmt.Errorf("skin data is empty %d", req.CardId), int32(cmd.ErrorCode_CardSkinNotExist)
	}

	// 处理逻辑
	card.SkinId = uint32(req.SkinId)
	err = h.actor.CardHandler.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.SkinDress{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_SkinDress, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		CardId:         req.CardId,
	//		SkinId:         req.SkinId,
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.SkinDress{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			CardId:            req.CardId,
			SkinId:            req.SkinId,
		}
		taptap.WriteDataLog(taptap.LogType_SkinDress, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &cmd.LS2C_CardDressSkinRes{Card: h.actor.CardHandler.ToClientData(card)}, nil, 0
}

func (h *SkinHandler) IsExistSkin(cardId, skinId int32) bool {
	skinData, ok := h.actor.GetCardSkinData().Skins[cardId]
	if !ok {
		return false
	}

	for _, skin := range skinData.Skins {
		if skin.SkinId == skinId {
			return true
		}
	}

	return false
}

func (h *SkinHandler) tryInitSkin(cardId int32) (*cmd.CardSkinData, error) {
	skinData, ok := h.actor.GetCardSkinData().Skins[cardId]
	if !ok {
		skinData = &cmd.CardSkinData{
			CardId: cardId,
			Skins:  make([]*cmd.PCommonSkinInfo, 0),
		}
		cfg := excel.GetBeastarMgr().GetById(cardId)
		if cfg == nil {
			return nil, fmt.Errorf("card not exist %d", cardId)
		}
		skinData.Skins = append(skinData.Skins, &cmd.PCommonSkinInfo{
			SkinId:     cfg.Skin0,
			IsNew:      false,
			CreateTime: time.Now().Unix(),
		}) // 默认皮肤
		h.actor.GetCardSkinData().Skins[cardId] = skinData
		if err := h.SaveDB(); err != nil {
			return nil, err
		}
	}

	return skinData, nil
}

func (h *SkinHandler) AddCardSkin(itemCfg *excel.ItemCfg, addNum uint32, commonData *clidto.Comdata) (*cmd.DropChange, error) {
	if addNum <= 0 {
		return nil, fmt.Errorf("illegal param")
	}

	skinId := itemCfg.GetSystemId()
	skinCfg := excel.GetSkinMgr().GetById(skinId)
	if skinCfg == nil {
		return nil, fmt.Errorf("skin cfg not found")
	}

	var (
		dropChange = &cmd.DropChange{}
	)

	cardId := skinCfg.GetHeroId()
	exist := h.IsExistSkin(cardId, skinId)

	// 发皮肤
	if !exist {
		skinData, err := h.tryInitSkin(cardId)
		if err != nil {
			return nil, err
		}

		skinData.Skins = append(skinData.Skins, &cmd.PCommonSkinInfo{
			SkinId:     skinId,
			IsNew:      true,
			CreateTime: time.Now().Unix(),
		})
		if err = h.SaveDB(); err != nil {
			return nil, err
		}

		// 更新comdata
		card, err := h.actor.CardHandler.GetCard(uint32(cardId))
		if err == nil {
			commonData.Data.Card = append(commonData.Data.Card, h.actor.CardHandler.ToClientData(card))
		}

		// 埋点log
		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.SkinDress{
		//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_SkinCreate, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		CardId:         cardId,
		//		SkinId:         skinId,
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.SkinDress{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				CardId:            cardId,
				SkinId:            skinId,
			}
			taptap.WriteDataLog(taptap.LogType_SkinCreate, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})

		// 任务
		errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_SKIN_ADD, []int32{TASK_TYPE_404, TASK_TYPE_406}, map[string]interface{}{
			"skin_id": skinId,
			"rarity":  0, // fixme
		}))
		if errx != nil {
			h.Error(errx)
		}

		dropChange.Items = append(dropChange.Items, &cmd.ItemReward{
			ItemId: uint32(itemCfg.ItemId),
			Num:    1,
		})

		exist = true
		addNum -= 1
		h.Infof("增加卡牌皮肤 cardId: %v, skinId: %v", cardId, skinId)
	}

	// 转换成其他道具
	if exist && addNum > 0 && skinCfg.Change != nil {
		dropOne, err := GetDropMgr(h.actor).DropList2(map[int32]int32{skinCfg.Change.ItemId: skinCfg.Change.Num}, true, nil, h.actor.comData, common.CR_ADD_CARD_SKIN)
		if err != nil {
			return nil, err
		}
		mergeDropChange(dropChange, dropOne)
		h.Infof("皮肤转材料 cardId: %d, reward: %+v, change:%+v ", cardId, skinCfg.Change, dropOne)
	}

	return dropChange, nil
}

func (h *SkinHandler) getSkinList(cardId int32) []*cmd.PCommonSkinInfo {
	skinData, ok := h.actor.GetCardSkinData().Skins[cardId]
	if !ok {
		return []*cmd.PCommonSkinInfo{}
	}
	return skinData.Skins
}

// getSkinCount
//
//	@Description: 获取玩家的皮肤数量，原皮不统计
//	@receiver h
//	@return int32 皮肤总数量
func (h *SkinHandler) getSkinCount() int32 {
	total := 0
	for _, skin := range h.actor.GetCardSkinData().Skins {
		total += len(skin.Skins) - 1
	}

	return int32(total)
}

// IsExistSkinId
//
//	@Description: 检查是否已拥有指定id的皮肤
//	@receiver h
//	@param skinId 指定皮肤id
//	@return bool 已拥有返回true，否则返回false
func (h *SkinHandler) IsExistSkinId(skinId int32) bool {
	for _, skinData := range h.actor.GetCardSkinData().Skins {
		for _, skin := range skinData.Skins {
			if skin.SkinId == skinId {
				return true
			}
		}
	}

	return false
}

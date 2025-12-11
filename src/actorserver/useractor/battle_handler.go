package useractor

import (
	"context"
	"fmt"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/musae/framework/baseconf"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"

	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type BattleHandler struct {
	*UABaseHandler
}

func NewBattleHandler(actor *UserActor) *BattleHandler {
	h := &BattleHandler{UABaseHandler: NewUABaseHandler(actor, "BattleHandler")}
	h.ChildHandler = h

	// actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_StartBattleEventReq), h.BattleStartReq)      // 战斗开始
	// actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_LevelBattleEventReq), h.BattleEndReq) // 战斗事件

	return h
}

func (h *BattleHandler) Init() error {
	return nil
}

func (h *BattleHandler) EnterGame() error {
	return nil
}

func (h *BattleHandler) DailyRefresh() error {
	return nil
}

func (h *BattleHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *BattleHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

func (h *BattleHandler) BattleStartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
		// errCode pb.ErrorCode
		// uid     string
		rsp *pb.LS2C_StartBattleEventRes
	)

	var req pb.C2LS_StartBattleEventReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *BattleHandler) LevelBattleEventReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err     error
		errCode pb.ErrorCode = pb.ErrorCode_Success
		// uid     string
		rsp *pb.LS2C_LevelBattleEventRes
	)

	var req pb.C2LS_LevelBattleEventReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	return rsp, nil, int32(errCode)
}

func (h *BattleHandler) CheckBattle(
	battleId uint64, battleRandomSeed uint32, battleResult pb.BattleResult,
	selfBattleTeam *pb.BattleTeam, battleEventId int32, battleFrameData []*pb.FrameCommand,
	versionData *pb.CheckBattleVersionData) (*pb.CheckBattleRes, error, pb.ErrorCode) {

	reqMsg := &pb.CheckUp{
		VersionData: versionData,
		CheckBattleReq: &pb.CheckBattleReq{
			BattleId:         battleId,
			BattleRandomSeed: battleRandomSeed,
			BattleResult:     battleResult,
			// EventId:          battleEventId,
			SelfTeam:        selfBattleTeam,
			BattleEventId:   battleEventId,
			BattleFrameData: battleFrameData, /*&pb.BattleFrameData{
				CommandSetDestination:      make([]*pb.CommandSetDestination, 0),
				CommandSetTarget:           make([]*pb.CommandSetTarget, 0),
				CommandTriggerGroundObject: make([]*pb.CommandTriggerGroundObject, 0),
				CommandUseComboSkill:       make([]*pb.CommandUseComboSkill, 0),
				CommandUseItem:             make([]*pb.CommandUseItem, 0),
				CommandUseSkill:            make([]*pb.CommandUseSkill, 0),
			},*/
		},
		// Name: "test-battle",
	}

	if baseconf.GetBaseConf().OpenCheckBattle == 0 {
		// 关闭战斗校验
		logger.Debugf("战斗校验未开启：uid:%s, %v", h.actor.GetUID(), battleId)
		return nil, nil, pb.ErrorCode_Success
	}

	logger.Debugf("开始战斗校验：uid:%s, %v", h.actor.GetUID(), battleId)

	out, err := h.actor.Srv.SvcInvoke(global.BATTLE_SVC, h.actor.GetUID(), h.actor.roleId, h.actor.ID(), reqMsg)
	if err != nil {
		h.Errorf(err.Error())
		return nil, err, pb.ErrorCode_RpcInvokeError
	}

	protoMsg, err := base.UnPackProtoMsg(out)
	if err != nil {
		h.Errorf(err.Error())
		return nil, err, pb.ErrorCode_UnrealizedTypeError
	}
	logger.Debugf("protoMsg：%v", protoMsg.Str())

	checkResp := &pb.CheckDown{}
	err = protoMsg.UnmarshalData(checkResp)
	if err != nil {
		h.Errorf(err.Error())
		return nil, err, pb.ErrorCode_UnrealizedTypeError
	}
	logger.Debugf("校验结果：%v", checkResp)

	if checkResp != nil && checkResp.CheckBattleRes.CheckBattleResult == pb.CheckBattleResult_CBR_version_fail {
		return nil, fmt.Errorf("校验失败, 版本不匹配, %v", checkResp), pb.ErrorCode_VersionLimit
	}

	if checkResp == nil || checkResp.CheckBattleRes.CheckBattleResult != pb.CheckBattleResult_CBR_success {
		return nil, fmt.Errorf("校验失败, %v", checkResp), pb.ErrorCode_CheckBattle_fail
	}

	return checkResp.CheckBattleRes, nil, pb.ErrorCode_Success
}

func (h *BattleHandler) buildSelfBattleCards(troopType pb.CardTroopType, troopId int32, playerLevelData *pb.PlayerLevelData) *pb.BattleTeam {
	var (
		battleTeam = &pb.BattleTeam{
			CardList: make([]*pb.BattleCard, 0),
			FoodList: make([]*pb.KeyValueItem, 0),
		}
		// battleCards = make([]*pb.BattleCard, 0)
		// foodItems   = make([]*pb.KeyValueItem, 0)
	)

	troopInfo, err := h.actor.TroopHandler.getTroopInfo(int32(troopType), troopId)
	if err != nil || len(troopInfo.Card) <= 0 {
		logger.Debugf("获取组队信息异常, err:%v, %v", err, troopInfo)
		return battleTeam
	}

	cardIds := make([]int32, len(troopInfo.Card), len(troopInfo.Card))
	for pos, cardId := range troopInfo.Card {
		cardIds[pos] = cardId
	}

	cardList := h.buildBattleCards(cardIds, playerLevelData)
	battleTeam.CardList = cardList

	foodList := h.buildBattleFoods(troopType, playerLevelData)
	battleTeam.FoodList = foodList

	return battleTeam
}

func (h *BattleHandler) buildCampaignCards(campaignTeam *pb.GeneralCampaignTeam, campaignType common.CAMPAIGN_TYPE) *pb.BattleTeam {
	var (
		battleTeam = &pb.BattleTeam{
			CardList: make([]*pb.BattleCard, 0),
			FoodList: make([]*pb.KeyValueItem, 0),
		}
	)

	cardIds := make([]int32, len(campaignTeam.Cards), len(campaignTeam.Cards))
	for pos, cardId := range campaignTeam.Cards {
		cardIds[pos] = cardId
	}

	cardList := h.buildBattleCards(cardIds, nil)
	battleTeam.CardList = cardList

	var troopType pb.CardTroopType
	switch campaignType {
	case common.CAMPAIGN_TYPE_97, common.CAMPAIGN_TYPE_98: // 队伍类型4
		troopType = pb.CardTroopType_CardTroopType_Resource_Campaign
	case common.CAMPAIGN_TYPE_100: // 队伍类型3
		troopType = pb.CardTroopType_CardTroopType_Electric_Campaign
	default:
		h.Errorf("未支持的关卡类型, campaignType=%d", campaignType)
	}

	// 食物列表
	foodList := h.buildBattleFoods(troopType, nil)
	battleTeam.FoodList = foodList

	return battleTeam
}

func (h *BattleHandler) buildBattleCards(cardIds []int32, playerLevelData *pb.PlayerLevelData) []*pb.BattleCard {
	var (
		cardList = make([]*pb.BattleCard, 0)
	)

	for pos, cardId := range cardIds {
		if cardId == 0 {
			continue
		}

		card, err := h.actor.CardHandler.GetCard(uint32(cardId))
		if err != nil {
			logger.Debugf("获取卡牌信息异常, err:%v, %v", err, card)
			return cardList
		}

		var cardHp uint32 = 0
		var cardEner uint32 = 0
		if playerLevelData == nil {
			// 满血进入
			cardHp = card.OldMaxHp
		} else {
			// 继承血量
			for _, each := range playerLevelData.BattleCards {
				if int32(each.CardId) == cardId {
					cardHp = each.CardHp
					cardEner = each.CardEner
					break
				}
			}
		}

		if cardHp <= 0 {
			logger.Debugf("角色死亡, 不传到校验服, %v", card)
			continue
		}

		equips, err := h.actor.EquipHandler.GetEquipList(cardId)
		if err != nil {
			logger.Debugf("获取装备信息异常, err:%v, %v", err, card)
			return cardList
		}

		bCard := &pb.BattleCard{
			Pos:      int32(pos),
			CardInfo: h.actor.CardHandler.ToClientData(card),
			CardHp:   cardHp,
			CardEner: cardEner,
			Equips:   equips,
		}
		cardList = append(cardList, bCard)
	}

	return cardList
}

func (h *BattleHandler) buildBattleFoods(troopType pb.CardTroopType, playerLevelData *pb.PlayerLevelData) []*pb.KeyValueItem {
	var (
		foodList = make([]*pb.KeyValueItem, 0)
	)

	// 食物列表
	foodIds := h.actor.TroopHandler.GetTroopFoodLog(int32(troopType))

	for _, foodId := range foodIds {
		// 背包中的数量
		item := h.actor.BagHandler.GetItemValueById(foodId)
		if item == nil || item.Key == 0 {
			continue
		}
		// 要是UseFoods有记录，就取useFoods 里的数量
		if troopType == pb.CardTroopType_CardTroopType_Normal {
			markItemNum := GetMarkItemNum(playerLevelData, foodId)
			item.Value = utils.Min(item.Value, markItemNum)
			h.Debug("传入战斗校验的食物数量:", foodId, item.Value)
		}
		foodList = append(foodList, item)
	}

	return foodList
}

func GetMarkItemNum(playerLevelData *pb.PlayerLevelData, foodId int32) int32 {
	maxFoodNum := excel.GetConfigMgr().GetCfg().FOOD_BATTLEUSE_LIMIT
	if playerLevelData != nil {
		for _, v := range playerLevelData.UseFoods {
			if v.GetKey() == foodId {
				return maxFoodNum - v.GetValue()
			}
		}
	}
	return maxFoodNum
}

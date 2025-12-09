package useractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	dapr "github.com/dapr/go-sdk/client"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	svc "gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"gitlab.musadisca-games.com/wangxw/musae/framework/wordfilter"

	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/state"

	"github.com/xuri/excelize/v2"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/idipserver/logic"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"google.golang.org/protobuf/proto"
)

type GmHandler struct {
	*UABaseHandler
	Cmds map[string]CmdLogicFunc
}

type CmdLogicFunc = func([]string, *clidto.Comdata) error

func NewGmHandler(actor *UserActor) *GmHandler {
	h := &GmHandler{
		UABaseHandler: NewUABaseHandler(actor, "GmHandler"),
		Cmds:          make(map[string]CmdLogicFunc),
	}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_UseGameCommandReq), h.UseGameCommandReq) // gm指令处理

	//
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_ReceiveGMAddResReq), h.GMAddRes)             // GM-添加道具
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_ReceiveGMCostResReq), h.GMCostRes)           // GM-扣除道具
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_ReceiveGMAddGiftReq), h.GMAddGift)           // GM-获取礼包道具
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_ReceiveGMAddGiftCodeReq), h.GMAddGiftCode)   // GM-获取礼包道具
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_ReceiveGMAddMailReq), h.GMAddMail)           // GM-添加个人邮件
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_GetUserInfo), h.GMGetUserInfo)               // GM-获取个人信息
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_GmExecuteReq), h.GMExecute)                  // GM-执行个人gm
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2AS_S2SSaveOfflineDataReq), h.GMSaveOfflineData) // GM-保存离线数据

	// 注册gm指令处理方法
	// h.RegisterCmdHandler(common.GM_ACTOR_SHOW, h.GMShowActor)
	h.RegisterCmdHandler(common.GM_ACTOR_DEL, h.GMDelActor)
	h.RegisterCmdHandler(common.GM_ADD_ITEM, h.GMAddItem)
	h.RegisterCmdHandler(common.GM_CLEAN_ITEM, h.GMCleanItem)
	h.RegisterCmdHandler(common.GM_ADD_ITEM_ALL, h.GMAddItemAll)
	h.RegisterCmdHandler(common.GM_ADD_ITEM_BY_TYPE, h.GMAddItemByType)
	h.RegisterCmdHandler(common.GM_DEL_ITEM_BY_ID, h.GMDelItemById)
	h.RegisterCmdHandler(common.GM_ADD_CARD_EXP, h.GmAddCardExp)
	h.RegisterCmdHandler(common.GM_ADD_FAVORITE_EXP, h.GmAddFavoriteExp)
	h.RegisterCmdHandler(common.GM_TEST_CARD, h.GmTestCard)
	h.RegisterCmdHandler(common.GM_KICKOUT, h.GmKickout)
	h.RegisterCmdHandler(common.GM_BANNED, h.GmBanned)
	h.RegisterCmdHandler(common.GM_SET_CARD_STRENGTH, h.GmSetCardStrength)
	h.RegisterCmdHandler(common.GM_TEST_MAIL, h.GmTestMail)
	h.RegisterCmdHandler(common.GM_DEL_MONEY, h.GmDelMoney)
	h.RegisterCmdHandler(common.GM_DEL_STAMINA, h.GmDelStamina)
	h.RegisterCmdHandler(common.GM_DIRECT_LEVEL_UP, h.GMDirectLevelUp)
	h.RegisterCmdHandler(common.GM_RESET_LEVEL, h.GmResetLevel)
	h.RegisterCmdHandler(common.GM_RESET_CARD_POOL_LOG, h.GmResetCardPoolLog)
	h.RegisterCmdHandler(common.GM_WEAR_EQUIP, h.GmWearEquip)
	h.RegisterCmdHandler(common.GM_ADD_PLAYER_EXP, h.GmAddPlayerExp)
	h.RegisterCmdHandler(common.GM_DIRECT_COMPLETE_OBJECT, h.GmDirectCompleteObject)
	h.RegisterCmdHandler(common.GM_DIRECT_COMPLETE_QUEST, h.GmDirectCompleteQuest)
	h.RegisterCmdHandler(common.GM_TEST_SIGN, h.GmTestSign)
	h.RegisterCmdHandler(common.GM_TEST_PROTO, h.GmTestCmd)
	h.RegisterCmdHandler(common.GM_SET_SUPER_CARD, h.GmSetSuperCard)
	h.RegisterCmdHandler(common.GM_SAVE_STORY_FLAG, h.GmSaveStoryFlag)
	h.RegisterCmdHandler(common.GM_LEVEL_FINISH, h.GmLevelFinish)
	h.RegisterCmdHandler(common.GM_TEST_UGC, h.GmTestUgc)
	h.RegisterCmdHandler(common.GM_TEST_SENSITIVE, h.GmTestSensitive)
	h.RegisterCmdHandler(common.GM_TEST_Battle_chapter, h.GmTestBattleChapter)
	h.RegisterCmdHandler(common.GM_CHECKBATTLE_RELOAD_EXCEL, h.GmTestCheckBattleReloadExcel)
	h.RegisterCmdHandler(common.GM_TEST_GEN_CODE, h.GmTestGenCode)
	h.RegisterCmdHandler(common.GM_TEST_USE_CODE, h.GMTestUseCode)
	h.RegisterCmdHandler(common.GM_TEST_DROP, h.GmTestDrop)
	h.RegisterCmdHandler(common.GM_TEST_GUID, h.GmTestGUID)
	h.RegisterCmdHandler(common.GM_TEST_DB, h.GmTestDB)
	h.RegisterCmdHandler(common.GM_RESET_DUTY_TASK, h.GMResetDutyTask)
	h.RegisterCmdHandler(common.GM_DIRECT_COMPLETE_DUTY_TASK, h.GMDirectCompleteDutyTask)
	h.RegisterCmdHandler(common.GM_ERR_CODE, h.GMTestErrCode)
	// h.RegisterCmdHandler(common.GM_TEST_BOOK, h.GMTestBook)
	h.RegisterCmdHandler(common.GM_TEST_ACHIEVE, h.GMTestAchieve)
	h.RegisterCmdHandler(common.GM_TEST_PVP_ROOM, h.GMTestRoom)
	h.RegisterCmdHandler(common.GM_CLOSE_BATTKE_CHECK, h.GMCloseBattleCheck)
	h.RegisterCmdHandler(common.GM_TEST_RECOMMEND, h.GMTestRecommend)
	h.RegisterCmdHandler(common.GM_ADD_CARDS_RELATION, h.AddCardsRelation)
	h.RegisterCmdHandler(common.GM_TEST_CampDouble, h.CampDouble)
	h.RegisterCmdHandler(common.GM_TEST_Card_Broad, h.CardBroad)
	h.RegisterCmdHandler(common.GM_TEST_Test_Cfg_Hot, h.TestCfgHot)

	return h
}

func (h *GmHandler) Init() error {
	return nil
}

func (h *GmHandler) EnterGame() error {
	return nil
}

func (h *GmHandler) DailyRefresh() error {
	return nil
}

func (h *GmHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *GmHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

func (h *GmHandler) RegisterCmdHandler(name string, handler CmdLogicFunc) {
	if _, ok := h.Cmds[name]; !ok {
		h.Cmds[name] = handler
		h.Debugf("register cmd: %s", name)
	} else if ok {
		h.Errorf("Duplicate cmd are registered: %s", name)
	}
}

func (h *GmHandler) GMExecute(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.S2AS_ExcuteGMReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	handler := h.Cmds[req.CmdName]
	if handler == nil {
		h.Debugf("GM指令不存在：%s", req.CmdName)
		return nil, errors.New("GM指令不存在"), int32(cmd.ErrorCode_ParamError)
	}
	args := strings.Split(req.OptVal, " ")
	err = handler(args, h.actor.comData)
	if err != nil {
		h.Errorf("GM handle err: %v", err)
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	rsp := &base.ProtoMsg{}
	return rsp, nil, 0
}

// 保存离线数据
func (h *GmHandler) GMSaveOfflineData(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.S2SSaveOfflineDataReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// h.actor.OfflineDataHandler.saveOfflineData(req)

	rsp := &base.ProtoMsg{}
	return rsp, nil, 0
}

func (h *GmHandler) GMGetUserInfo(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	account := h.actor.Account
	var coin1 int
	coins := make([]*logic.CommonCoin, 0)
	roleInfo := h.actor.Data.Base
	for k, v := range h.actor.Data.Currency.GetCurrencyx() {
		if k == common.CURRENCY_ITEM_ID_2006 {
			coin1 = int(v.Value)
		}
		coins = append(coins, &logic.CommonCoin{
			CoinName:  strconv.Itoa(int(v.Key)),
			CoinValue: int(v.Value),
		})
	}
	items := make([]*logic.CommonItem, 0)
	for _, v := range h.actor.Data.ItemData.GetItems() {
		items = append(items, &logic.CommonItem{
			ItemId:    strconv.Itoa(int(v.BaseId)),
			ItemCount: int32(v.ItemNum),
		})
	}
	player := account.PlayerList.Players[1]
	rsp := &logic.CommonUser{
		SvrId:             0,
		BornsvrId:         0,
		UserId:            int(player.Id),
		UserName:          roleInfo.Common.RoleName,
		OpenId:            account.Account.OpenId,
		Plat:              account.CliDeviceInfo.Os,
		UserLevel:         int32(roleInfo.Common.RoleLevel),
		Vip:               0, // todo
		RechargeSum:       account.Recharge.SaveMoney,
		Currency:          coin1,
		Coins:             coins,
		Items:             items,
		MonthcardLeftdays: "", // todo
		LastLoginTime:     roleInfo.LastSaveTimestamp,
		LastLoginIp:       account.CliDeviceInfo.Ip,
		CreateTime:        account.CreateTs,
		UnlockTime:        account.Account.BannedTs,
		UnsilenceTime:     0,     // todo
		IsShield:          false, // todo
	}
	data, err := json.Marshal(rsp)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	return &cmd.S2AS_GetUserInfoRes{User: string(data)}, nil, 0
}

func (h *GmHandler) GMAddMail(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.S2S_SendGMAddUserMailReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	if err = h.actor.MailHandler.GMTAddUserMail(req, h.actor.comData); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	return req, nil, 0
}

func (h *GmHandler) GMAddGiftCode(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.C2LS_UseGiftCodeReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	err = h.actor.GiftHandler.Redeem(req.Code, h.actor.comData)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InvalidParam)
	}
	return &cmd.C2LS_UseGiftCodeReq{}, nil, 0
}

func (h *GmHandler) GMAddGift(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	req := &cmd.S2SReceiveGMAddGiftReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	cfg := excel.GetPackageMgr().GetById(req.PackageId)
	if cfg == nil {
		return nil, errors.New("配置不存在"), int32(cmd.ErrorCode_ConfigError)
	}

	_, err = GetDropMgr(h.actor).DropList2(cfg.Itemcontain, true, nil, h.actor.comData, common.CR_GM)

	return &cmd.S2SReceiveGMAddResRsp{}, nil, 0
}

func (h *GmHandler) GMCostRes(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.S2SReceiveGMCostResReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	costItem := make(map[int32]int32)
	for item, itemNum := range req.Items {
		costItem[item] += itemNum
	}
	for coin, coinNum := range req.Coins {
		costItem[coin] += coinNum
	}
	err = GetConsumeMgr(h.actor).ConsumeList(costItem, h.actor.comData, common.CR_GM)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	return &cmd.S2SReceiveGMAddResRsp{}, nil, 0
}

func (h *GmHandler) GMAddRes(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.S2SReceiveGMAddResReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	costItem := make(map[uint32]uint32)
	for item, itemNum := range req.Items {
		costItem[uint32(item)] += uint32(itemNum)
	}
	for coin, coinNum := range req.Coins {
		costItem[uint32(coin)] += uint32(coinNum)
	}
	_, err = GetDropMgr(h.actor).DropList(costItem, true, nil, h.actor.comData, common.CR_GM)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	return &cmd.S2SReceiveGMAddResRsp{}, nil, 0

}

func (h *GmHandler) UseGameCommandReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.C2LS_UseGameCommandReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	h.Debugf("req: %+v", &req)

	handler := h.Cmds[req.Cmd]
	if handler == nil {
		h.Debugf("GM指令不存在：%s", req.Cmd)
		return nil, errors.New("GM指令不存在"), int32(cmd.ErrorCode_ParamError)
	}

	err = handler(req.Param, h.actor.comData)
	if err != nil {
		h.Errorf("GM handle err: %v", err)
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	rsp := &cmd.LS2C_UseGameCommandRes{CommonData: h.actor.comData.FixDownComData()}
	return rsp, nil, 0
}

func (h *GmHandler) GmAddCardExp(param []string, commonData *clidto.Comdata) error {
	cardId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	exp, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	card, err := h.actor.CardHandler.GetCard(uint32(cardId))
	if err != nil {
		return err
	}
	_, err = h.actor.CardHandler.AddExp(card, int32(exp), false)
	err = h.actor.CardHandler.SaveDB()
	if err != nil {
		return err
	}

	commonData.Data.Card = append(commonData.Data.Card, h.actor.CardHandler.ToClientData(card))
	return nil
}

func (h *GmHandler) GmAddFavoriteExp(param []string, commonData *clidto.Comdata) error {
	cardId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	exp, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	card, errCode := h.actor.CardHandler.AddFavoriteExpById(int32(cardId), uint32(exp))
	if errCode != cmd.ErrorCode_Success {
		return fmt.Errorf("add favorite exp failed, code: %d", int32(errCode))
	}
	commonData.Data.Card = append(commonData.Data.Card, card)
	return nil
}

func (h *GmHandler) GmTestCard(param []string, commonData *clidto.Comdata) error {
	poolId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	total, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	var typ int
	if len(param) > 2 {
		typ, err = strconv.Atoi(param[2])
		if err != nil {
			return err
		}
	}
	var (
		result  []int32
		quality []int32
	)

	if typ == 1 {
		// result, quality, _, err = h.actor.CampPoolHandler.handlePoolExtract(int32(poolId), total, commonData)
	} else if typ == 2 {
		result, _, quality, _, err = h.actor.PoolHandler.handlePoolExtract(int32(poolId), total, commonData)
	} else if typ == 3 {
		result, _, quality, _, err = h.actor.PoolHandler.handleNewbiePoolExtract(int32(poolId), total, commonData)
	}

	if err != nil {
		return err
	}
	// 处理结果统计
	resultMap := make(map[int32]int32)
	qualityMap := make(map[int32]int32)
	for _, v := range result {
		resultMap[v] += 1
	}
	for _, v := range quality {
		qualityMap[v] += 1
	}
	str := fmt.Sprintf("player %s, 卡池Id: %d, 抽卡次数: %d,\n抽卡结果 %+v \n品质分布 %+v \n抽卡统计 %+v \n品质统计 %+v", h.actor.ID(), poolId, total, result, quality, resultMap, qualityMap)
	myUtils.SaveLogToFile("./log/plog/testcard.txt", str)
	return nil
}

func (h *GmHandler) GmSetCardStrength(param []string, commonData *clidto.Comdata) error {
	cardId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	curValue, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	err, card := h.actor.CardHandler.SetCardHp(int32(cardId), int32(curValue))
	if err != nil {
		return err
	}
	commonData.Data.Card = append(commonData.Data.Card, card)
	return nil
}

func (h *GmHandler) GmTestMail(param []string, commonData *clidto.Comdata) error {
	mailId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	count, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		err = h.actor.MailHandler.AddUserMail(int32(mailId), nil, commonData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *GmHandler) GmDelMoney(param []string, commonData *clidto.Comdata) error {
	typ, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	value, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	return h.actor.CurrencyHandler.SubCurrency(int32(typ), int64(value), commonData, common.CR_GM)
}

func (h *GmHandler) GmDelStamina(param []string, commonData *clidto.Comdata) error {
	value, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	costVal := int32(value)

	stamina := h.actor.PlayerLevelHandler.GetPlayerStamina()
	if stamina.Value < costVal { // 输入扣除的值超过当前体力, 则全部扣完
		costVal = stamina.Value
	}

	return h.actor.PlayerLevelHandler.SubStamina(costVal, commonData, common.CR_GM)
}

func (h *GmHandler) GmResetLevel(param []string, commonData *clidto.Comdata) error {
	return h.actor.LoginHandler.ResetLevelByGM(commonData)
}

func (h *GmHandler) GmResetCardPoolLog(param []string, commonData *clidto.Comdata) error {
	return h.actor.PoolHandler.ResetCardLogByGM()
}

func (h *GmHandler) GmKickout(param []string, commonData *clidto.Comdata) error {
	var sec int64
	if len(param) > 0 {
		value, err := strconv.ParseInt(param[0], 10, 64)
		if err == nil {
			sec = value
		}
	}
	if err := h.actor.Srv.KickOutUser(h.actor.uid); err != nil {
		return err
	}
	if sec > 0 {
		return h.actor.AccountHandler.Banned("踢人临时封禁", sec)
	}
	return nil
}

func (h *GmHandler) GmBanned(param []string, commonData *clidto.Comdata) error {
	value, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return err
	}
	if err = h.actor.Srv.KickOutUser(h.actor.uid); err != nil {
		return err
	}
	return h.actor.AccountHandler.Banned(param[1], value)
}

func (h *GmHandler) GmWearEquip(param []string, commonData *clidto.Comdata) error {
	cardId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	equipConfigId, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	return h.actor.EquipHandler.wearEquipByGM(int32(cardId), int32(equipConfigId), commonData)
}

func (h *GmHandler) GmAddPlayerExp(param []string, commonData *clidto.Comdata) error {
	value, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	_, err = h.actor.LoginHandler.AddRoleExp(uint64(value), commonData)
	return err
}

func (h *GmHandler) GmSetSuperCard(param []string, commonData *clidto.Comdata) error {
	cardId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	var typ int
	if len(param) > 1 {
		typ, err = strconv.Atoi(param[1])
		if err != nil {
			return err
		}
	}
	return h.actor.CardHandler.SetSuperCardByGM(cardId, typ, commonData)
}

func (h *GmHandler) GmDirectCompleteObject(param []string, commonData *clidto.Comdata) error {
	objId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	f := 0
	if len(param) > 1 {
		f, err = strconv.Atoi(param[1])
		if err != nil {
			return err
		}
	}
	return h.actor.QuestHandler.directCompleteObjectByGM(int32(objId), int32(f), commonData)
}

func (h *GmHandler) GmDirectCompleteQuest(param []string, commonData *clidto.Comdata) error {
	questId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	return h.actor.QuestHandler.directCompleteQuestByGM(int32(questId), commonData)
}

func (h *GmHandler) GmTestSign(param []string, commonData *clidto.Comdata) error {
	groupId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	p, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	return h.actor.SignHandler.DaySignByGM(int32(groupId), int32(p), commonData)
}

// 调试协议GM
func (h *GmHandler) GmTestCmd(param []string, comdata *clidto.Comdata) error {
	if len(param) < 1 {
		return fmt.Errorf("invalid param")
	}

	msgId, err := strconv.Atoi(param[0]) // <<<=== messageId
	if err != nil {
		return err
	}

	// 构建协议数据
	req := &cmd.C2MS_MailReceiveReq{} // <<<=== 协议体，根据要调试的协议修改

	// 填充参数(如果协议需要参数的话)
	typ, err := strconv.Atoi(param[1]) // <<<=== 参数列表，根据不同的协议填充参数
	mailId, err := strconv.Atoi(param[2])
	if err != nil {
		return err
	}
	req.ReceiveType = int32(typ)
	req.MailIds = int64(mailId)

	// 序列化
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	// 调用
	handler, ok := h.actor.MsgFunc[int32(msgId)]
	if !ok {
		return fmt.Errorf("invalid msgId")
	}
	message, err, code := handler(context.Background(), &base.ProtoMsg{
		AppId:   h.actor.Srv.AppId,
		MsgId:   int32(msgId),
		UserId:  h.actor.uid,
		RoleId:  h.actor.roleId,
		UAID:    "",
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: utils.GenIntUUID(),
	})
	logger.Debugf("调用结果 msg:%+v, err:%v, code:%d", message, err, code)
	return nil
}

func (h *GmHandler) GmSaveStoryFlag(param []string, commonData *clidto.Comdata) error {
	var (
		err error
	)

	if len(param) < 1 {
		return fmt.Errorf("invalid param")
	}
	flag := param[0]
	val := datahelper.STORY_FLAG_V
	if len(param) >= 2 {
		val, err = strconv.Atoi(param[1])
	}
	err, _ = h.actor.StoryFlagHandler.saveStoryFlagVal(commonData, &cmd.FlagInfo{
		Key: flag,
		Val: int32(val),
	})
	return err
}

func (h *GmHandler) GmLevelFinish(param []string, commonData *clidto.Comdata) error {
	// 第一个参数指定关卡id
	levelId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}

	// 第二个参数指定结束点id
	endId := 0
	if len(param) >= 2 {
		endId, err = strconv.Atoi(param[1])
		if err != nil {
			return err
		}
	}

	// 第三个参数为2, 表示以失败结束关卡
	battleResult := cmd.BattleResult_BattleResult_Winer // 默认胜利
	if len(param) >= 3 {
		_succ, err := strconv.Atoi(param[2])
		if err != nil {
			return err
		}
		if _succ == 1 {
			// 胜利
		} else {
			battleResult = cmd.BattleResult_BattleResult_Loser
		}
	}

	if err, _ = checkIsTravelLevel(int32(levelId)); err == nil {
		// 是间章关卡
		h.actor.TravelLevelHandler.GmExitTravelLevel(int32(levelId))
	} else {
		err, _ = h.actor.ChapterHandler.GmExitLevel(h.actor.ID(), levelId, endId, battleResult)
	}

	return err
}

func (h *GmHandler) GmTestSensitive(param []string, commonData *clidto.Comdata) error {
	ok, str := wordfilter.GetSensitiveWordMgr().FindIn(param[0])
	h.Debugf("屏蔽词校验 result: %v, find: %s", ok, str)
	return nil
}

func (h *GmHandler) GmTestUgc(param []string, commonData *clidto.Comdata) error {
	// 测试指定字符串
	if len(param) > 0 {
		cType, err := strconv.Atoi(param[1])
		if err != nil {
			return err
		}
		_, err = h.test_UGCStringCheck(param[0], int32(cType), true)
		return err
	} else {
		// 测试随机昵称库
		fileHandle, err := excelize.OpenFile("./output/res/Name.xlsx")
		if err != nil {
			return err
		}
		defer func() {
			if err = fileHandle.Close(); err != nil {
				h.Warn(err)
			}
		}()
		// 校验
		invalidNameList := make([]string, 0)
		rowsA, err := fileHandle.GetRows("name_adj")
		if err != nil {
			return err
		}
		rowsB, err := fileHandle.GetRows("name_noun")
		if err != nil {
			return err
		}

		var total int32
		for i, rowA := range rowsA {
			if i == 0 || i == 1 {
				continue
			}
			// 请求接口
			total++
			check, err := h.test_UGCStringCheck(rowA[2], common.CHECK_TYPE_PLAYERNAME, false)
			if err != nil {
				return err
			}
			if !check {
				invalidNameList = append(invalidNameList, rowA[2])
			}
		}
		for j, rowB := range rowsB {
			if j == 0 || j == 1 {
				continue
			}
			// 请求接口
			total++
			check, err := h.test_UGCStringCheck(rowB[2], common.CHECK_TYPE_PLAYERNAME, false)
			if err != nil {
				return err
			}
			if !check {
				invalidNameList = append(invalidNameList, rowB[2])
			}
		}

		// 输出到文件中
		myUtils.SaveLogToFile("./output/res/invalidName.txt", fmt.Sprintf("本次校验总数：%d 个\n非法昵称：%v", total, invalidNameList))
		return nil
	}
}

func (h *GmHandler) GmTestBattleChapter(param []string, commonData *clidto.Comdata) error {
	var operateType = 0

	if len(param) > 0 {
		operateType, _ = strconv.Atoi(param[0])
	}

	if operateType == 1 { // 从battleserver获取json数据
		var Id = "73"
		if len(param) > 1 {
			Id = param[1]
		}

		err := h.testExcelBattleEventReq(Id)
		if err != nil {
			return err
		}
	} else {
		err := h.testCheckBattle()
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *GmHandler) testCheckBattle() error {
	SelfTeam := &cmd.BattleTeam{
		CardList: []*cmd.BattleCard{
			{
				Pos: 1,
				CardInfo: &cmd.PClientCardInfo{Common: &cmd.PCommonCardInfo{
					CardId:            2,
					CardLevel:         5,
					CardExp:           45,
					Hp:                966,
					CreateTimestamp:   1683598892,
					SkillId:           []uint32{21001, 22001, 23001},
					SkinId:            2,
					BreakthroughLevel: 0,
					AwakenCfgId:       60200,
					CharacterLevel:    0,
					EquipId:           []uint64{0, 0, 0},
					FavoriteLevel:     0,
					FavoriteExp:       0,
					Skins: []*cmd.PCommonSkinInfo{
						{
							SkinId:     2,
							IsNew:      false,
							CreateTime: 1683598892,
						},
					},
					IsNew:          false,
					Character:      []int32{1},
					CurCharacter:   1,
					FavoriteReward: nil,
				}},
				CardHp:   924,
				CardEner: 0,
				Equips:   nil,
			}},
		FoodList: nil,
	}

	_, err, _ := h.actor.BattleHandler.CheckBattle(1, 1, cmd.BattleResult_BattleResult_Winer, SelfTeam, 73, nil, nil)
	if err != nil {
		h.Errorf(err.Error())
		return err
	}

	return nil
}

func (h *GmHandler) GmTestGenCode(param []string, commonData *clidto.Comdata) error {
	contentId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	genNum, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}
	_, err = h.actor.GiftHandler.Generate(int64(contentId), 1, genNum)
	return err
}

func (h *GmHandler) GMTestUseCode(param []string, commonData *clidto.Comdata) error {
	return h.actor.GiftHandler.Redeem(param[0], commonData)
}

func (h *GmHandler) GMDelItemById(param []string, commonData *clidto.Comdata) error {
	delItems := make(map[int32]int32)
	for i := 0; i < len(param); {
		itemId, err := strconv.ParseInt(param[i], 10, 32)
		if err != nil {
			return err
		}
		itemNum, err := strconv.ParseInt(param[i+1], 10, 32)
		if err != nil {
			return err
		}

		delItems[int32(itemId)] = int32(itemNum)

		i += 2
	}

	return GetConsumeMgr(h.actor).ConsumeList(delItems, commonData, common.CR_GM)
}

func (h *GmHandler) GMAddItemAll(param []string, commonData *clidto.Comdata) error {
	addItems := make(map[int32]int32)
	excel.GetItemMgr().Foreach(func(cfg *excel.ItemCfg) bool {
		addItems[cfg.Id] = int32(cfg.NumLimit)
		return true
	}, true)

	_, err := GetDropMgr(h.actor).DropList2(addItems, true, nil, commonData, common.CR_GM)
	return err
}

func (h *GmHandler) GMDirectLevelUp(param []string, commonData *clidto.Comdata) error {
	value, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	return h.actor.LoginHandler.DirectLevelUpByGM(uint32(value), commonData)
}

func (h *GmHandler) GMResetDutyTask(param []string, commonData *clidto.Comdata) error {
	return h.actor.DutyHandler.ResetTaskByGM(commonData)
}

func (h *GmHandler) GMDirectCompleteDutyTask(param []string, commonData *clidto.Comdata) error {
	taskId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	return h.actor.DutyHandler.DirectCompleteTaskByGM(int32(taskId), commonData)
}

func (h *GmHandler) GMTestErrCode(param []string, commonData *clidto.Comdata) error {
	// var (
	//	err        error
	//	errCodeVal = 0
	// )
	// if len(param) > 0 {
	//	errCodeVal, err = strconv.Atoi(param[0])
	//	if err != nil {
	//		return err
	//	}
	// }
	return fmt.Errorf("GM : user-defined error")
}

// GMCleanItem 清空道具数据
func (h *GmHandler) GMCleanItem(param []string, commonData *clidto.Comdata) error {
	h.actor.Data.ItemData.Items = make(map[uint64]*cmd.PCommonItemInfo, 0)
	err := h.actor.BagHandler.SaveDB()
	if err != nil {
		return err
	}
	commonData.Data.Items = make([]*cmd.PCommonItemInfo, 0)
	return nil
}

func (h *GmHandler) GMAddItem(param []string, commonData *clidto.Comdata) error {
	addItems := make(map[uint32]uint32)
	subItems := make(map[int32]int32)
	for i := 0; i < len(param); {
		itemId, err := strconv.ParseInt(param[i], 10, 32)
		if err != nil {
			return err
		}
		itemNum, err := strconv.ParseInt(param[i+1], 10, 32)
		if err != nil {
			return err
		}

		if itemNum < 0 {
			subItems[int32(itemId)] = int32(itemNum * -1) // 负数先转正再强转
		} else {
			addItems[uint32(itemId)] = uint32(itemNum)
		}

		i += 2
	}

	_, err := GetDropMgr(h.actor).DropList(addItems, true, nil, commonData, common.CR_GM)
	h.Debugf("dropMgr.DropList ===>>>addItems:%+v dropList:%+v, err:%+v", addItems, commonData.Data.Items, err)

	err = GetConsumeMgr(h.actor).ConsumeList(subItems, commonData, common.CR_GM)
	h.Debugf("consumeMgr.consumeList ===>>>subItems:%+v dropList:%+v, err:%+v", subItems, commonData.Data.Items, err)
	return err
}

func (h *GmHandler) GMAddItemByType(param []string, commonData *clidto.Comdata) error {
	addItems := make(map[uint32]uint32)
	for i := 0; i < len(param); {
		typeId, err := strconv.ParseInt(param[i], 10, 32)
		if err != nil {
			h.Debugf("gm 命令解析错误, err=%+v", err)
		}
		itemNum, err := strconv.ParseInt(param[i+1], 10, 32)
		if err != nil {
			h.Debugf("gm 命令解析错误, err=%+v", err)
		}

		// 根据typeId查找道具id
		excel.GetItemMgr().Foreach(func(cfg *excel.ItemCfg) bool {
			if typeId == int64(cfg.GetType()) {
				addItems[uint32(cfg.GetId())] = uint32(itemNum)
			}
			return true
		}, true)

		i += 2
	}

	_, err := GetDropMgr(h.actor).DropList(addItems, true, nil, commonData, common.CR_GM)
	return err
}

func (h *GmHandler) GmTestDrop(param []string, commonData *clidto.Comdata) error {
	if len(param) < 2 {
		return fmt.Errorf("param error")
	}
	dropId, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	num, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}

	var builder strings.Builder
	for i := 1; i <= num; i++ {
		itemRewards := datahelper.GetRewardsByDropId(int32(dropId))

		builder.WriteString(fmt.Sprintf("dropId: %d, num: %d, reward: %v \n", dropId, i, datahelper.ConvertItem2ByTpl(itemRewards)))
	}
	myUtils.SaveLogToFile("./log/plog/testdrop.txt", builder.String())
	return nil
}

func (h *GmHandler) test_UGCStringCheck(str string, cType int32, f bool) (bool, error) {
	// if f {
	//	reqMsg := &cmd.C2LS_HeartBeatReq{}
	//	_, err := h.actor.Srv.SvcInvoke(global.IDIP_SVC, h.actor.GetUID(), h.actor.roleId, h.actor.ID(), reqMsg)
	//	if err != nil {
	//		h.Error(err)
	//	}
	// }

	// 校验合法性
	h.Debugf("数据校验: %s %d", str, cType)
	result, err := h.actor.Srv.CheckSensitiveWord(cType, str)
	if err != nil {
		return false, err
	}

	h.Debug("校验结果：", result)
	return result, nil
}

// GmTestDB
//
//	@Description: 测试数据库GM
//	@receiver h
//	@param param 参数1=DB类型（redis/mongo），参数2=读写（read/write），参数3=协程数量，参数4=读写次数
//	@param commonData
//	@return error
func (h *GmHandler) GmTestDB(param []string, commonData *clidto.Comdata) error {
	if len(param) < 4 {
		return fmt.Errorf("param error")
	}
	var (
		isRedis bool
		isRead  bool
	)

	if param[0] == "redis" {
		isRedis = true
	}
	if param[1] == "read" {
		isRead = true
	}
	threadNum, err := strconv.Atoi(param[2])
	if err != nil {
		return err
	}
	opNum, err := strconv.Atoi(param[3])
	if err != nil {
		return err
	}

	delayMap := sync.Map{}
	errMap := sync.Map{}
	var wg sync.WaitGroup
	for i := 1; i <= threadNum; i++ {
		wg.Add(1)
		threading.RunSafe(func() {
			for x := 1; x <= opNum; x++ {
				// 随机10毫秒内延迟执行
				time.Sleep(time.Millisecond * time.Duration(rand.Int31n(10)))
				delay, err := h.operateDB(isRedis, isRead)
				if err != nil {
					v, ok := errMap.LoadOrStore(err, 1)
					if ok {
						errMap.Store(err, v.(int)+1)
					}
				} else {
					v, ok := delayMap.LoadOrStore(delay/10, 1)
					if ok {
						delayMap.Store(delay/10, v.(int)+1)
					}
				}
			}
			wg.Done()
		})
	}
	wg.Wait()

	var errStr strings.Builder
	errMap.Range(func(key, value any) bool {
		errStr.WriteString(fmt.Sprintf("GmTestDB 错误统计===>>> err: %v, count: %v\n", key, value))
		return true
	})

	var delayStr strings.Builder
	delayMap.Range(func(key, value any) bool {
		delayStr.WriteString(fmt.Sprintf("GmTestDB 耗时统计===>>>, delay:%v, count:%v\n", key, value))
		return true
	})
	str := fmt.Sprintf("数据库: %s, 操作: %s, 并发: %d, 次数: %d\n %s %s", param[0], param[1], threadNum, opNum, errStr.String(), delayStr.String())
	myUtils.SaveLogToFile("./log/testdb.txt", str)
	logger.Error(str)
	return nil
}

func (h *GmHandler) operateDB(isRedis, isRead bool) (int64, error) {
	var (
		delay int64
		err   error
	)
	begin := time.Now()

	// 测试redis
	mongoDbType, dbKey, dbMsg := h.actor.CardHandler.DBTable()
	if isRedis {
		if isRead {
			_, err := h.actor.Srv.GetCacheOnlyFromRedis(dbKey, nil, dbMsg)
			if err != nil {
				return delay, err
			}
		} else {
			err = h.actor.Cache2Redis(mongoDbType, h.actor.ID(), dbKey, dbMsg)
			if err != nil {
				return delay, err
			}
		}
	} else {
		// 测试mongo
		if isRead {
			_, err = h.actor.Srv.GetMongo(mongoDbType, dbKey, nil)
		} else {
			kvTable, err := db.BuildKvTable(dbMsg, dbKey)
			if err != nil {
				return delay, err
			}
			m := map[string]*state.KvTable{dbKey: kvTable}
			err = h.actor.Srv.UpsertMongoTableTransaction(service.MongoDbType_MongoGame, nil, m)
		}
	}
	delay = time.Since(begin).Milliseconds()
	return delay, err
}

func (h *GmHandler) GmTestGUID(param []string, commonData *clidto.Comdata) error {
	if len(param) < 2 {
		return fmt.Errorf("param error")
	}
	threadNum, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	idNum, err := strconv.Atoi(param[1])
	if err != nil {
		return err
	}

	startId := h.actor.Srv.GenGUID(guid.GUID_PLAYER)
	guids := sync.Map{}
	var wg sync.WaitGroup
	for i := 1; i <= threadNum; i++ {
		wg.Add(1)
		threading.RunSafe(func() {
			for i := 1; i <= idNum; i++ {
				time.Sleep(time.Millisecond * time.Duration(rand.Int()%30+10))
				id := h.actor.Srv.GenGUID(guid.GUID_PLAYER)
				if id == 0 {
					h.Warn("GUID Test id err")
					break
				}
				guids.Store(id, id)
			}
			wg.Done()
		})
	}
	endId := h.actor.Srv.GenGUID(guid.GUID_PLAYER)
	wg.Wait()
	realCount := 0
	guids.Range(func(key, value any) bool {
		realCount += 1
		return true
	})
	allCount := threadNum * idNum
	h.Warnf("GUID Test,startId:[%d],endId[%d],allCount:[%d],realCount[%d]", startId, endId, allCount, realCount)
	return nil
}

func (h *GmHandler) GMTestAchieve(param []string, commonData *clidto.Comdata) error {

	// cardIds := make([]int32, 0)
	// for _, id := range param {
	//	cardId, _ := strconv.Atoi(id)
	//	cardIds = append(cardIds, int32(cardId))
	//	rarity, _ := GetCardRarityByItemId(int32(cardId))
	//	h.Debug("GM 触发广播时，卡片的稀有度:", rarity)
	// }

	h.actor.UserChatHandler.DeleteFriendChatMessage(179794, 179193)
	return nil
}

func (h *GmHandler) testExcelBattleEventReq(dataId string) error {
	reqMsg := &cmd.CheckUp{
		TestExcelBattleEventReq: &cmd.TestExcelBattleEventReq{
			Id: dataId,
		},
	}

	out, err := h.actor.Srv.SvcInvoke(global.BATTLE_SVC, h.actor.GetUID(), h.actor.roleId, h.actor.ID(), reqMsg)
	if err != nil {
		h.Errorf(err.Error())
		return err
	}

	protoMsg, err := base.UnPackProtoMsg(out)
	if err != nil {
		h.Errorf(err.Error())
		return err
	}
	logger.Debugf("protoMsg：%v", protoMsg.Str())

	checkResp := &cmd.CheckDown{}
	err = protoMsg.UnmarshalData(checkResp)
	if err != nil {
		h.Errorf(err.Error())
		return err
	}
	logger.Debugf("校验结果：%v", checkResp)

	return nil
}

func (h *GmHandler) GMTestRoom(param []string, commonData *clidto.Comdata) error {
	if len(param) < 2 {
		return fmt.Errorf("param error")
	}
	operate, err := strconv.Atoi(param[0])
	if err != nil {
		return err
	}
	roomId := param[1]

	switch operate {
	case 1: // 创建房间
		reqMsg := &cmd.C2LS_CreateRoomReq{
			PlayType: cmd.RoomModel_RoomModel_tug,
			// PlayerUid: h.actor.uid,
		}

		data, err := proto.Marshal(reqMsg)
		if err != nil {
			logger.Debug("proto marshal err:", err)
			return err
		}

		msg := base.ProtoMsg{
			MsgId:  int32(cmd.Protocols_PC2LS_CreateRoomReq),
			AppId:  global.ACTOR_SVC,
			UserId: h.actor.uid,
			RoleId: 0,
			UAID:   h.actor.Srv.UAID(h.actor.ID(), h.actor.roleId),
			Data:   data,
			// GUID:    utils.GenIntUUID(),
			ServerReqIdx: utils.GenIntUUID(),
		}

		resp, err := h.actor.Srv.ActorInvoke(global.RoomActorType, roomId, &msg)
		if err != nil {
			return err
		}

		h.Debugf(resp.Str())
		// _, err = h.actor.Srv.SvcInvoke(global.RoomActorType, strconv.Itoa(roomId), uint64(roomId), strconv.Itoa(roomId), reqMsg)
		// if err != nil {
		//	h.Error(err)
		// }
	case 2: // 进入房间
		reqMsg := &cmd.C2LS_JoinRoomReq{
			// PlayerUid: h.actor.uid,
			RoomId: roomId,
		}

		data, err := proto.Marshal(reqMsg)
		if err != nil {
			logger.Debug("proto marshal err:", err)
			return err
		}

		msg := base.ProtoMsg{
			MsgId:  int32(cmd.Protocols_PC2LS_JoinRoomReq),
			AppId:  global.ACTOR_SVC,
			UserId: h.actor.uid,
			RoleId: 0,
			UAID:   h.actor.Srv.UAID(h.actor.ID(), h.actor.roleId),
			Data:   data,
			// GUID:    utils.GenIntUUID(),
			ServerReqIdx: utils.GenIntUUID(),
		}

		resp, err := h.actor.Srv.ActorInvoke(global.RoomActorType, roomId, &msg)
		if err != nil {
			return err
		}
		h.Debugf(resp.Str())

	default:
		return errors.New(fmt.Sprintf("还未支持的操作类型, %d", operate))
	}
	return nil
}

func (h *GmHandler) GMCloseBattleCheck(param []string, commonData *clidto.Comdata) error {
	// 战斗校验开关
	if len(param) > 0 {
		v, err := strconv.Atoi(param[0])
		if err != nil {
			return err
		}
		conf.GConf().Base.OpenCheckBattle = int32(v)
	} else {
		conf.GConf().Base.OpenCheckBattle = 0
	}

	return nil
}

func (h *GmHandler) GMTestRecommend(param []string, comdata *clidto.Comdata) error {
	if len(param) > 0 {
		roleId, err := strconv.ParseUint(param[0], 10, 64)
		if err != nil {
			h.Errorf("ParseUint err:%v", err)
			return err
		}
		info, err := h.actor.getRoleDetailInfoByRoleId(roleId)
		if err != nil {
			h.Errorf("拉取好友数据出错了 err:%v", err)
			return err
		}
		h.Debugf("拉到的好友数据 %+v", info)
		return nil
	}

	list := h.actor.FriendHandler.getRecommendList()
	h.Debugf("测试好友推荐列表: %+v", list)
	return nil
}

func (h *GmHandler) AddCardsRelation(param []string, comdata *clidto.Comdata) error {
	if len(param) < 2 {
		return nil
	}
	cardIds := make([]int32, 0)
	id1, _ := strconv.Atoi(param[0])
	id2, _ := strconv.Atoi(param[1])
	value, _ := strconv.Atoi(param[2])
	cardIds = append(cardIds, int32(id1))
	cardIds = append(cardIds, int32(id2))
	h.actor.UserRelationHandler.GMAddRelation(cardIds, comdata, int32(value))
	return nil
}

func (h *GmHandler) CampDouble(param []string, comdata *clidto.Comdata) error {
	// building :=
	//	h.actor.CampHandler.LifeSkillAddProduct()
	return nil
}

func (h *GmHandler) CardBroad(param []string, comdata *clidto.Comdata) error {
	cardIds := make([]int32, 0)
	for _, id := range param {
		cardId, _ := strconv.Atoi(id)
		cardIds = append(cardIds, int32(cardId))
		rarity, _ := GetCardRarityByItemId(int32(cardId))
		h.Debug("GM 触发广播时，卡片的稀有度:", rarity)
	}
	h.actor.PoolHandler.BroadcastMessage(h.actor.roleId, cardIds)
	return nil
}

func (h *GmHandler) TestCfgHot(param []string, comdata *clidto.Comdata) error {

	cfg := excel.GetAbilityMgr().GetById(10101)
	h.Debugf("测试服务器热更:", cfg.SkilltreePara)

	return nil
}

// user.test.actorDel UserActor
func (h *GmHandler) GMDelActor(param []string, comdata *clidto.Comdata) error {
	h.Debugf("param:", param)
	if len(param) < 2 {
		return fmt.Errorf("param error")
	}

	actorType := param[0]
	actorId := param[1]
	err := h.actor.Srv.DeleteActor(actorType, actorId)
	if err != nil {
		return err
	}

	return nil
}

// GmTestCheckBattleReloadExcel 测试校验服热更新
// client CMD: user.test.checkBattleReloadExcel battle:192.168.1.135 achievement_data
// client CMD: user.test.checkBattleReloadExcel battleserver-0 achievement_data
func (h *GmHandler) GmTestCheckBattleReloadExcel(param []string, comdata *clidto.Comdata) error {
	fmt.Println("param:", param)
	if len(param) < 2 {
		return fmt.Errorf("param error")
	}

	battleTopic := param[0]
	fileName := param[1]

	hotReloadReq := &cmd.S2S_HotReloadReq{
		Type:     0,
		Files:    []string{fileName},
		Services: nil,
	}

	err := h.actor.Srv.PubTopicEvent(svc.EVENT_PRIVATE, battleTopic, h.actor.ID(), nil, hotReloadReq)
	if err != nil {
		h.Errorf("gm测试校验服热更新", err.Error())
	}

	_, err = h.actor.Srv.Daprc.InvokeMethodWithContent(context.Background(), battleTopic, "testSaveRedis", "post",
		&dapr.DataContent{Data: nil, ContentType: "text/plain"})

	return nil
}

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
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type PlayerLevelHandler struct {
	*UABaseHandler
}

// 升级奖励
type LevelUpReward struct {
	stamina int32
}

func NewPlayerLevelHandler(actor *UserActor) *PlayerLevelHandler {
	h := &PlayerLevelHandler{UABaseHandler: NewUABaseHandler(actor, "PlayerLevelHandler")}
	h.ChildHandler = h

	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerStaminaBuyReq), h.PlayerStaminaBuyReq)

	return h
}

func (h *PlayerLevelHandler) Init() error {
	// 初始化玩家业务信息
	h.actor.Data.PlayerLevelData = &cmd.PPlayerLevelInfo{
		Createtime: time.Now().Unix(),
		Stamina: &cmd.PStaminaInfo{
			Value:            excel.GetConfigMgr().GetCfg().NEWACCOUNTSTAMINA,
			LastRecoveryTime: time.Now().Unix(),
		},
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init stamina data success.")
	return nil
}

func (h *PlayerLevelHandler) EnterGame() error {
	return nil
}

func (h *PlayerLevelHandler) DailyRefresh() error {
	return nil
}

func (h *PlayerLevelHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PPlayerLevelInfo); ok {
		h.actor.Data.PlayerLevelData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *PlayerLevelHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserLevelData(h.actor.ID()), h.actor.Data.PlayerLevelData
}

func (h *PlayerLevelHandler) PlayerStaminaBuyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.C2LS_PlayerStaminaBuyReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	stamina := h.GetPlayerStamina()

	// 购买次数
	limit := excel.GetConfigMgr().GetCfg().STAMINA_PURCHASE_LIMIT
	if stamina.BuyCount >= limit {
		return nil, fmt.Errorf("stamina buy count is limit"), int32(cmd.ErrorCode_StaminaBuyCountLimit)
	}

	// 硬上限
	hardLimit := excel.GetConfigMgr().GetCfg().STAMINA_HARD_LIMIT
	value := excel.GetConfigMgr().GetCfg().STAMINA_PURCHASE_VALUE
	if stamina.GetValue()+value > hardLimit {
		return nil, fmt.Errorf("stamina value is limit"), int32(cmd.ErrorCode_StaminaValueLimit)
	}

	// 货币检查
	cfg := excel.GetStaminaPurchaseMgr().GetById(stamina.BuyCount + 1)
	if cfg == nil {
		return nil, fmt.Errorf("stamina config not found %d", stamina.BuyCount+1), int32(cmd.ErrorCode_NotFoundConfig)
	}
	if !GetConsumeMgr(h.actor).CheckKeyValEnough([]*excel.KeyVal{cfg.Cost}) {
		return nil, fmt.Errorf("currency not enough"), int32(cmd.ErrorCode_CurrencyNotEnough)
	}

	// 买一次
	err = GetConsumeMgr(h.actor).ConsumeKeyValList([]*excel.KeyVal{cfg.Cost}, h.actor.comData, common.CR_STAMINA_BUY)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	stamina.BuyCount++
	stamina.BuyTime = time.Now().Unix()

	_, err = GetDropMgr(h.actor).DropList2(map[int32]int32{common.ITEM_ID_STAMINA_1004: value}, true, nil, h.actor.comData, common.CR_STAMINA_BUY)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	return &cmd.LS2C_PlayerStaminaBuyRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// AddStamina 增加玩家体力
func (h *PlayerLevelHandler) AddStamina(value int32, commonData *clidto.Comdata, reason common.ChangeReason) error {
	// 增加
	stamina := h.GetPlayerStamina()
	beforeNum := stamina.Value // 变化前的体力
	stamina.Value += value
	afterNum := stamina.Value // 变化后的体力

	// 溢出判定
	hardLimit := excel.GetConfigMgr().GetCfg().STAMINA_HARD_LIMIT
	if stamina.GetValue() > hardLimit {
		sub := stamina.GetValue() - hardLimit
		stamina.Value = hardLimit
		h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_4, map[int32]int32{common.ITEM_ID_STAMINA_1004: sub}, commonData)
	}

	// 尝试修正时间记录
	limit := h.getSoftLimit()
	// 到上限了
	if stamina.GetLastRecoveryTime() > 0 && stamina.GetValue() >= limit {
		stamina.LastRecoveryTime = 0
	} else if stamina.GetLastRecoveryTime() == 0 && stamina.GetValue() < limit {
		// 上限提高了
		stamina.LastRecoveryTime = time.Now().Unix()
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	// 体力增加埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.StaminaChange{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_StaminaChange, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Action:         int32(reason),                          // 变化来源
	//		Num:            value,                                  // 变化数量
	//		BeforeNum:      beforeNum,                              // 变化前数量
	//		AfterNum:       afterNum,                               // 变化后数量
	//		Flow:           "in",                                   // 流向，获得为"in" 消耗为"out"
	//		Level:          h.actor.GetUserData().Common.RoleLevel, // 玩家等级
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.StaminaChange{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Action:            int32(reason),                          // 变化来源
			Num:               value,                                  // 变化数量
			BeforeNum:         beforeNum,                              // 变化前数量
			AfterNum:          afterNum,                               // 变化后数量
			Flow:              "in",                                   // 流向，获得为"in" 消耗为"out"
			Level:             h.actor.GetUserData().Common.RoleLevel, // 玩家等级
		}
		taptap.WriteDataLog(taptap.LogType_StaminaChange, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	commonData.Data.Stamina = stamina
	return nil
}

// SubStamina 扣除玩家体力
func (h *PlayerLevelHandler) SubStamina(value int32, commonData *clidto.Comdata, reason common.ChangeReason) error {

	stamina := h.GetPlayerStamina()

	// 够扣除？
	if stamina.GetValue() < value {
		return fmt.Errorf("sub stamina failed. curValue: %d, subValue: %d", stamina.GetValue(), value)
	}

	beforeNum := stamina.Value // 变化前的体力
	stamina.Value -= value
	afterNum := stamina.Value // 变化后的体力
	// 可恢复货币处理初始化
	if stamina.LastRecoveryTime == 0 {
		limit := h.getSoftLimit()
		// 小于软上限
		if stamina.Value < limit {
			stamina.LastRecoveryTime = time.Now().Unix()
		}
	}

	if err := h.SaveDB(); err != nil {
		return err
	}
	commonData.Data.Stamina = stamina

	// 扣除体力埋点
	if value > 0 {
		//threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.StaminaChange{
		//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_StaminaChange, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		Action:         int32(reason),                          // 变化来源
		//		Num:            value,                                  // 变化数量
		//		BeforeNum:      beforeNum,                              // 变化前数量
		//		AfterNum:       afterNum,                               // 变化后数量
		//		Flow:           "out",                                  // 流向，获得为"in" 消耗为"out"
		//		Level:          h.actor.GetUserData().Common.RoleLevel, // 玩家等级
		//	})
		//})
		threading.RunSafe(func() {
			e := &taptap.StaminaChange{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				Action:            int32(reason),                          // 变化来源
				Num:               value,                                  // 变化数量
				BeforeNum:         beforeNum,                              // 变化前数量
				AfterNum:          afterNum,                               // 变化后数量
				Flow:              "out",                                  // 流向，获得为"in" 消耗为"out"
				Level:             h.actor.GetUserData().Common.RoleLevel, // 玩家等级
			}
			taptap.WriteDataLog(taptap.LogType_StaminaChange, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_STAMINA_SUB, []int32{TASK_TYPE_112}, map[string]interface{}{
		"count": value,
	}))
	if errx != nil {
		h.Error(errx)
	}

	return nil
}

func (h *PlayerLevelHandler) buildPlayerStaminaInfo() *cmd.PStaminaInfo {

	// 刷新数据
	stamina := h.tryRecovery()
	if err := h.SaveDB(); err != nil {
		return nil
	}

	return stamina
}

// CheckStaminaEnough 检查体力是否足够，足够则返回true
func (h *PlayerLevelHandler) CheckStaminaEnough(value int32) bool {
	stamina := h.GetPlayerStamina()
	return stamina.GetValue() >= value
}

func (h *PlayerLevelHandler) GetPlayerStamina() *cmd.PStaminaInfo {
	return h.tryRecovery()
}

// 尝试刷新回复值
func (h *PlayerLevelHandler) tryRecovery() *cmd.PStaminaInfo {
	stamina := h.actor.GetPlayerLevelData().Stamina

	now := time.Now().Unix()
	rTime := int64(excel.GetConfigMgr().GetCfg().STAMINA_RECOVERY_INTERVAL) // 每次回复的时间间隔
	limit := h.getSoftLimit()                                               // 软上限
	rValue := excel.GetConfigMgr().GetCfg().STAMINA_RECOVERY_RATE           // 每次回复的回复量

	var rNum int32
	// 计算出恢复次数
	if stamina.GetValue() < limit && stamina.GetLastRecoveryTime()+rTime <= now {
		rNum1 := (limit - stamina.GetValue()) / rValue
		rNum2 := int32((now - stamina.GetLastRecoveryTime()) / rTime)
		rNum = utils.Min(rNum1, rNum2)
	}

	// 刷新数据
	if rNum > 0 {
		stamina.LastRecoveryTime += rTime * int64(rNum)
		stamina.Value += rValue * rNum
	}

	// 后置处理
	if stamina.GetValue() >= limit {
		stamina.LastRecoveryTime = 0
	}

	// 重置次数处理
	if now >= stamina.ResetTime {
		stamina.ResetTime = common.GetNextDailyRefreshTime()
		stamina.BuyCount = 0
	}

	return stamina
}

// 获取软上限
func (h *PlayerLevelHandler) getSoftLimit() int32 {
	level := h.actor.LoginHandler.getRoleLevel()

	cfg := excel.GetPlayerLevelMgr().GetById(int32(level))
	if cfg == nil {
		return 0
	}

	return cfg.GetStaminaLimit()
}

func (h *PlayerLevelHandler) CheckLimit(value int32) bool {
	item := h.GetPlayerStamina()

	hardLimit := excel.GetConfigMgr().GetCfg().STAMINA_HARD_LIMIT
	if item.GetValue()+value > hardLimit {
		return true
	}

	return false
}

func (h *PlayerLevelHandler) useStaminaItemCheck(itemId, itemNum int32) cmd.ErrorCode {
	cfg := excel.GetItemMgr().GetById(itemId)
	if cfg == nil {
		return cmd.ErrorCode_NotFoundConfig
	}

	stamina := h.GetPlayerStamina()
	hardLimit := excel.GetConfigMgr().GetCfg().STAMINA_HARD_LIMIT
	if stamina.GetValue()+cfg.UseEffectShow*itemNum > hardLimit {
		return cmd.ErrorCode_StaminaValueLimit
	}
	return cmd.ErrorCode_Success
}

func (h *PlayerLevelHandler) useStaminaItem(commonData *clidto.Comdata, itemId, itemNum int32) error {
	cfg := excel.GetItemMgr().GetById(itemId)
	if cfg == nil {
		return fmt.Errorf("cfg not found %d", itemId)
	}
	_, err := GetDropMgr(h.actor).DropList2(map[int32]int32{common.ITEM_ID_STAMINA_1004: cfg.UseEffectShow * itemNum}, true, nil, commonData, common.CR_UseItem)
	if err != nil {
		return err
	}

	return nil
}

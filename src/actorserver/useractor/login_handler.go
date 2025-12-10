package useractor

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/server"

	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"

	"github.com/pkg/errors"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

const UidBase = 10000000000

type LoginHandler struct {
	*UABaseHandler
}

func NewLoginHandler(actor *UserActor) *LoginHandler {
	h := &LoginHandler{UABaseHandler: NewUABaseHandler(actor, "LoginHandler")}
	h.ChildHandler = h
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2S_KickoutPlayerNtf), h.KickoutPlayer)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ChangeNicknameReq), h.ChangeNicknameReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ChangeHeadReq), h.ChangeHeadReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CreateRoleInfoReq), h.CreateRoleInfoReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2G_LoginGameReq), h.LoginEnterGame)
	return h
}

func (h *LoginHandler) Init() error {
	now := time.Now().Unix()
	h.actor.Data.Base = &cmd.PServerRoleBaseInfo{Createtime: now}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	return nil
}

func (h *LoginHandler) EnterGame() error {
	// 尝试解锁头像
	return h.tryUnlockHeads(common.HEAD_UNLOCK_TYPE_3, 0)
}

func (h *LoginHandler) DailyRefresh() error {
	// 跨天在线玩家自动上报一次
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.RoleLogin{
	//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_RoleLogin, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		RoleId:   h.actor.ID(),
	//		Level:    int32(h.getRoleLevel()),
	//		VipLevel: 0,
	//		Recharge: 0,
	//		Language: h.actor.Data.Base.Common.Language,
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.RoleLogin{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			RoleId:            h.actor.ID(),
			Level:             int32(h.getRoleLevel()),
			VipLevel:          0,
			Recharge:          0,
			Language:          h.actor.Data.Base.Common.Language,
		}
		taptap.WriteDataLog(taptap.LogType_RoleLogin, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})
	return nil
}

func (h *LoginHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PServerRoleBaseInfo); ok {
		h.actor.Data.Base = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *LoginHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserBaseInfo(h.actor.ID()), h.actor.Data.Base
}

// 尝试解锁玩家头像
func (h *LoginHandler) tryUnlockHeadsEvent(e event.IEvent) error {
	var err error

	name := e.Name()
	if name == TASK_EVENT_CARD_CREATE {
		cardId := e.Get("cardId").(int32)
		err = h.tryUnlockHeads(common.HEAD_UNLOCK_TYPE_1, cardId)
	} else if name == TASK_EVENT_BREAKTHROUGH {
		cardId := e.Get("card_id").(int32)
		isAwaken := e.Get("is_awaken").(bool)
		if isAwaken {
			err = h.tryUnlockHeads(common.HEAD_UNLOCK_TYPE_2, cardId)
		}
	} else {
		h.Warnf("unrealized event type %s", name)
	}
	if err != nil {
		h.Errorf("tryUnlockHeadsEvent got err: %v", err)
	}
	return nil
}

func (h *LoginHandler) tryUnlockHeads(unlockType, param int32) error {
	newHeads := make([]int32, 0)
	tempMap := make(map[int32]int32)
	excel.GetPlayerInfoMgr().Foreach(func(cfg *excel.PlayerInfoCfg) bool {
		if cfg.Type != common.PLAYER_HEAD {
			return true
		}
		if cfg.UnlockWay != unlockType {
			return true
		}
		// 尝试解锁
		if cfg.WayId == param {
			newHeads = append(newHeads, cfg.Id)
		}

		return true
	}, true)

	data := h.actor.GetUserData()
	if len(newHeads) > 0 {
		for _, id := range data.Heads {
			tempMap[id] = id
		}
		for _, id := range newHeads {
			if _, ok := tempMap[id]; !ok {
				data.Heads = append(data.Heads, id)
				data.NewHeads = append(data.NewHeads, id)
				h.actor.comData.GetBaseData().Heads = append(h.actor.comData.GetBaseData().Heads, id)
				h.actor.comData.GetBaseData().NewHeads = append(h.actor.comData.GetBaseData().NewHeads, id)
			}
		}
	}
	// 默认头像容错
	if unlockType == common.HEAD_UNLOCK_TYPE_3 && data.Common.RoleHead == 0 && len(data.Heads) > 0 {
		data.Common.RoleHead = data.Heads[0]
		h.actor.RoleDetailHandler.ChangeHeadId(data.Common.RoleHead)
	}
	// 自动佩戴
	if unlockType == common.HEAD_UNLOCK_TYPE_4 {
		data.Common.RoleHead = newHeads[0]
		h.actor.RoleDetailHandler.ChangeHeadId(data.Common.RoleHead)
	}
	return h.SaveDB()
}

func (h *LoginHandler) KickoutPlayer(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.S2S_KickoutPlayerNtf
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	err = h.sendPlayerKickOutNtf(req.Reason)
	if err != nil {
		h.Warn(err)
	}
	return nil, nil, 0
}

func (h *LoginHandler) changeNickname(newNickname string) {
	h.actor.GetUserData().Common.RoleName = newNickname
	h.actor.GetUserData().ChangeName++
	if err := h.SaveDB(); err != nil {
		h.Errorf("LoginHandler 更新昵称报错, err:%v", err.Error())
	}

	// 同步到account数据中
	h.actor.AccountHandler.ChangeNickname(newNickname)
	h.actor.RoleDetailHandler.ChangeNickname(newNickname)
}

func (h *LoginHandler) ChangeNicknameReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_ChangeNicknameReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	err, code := h.checkNickName(req.NewNickname, false)
	if err != nil {
		return nil, err, code
	}
	err, code = h.handleChangeNickname(req.NewNickname, false)
	if err != nil {
		return nil, err, code
	}
	return &cmd.LS2C_ChangeNicknameRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *LoginHandler) checkNickName(nickname string, isCreate bool) (error, int32) {
	nameLen := utf8.RuneCountInString(nickname)
	if nameLen == 0 || nameLen > 7 {
		return fmt.Errorf("nickname length is illegal"), int32(cmd.ErrorCode_NicknameInvalid)
	}

	data := h.actor.GetUserData()
	if !isCreate && data.Common.RoleName == nickname {
		return fmt.Errorf("nickname not changed"), int32(cmd.ErrorCode_NicknameInvalid)
	}

	// 校验合法性
	if h.actor.Srv.CheckSpecialLetters(nickname, false) {
		return fmt.Errorf("nickname invalid"), int32(cmd.ErrorCode_NicknameInvalid)
	}
	result, err := h.actor.Srv.CheckSensitiveWord(common.CHECK_TYPE_PLAYERNAME, nickname)
	if err != nil {
		return err, int32(cmd.ErrorCode_InternalError)
	}
	if !result {
		return fmt.Errorf("nickname invalid"), int32(cmd.ErrorCode_NicknameInvalid)
	}

	// 扣除改名道具
	if !isCreate && data.ChangeName > 0 {
		cost := excel.GetConfigMgr().GetCfg().PLAYER_RENAME_COST
		if !GetConsumeMgr(h.actor).CheckKeyValEnough([]*excel.KeyVal{cost}) {
			return fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
		}
	}
	return nil, 0
}

func (h *LoginHandler) handleChangeNickname(nickname string, isCreate bool) (error, int32) {
	if !isCreate && h.actor.GetUserData().ChangeName > 0 {
		cost := excel.GetConfigMgr().GetCfg().PLAYER_RENAME_COST
		err := GetConsumeMgr(h.actor).ConsumeKeyValList([]*excel.KeyVal{cost}, h.actor.comData, common.CR_CHANGE_NICKNAME)
		if err != nil {
			return err, int32(cmd.ErrorCode_InternalError)
		}
	}
	// 保存新的昵称
	h.changeNickname(nickname)
	h.actor.comData.Data.Base = h.buildRoleBaseInfo()
	h.Debug("修改昵称成功")
	return nil, 0
}

func (h *LoginHandler) CreateRoleInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_CreateRoleInfoReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	// 校验昵称
	err, code := h.checkNickName(req.Nickname, true)
	if err != nil {
		return nil, err, code
	}
	// 校验性别
	if req.Sex != int32(cmd.RoleSexType_RoleSexType_Female) && req.Sex != int32(cmd.RoleSexType_RoleSexType_Male) {
		return nil, fmt.Errorf("illegal param"), int32(cmd.ErrorCode_InvalidParam)
	}
	// 修改昵称
	err, code = h.handleChangeNickname(req.Nickname, true)
	if err != nil {
		return nil, err, code
	}
	// 修改性别
	h.changeSex(req.Sex)
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	return &cmd.LS2C_CreateRoleInfoRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *LoginHandler) changeSex(sex int32) {
	data := h.actor.GetUserData()
	data.Common.RoleSex = uint32(sex)
	h.tryUnlockHeads(common.HEAD_UNLOCK_TYPE_4, sex)
	h.actor.RoleDetailHandler.ChangeRoleSex(sex)
}

func (h *LoginHandler) ChangeHeadReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_ChangeHeadReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 头像是否拥有
	data := h.actor.GetUserData()
	var exist bool
	for _, head := range data.Heads {
		if head == req.HeadId {
			exist = true
			break
		}
	}
	if !exist {
		return nil, fmt.Errorf("head not found"), int32(cmd.ErrorCode_ParamError)
	}

	// 保存
	data.Common.RoleHead = req.HeadId
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	h.actor.RoleDetailHandler.ChangeHeadId(req.HeadId)
	h.Debug("修改头像成功")
	return &cmd.LS2C_ChangeHeadRes{HeadId: req.HeadId}, nil, 0
}

// 处理头像红点标识
func (h *LoginHandler) handleRedPoint(commonData *clidto.Comdata, ids []int64) error {
	userData := h.actor.GetUserData()
	for _, id := range ids {
		userData.NewHeads = myUtils.DeleteAllByElement(userData.NewHeads, int32(id))
	}
	return h.SaveDB()
}

func (h *LoginHandler) buildRoleBaseInfo() *cmd.PClientRoleBaseInfo {
	return toClientBaseInfo(h.actor.GetUserData())
}

func (h *LoginHandler) toClientDetailInfo(detail *cmd.PServerRoleDetailInfo) *cmd.PClientRoleDetailInfo {
	// 生涯数据
	lifes := make([]int32, 0)
	lifes = append(lifes, detail.Lifex[0])
	lifes = append(lifes, detail.Lifex[1])
	lifes = append(lifes, detail.Lifex[2])
	// 卡牌数据
	ret, err := h.actor.getCardDataByRoleId(detail.Common.RoleId, detail.Cards)
	if err != nil {
		h.Errorf("toClientDetailInfo got err: %v", err)
	}

	return &cmd.PClientRoleDetailInfo{
		Common: detail.Common,
		Lifes:  lifes,
		Cards:  ret,
	}
}

func toClientBaseInfo(server *cmd.PServerRoleBaseInfo) *cmd.PClientRoleBaseInfo {
	return &cmd.PClientRoleBaseInfo{
		Common:     server.Common,
		ChangeName: server.ChangeName,
		Heads:      server.Heads,
		NewHeads:   server.NewHeads,
	}
}

func (h *LoginHandler) getStoryFlags() []*cmd.FlagInfo {
	var (
		flagList = make([]*cmd.FlagInfo, 0)
	)

	// 全量推送
	flagData := h.actor.GetStoryFlagData()
	for _, flag := range flagData.Flags {
		flagList = append(flagList, flag)
	}

	return flagList
}

func (h *LoginHandler) sendPlayerKickOutNtf(reason string) error {
	ntf := &cmd.GWS2C_KickOutNtf{
		Reason: reason,
	}

	err := h.actor.PushMsg2Gate(ntf)
	if err != nil {
		h.Error("actor invoke to gate got err ", err)
		return err
	}

	h.Debug("sendPlayerKickOutNtf :", h.actor.ID())
	return nil
}

// 新手礼包
func (h *LoginHandler) handleNewbieGift() error {
	items := excel.GetConfigMgr().GetCfg().ACCOUNT_CREATE_STATE

	_, err := GetDropMgr(h.actor).DropListByItems(items, true, nil, h.actor.comData, common.CR_NewbieGift)
	if err != nil {
		return err
	}

	h.actor.GetUserData().IsNewbieGift = true
	err = h.SaveDB()
	if err != nil {
		return err
	}

	h.Infof("handleNewbieGift success. player：%s", h.actor.ID())
	return nil
}

// 尝试刷新最后登陆时间戳
// @return 是否隔天登陆
// @return error
func (h *LoginHandler) TryUpdateLastLoginDate() (bool, error) {
	// 上次登陆的时间戳
	oldLastLoginDate := time.Unix(h.actor.GetUserData().Common.OnlineTime, 0)

	// 更新本次登陆时间戳
	h.UpdateOnlineTS(time.Now().Unix())
	// h.UpdateOfflineTS(-1)

	// 判断是否跨天
	if !common.IsSameDayByOffset(oldLastLoginDate, time.Now(), common.GAME_DAILY_REFRESH_HOUR) {
		// 隔天登陆
		h.actor.GetUserData().Common.LoginDay++
		h.actor.RoleDetailHandler.ChangeLoginDay()
		// 发布事件
		err := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_PLAYER_LOGIN, []int32{TASK_TYPE_501, TASK_TYPE_503}, map[string]interface{}{
			"login_day": h.actor.GetUserData().Common.LoginDay,
		}))
		if err != nil {
			h.Error(err)
		}

		return true, nil
	}

	return false, nil
}

func (h *LoginHandler) LoginEnterGame(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	h.actor.SetUID(in.UserId)
	req := &cmd.C2G_LoginGameReq{}
	err := in.UnmarshalData(req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	logger.Infof("OnLoginGame [LoginStep] userId:%s, roleId:%v, uaid:%s, req:%+v", in.UserId, in.RoleId, in.UAID, req)

	h.actor.roleId = in.RoleId
	if h.actor.roleId == 0 {
		return nil, errors.New("roleId is 0"), int32(cmd.ErrorCode_InternalError)
	}

	// check role exist, return old role data,not create
	// 查询账号,查不到返回错误码

	// account, err := h.actor.Srv.GetAccount(db.KeyAccountInfo(h.actor.GetUID()))
	err = h.actor.loadDBDataByDBType(service.MongoDbType_MongoAccount)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_NotFoundAccount)
	}

	curTime := time.Now().Unix()

	player, ok := h.actor.Account.PlayerList.Players[1]
	_, err = h.actor.Srv.GetMongoGame(db.KeyUserBaseInfo(h.actor.ID()), nil)

	// var bSyncDB bool
	// 角色不存在 创建角色
	var bNewPlayer bool
	if err != nil && errors.Is(err, service.DB_ERROR_NOT_EXIST) && !ok && player == nil {
		// 配置了注册上限，判定一下
		var isRegisterLimit bool
		if limit, err := h.actor.Srv.GetConfigKeyForInt(db.KeyCfgServerRegisterLimit); err == nil && limit > 0 {
			count, err := h.actor.Srv.RedisBitCount(context.Background(), db.KeyServerRegisterUsers(), nil)
			if err != nil || count >= int64(limit) {
				return nil, err, int32(cmd.ErrorCode_RegisterLimit)
			}
			isRegisterLimit = true
		}

		err, errCode := h.CreatePlayer(h.actor.roleId, curTime, req.Language)
		if err != nil || errCode != 0 {
			return nil, err, int32(errCode)
		}
		// h.actor.Account.PlayerList.UserMap[1] = &cmd.Player{Id: h.actor.roleId, CreateTs: curTime}
		// h.actor.Account.PlayerList.Pid = h.actor.roleId
		// h.actor.Account.PlayerList.UpdateTs = curTime
		h.actor.AccountHandler.SavePlayer(in.RoleId)
		// bSyncDB = true
		bNewPlayer = true

		// 记录注册redis
		if isRegisterLimit {
			_, err = h.actor.Srv.RedisSetBit(context.Background(), db.KeyServerRegisterUsers(), int64(h.actor.roleId), 1)
			if err != nil {
				return nil, err, int32(cmd.ErrorCode_InternalError)
			}
		}

		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.UserCreate{
		//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_UserCreate, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//	})
		// })

		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.RoleCreate{
		//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_RoleCreate, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		RoleId:   strconv.FormatUint(h.actor.roleId, 10),
		//	})
		// })
		threading.RunSafe(func() {
			e := &taptap.RoleCreate{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(h.actor.roleId, 10),
			}
			taptap.WriteDataLog(taptap.LogType_RoleCreate, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	_, err, errCode := h.DoEnterGame(bNewPlayer)

	// 解除绑定roomId
	if req.IsReconnect == 0 {
		h.actor.UserRoomHandler.tryExitRoom()
	}

	// 手动清除一次actor上的缓存数据
	h.actor.comData = clidto.BuildComData()

	// 构建最新的数据
	h.actor.comData.Data = &cmd.CliComData{
		ServerTimestamp:     time.Now().UnixMilli(),
		OpenServerTimestamp: time.Now().Unix() - 60*60*12, // TODO 临时数据
		NextRefreshTime:     common.GetNextDailyRefreshTime(),
		NewMails:            h.actor.MailHandler.getNewMailsCount(),
		Base:                h.buildRoleBaseInfo(),
		Items:               h.actor.BagHandler.buildItemList(),
		Equip:               h.actor.EquipHandler.buildEquipList(),
		Card:                h.actor.CardHandler.buildCardList(),
		Currency:            h.actor.CurrencyHandler.buildCurrencyList(),
		Tutorial:            h.actor.TutorialHandler.buildPlayerBeginnerTutorial(),
		Troop:               h.actor.TroopHandler.buildTroopList(),
		Duty:                h.actor.DutyHandler.buildDutyInfo(false),
		SignGroups:          h.actor.SignHandler.buildSignInfo(),
		Flags:               h.getStoryFlags(),
		Stamina:             h.actor.PlayerLevelHandler.buildPlayerStaminaInfo(),
		Friends:             h.actor.FriendHandler.buildFriendData(true),
		Alliance:            h.actor.UserAllianceHandler.buildAllianceData(true),
		GuideTask:           h.actor.GuideTaskHandler.buildGuideTask(),
		ActivityData:        h.actor.ActivityHandler.formatActivity2Client(),
	}

	res := &cmd.G2C_LoginGameRes{
		Err_Code:        errCode,
		RoleId:          h.actor.roleId,
		Ticket:          0,
		ServerTimestamp: time.Now().UnixMilli(),
		CommonData:      h.actor.comData.FixDownComData(),
		OrderInfo:       h.actor.OrderHandler.buildOrderInfo(),
	}

	// 下发副本id
	if h.actor.GetLevelsData().InSubLevel == cmd.InSubLevelType_yes {
		res.LevelId = h.actor.GetLevelsData().CurrLevelId
	}

	// 重连
	if req.IsReconnect == 1 {
		taptap.ReconnectComm(h.actor.uid, h.actor.Account.TapUserInfo, h.actor.Account.CliDeviceInfo)
	}

	// 向allianceActor 发送topic 信息
	h.actor.UserAllianceHandler.PushTopic2Alliance(cmd.GateTopicOperator_GTO_bind, in.GetTopic())

	h.actor.SetState(State_Online)
	h.Infof("[UserActor] %s EnterGame State: %v", h.actor.ID(), cmd.ErrorCode(errCode))
	return res, err, errCode
}

func (h *LoginHandler) CreatePlayer(playerId uint64, ts int64, language string) (error, cmd.ErrorCode) {

	// 查询账号,查不到返回错误码
	roleBaseInfo := &cmd.PCommonRoleBaseInfo{
		RoleName:        "test",
		RoleId:          playerId,
		RoleSex:         uint32(cmd.RoleSexType_RoleSexType_None),
		RoleLevel:       1,
		RoleExp:         0,
		OnlineTime:      ts,
		CreateTimestamp: ts,
		Language:        language,
		LoginDay:        1,
	}

	role := &cmd.PServerRoleBaseInfo{
		Common:                  roleBaseInfo,
		IsNewRole:               true,
		ChangeName:              0,
		LastSaveTimestamp:       ts,
		FirstEnterGameTimestamp: ts,
		FirstLoginTimestamp:     ts,
	}

	// TODO 增加配置选项查询role

	// 查询 role:uaid
	if conf.GConf().BaseConf().RoleIdCheck {
		_, err := h.actor.Srv.GetCache(service.MongoDbType_MongoGame, db.KeyPlayerUAID(playerId), server.ICache(h.actor.Srv))
		if err == nil || err != nil && !errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			h.Error("[GateServer] CreatePlayer playerId check failed,", err)
			return err, cmd.ErrorCode_InternalError
		}
	}

	h.actor.Data.Base = role

	// 持久化
	err := h.SaveDB(true)
	if err != nil {
		h.Error("[GateServer] SaveDB save db failed,", err)
		return err, cmd.ErrorCode_SaveDBError
	}

	// 更新uaid缓存
	h.actor.Srv.UpdateUAIDCache(h.actor.uid, h.actor.roleId, true)

	h.Infof("CreatePlayer player: %+v", h.actor.Data.Base)

	defer func() {
		// 立即落库 防止在UserActor.SaveState中, 从DB里获取数据覆盖内存中的
		h.actor.FixedTime2DB()
	}()

	return nil, cmd.ErrorCode_Success
}

func (h *LoginHandler) DoEnterGame(bNewPlayer bool) (proto.Message, error, int32) {

	h.Infof("UserActor DoEnterGame, is_new: %v", bNewPlayer)

	// 全量load玩家数据
	if bNewPlayer {
		err := h.actor.loadDBDataByDBType(service.MongoDbType_MongoGame)
		if err != nil {
			h.Errorf("DoEnterGame, load user data failed: %+v", err)
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}

		// 调用EnterGame接口
		err = h.actor.EnterGame()
		if err != nil {
			h.Warnf("DoEnterGame, err:%+v", err)
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}

		// 更新最近上线时间
		err = h.actor.DailyRefreshAll()
		if err != nil {
			h.Debugf("dailyRefresh, 获取最后次登陆时间报错, err:%+v", err)
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 判定新手礼包发放
	if !h.actor.GetUserData().IsNewbieGift {

		if err := h.handleNewbieGift(); err != nil {
			h.Debug(err)
		}

		// 初始化默认阵容
		if err, errCode := h.actor.TroopHandler.CardTroopOperate(
			cmd.CardTroopType_CardTroopType_Normal, 1, // 固定默认值
			cmd.CardTroopSubType_Map_Out, excel.GetConfigMgr().GetCfg().DEFAULT_LEVEL_LINEUP); err != nil {
			return nil, err, int32(errCode)
		}

		if err := h.SaveDB(); err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 发布事件
	err := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_ENTER_GAME, []int32{}, nil))
	if err != nil {
		h.Error(err)
	}

	h.UpdateOfflineTS(-1)

	// 尝试同步es
	h.actor.RoleDetailHandler.tryUploadRoleInfoToES()

	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.UserLogin{
	//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_UserLogin, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.RoleLogin{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			RoleId:            h.actor.ID(),
			Level:             int32(h.getRoleLevel()),
			VipLevel:          0,
			Recharge:          0,
			Language:          h.actor.Data.Base.Common.Language,
		}
		taptap.WriteDataLog(taptap.LogType_RoleLogin, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return nil, nil, int32(cmd.ErrorCode_Success)
}

// 玩家等级
func (h *LoginHandler) getRoleLevel() uint32 {
	return h.actor.GetUserData().Common.RoleLevel
}

// 更新角色基础属性
func (h *LoginHandler) updateRoleBase(newLevel uint32, newExp uint64, reward *LevelUpReward, commonData *clidto.Comdata) error {
	h.actor.GetUserData().Common.RoleExp = newExp
	h.actor.GetUserData().Common.RoleLevel = newLevel

	if reward != nil && reward.stamina > 0 {
		_, err := GetDropMgr(h.actor).DropList2(map[int32]int32{common.ITEM_ID_STAMINA_1004: reward.stamina}, true, nil, commonData, common.CR_STAMINA_PLAY_LEVEL)
		if err != nil {
			return err
		}
	}

	if err := h.SaveDB(); err != nil {
		return err
	}
	h.actor.RoleDetailHandler.ChangeRoleLevel(newExp, newLevel)
	// 刷新剧情任务
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_ROLE_LEVEL_CHANGE, []int32{TASK_TYPE_504}, map[string]interface{}{
		"level": int32(newLevel),
	}))
	if errx != nil {
		h.Error(errx)
	}

	commonData.Data.Base = h.actor.LoginHandler.buildRoleBaseInfo()
	return nil
}

// AddRoleExp 人物经验增加
// 返回实际增加的经验值和错误信息
func (h *LoginHandler) AddRoleExp(expValue uint64, commonData *clidto.Comdata) (uint64, error) {

	maxLevel := uint32(excel.GetConfigMgr().GetCfg().PLAYER_MAX_LEVEL)
	oldLevel := h.getRoleLevel()
	oldExp := h.actor.GetUserData().Common.RoleExp
	maxLevelConfig := excel.GetPlayerLevelMgr().GetById(int32(maxLevel))

	if maxLevelConfig == nil {
		return 0, fmt.Errorf("player max level config not found: %d", maxLevel)
	}
	remainExp := oldExp + expValue
	newLevel := oldLevel

	var addExp uint64
	var lvUpTimes uint32

	reward := &LevelUpReward{}
	// 获取当前等级配置
	curLevelConfig := excel.GetPlayerLevelMgr().GetById(int32(oldLevel))
	// 配置不存在，返回错误
	if curLevelConfig == nil {
		return 0, fmt.Errorf("player level config not found: %d", oldLevel)
	}

	for {
		expLimit := uint64(curLevelConfig.Exp)
		if remainExp < expLimit {
			break
		}
		// 获取下一等级配置
		nextLevelConfig := excel.GetPlayerLevelMgr().GetById(int32(newLevel + 1))
		if nextLevelConfig == nil || maxLevel <= newLevel {
			// 配置不存在或者达到当前等级上限了,不可以升级
			remainExp = expLimit
			break
		}
		// 可以升级
		remainExp -= expLimit
		newLevel++
		lvUpTimes++
		if lvUpTimes == 1 {
			addExp = expLimit - oldExp
		} else {
			addExp += expLimit
		}
		reward.stamina += nextLevelConfig.StaminaGainUpgrade
		curLevelConfig = nextLevelConfig
	}
	if newLevel != oldLevel || remainExp != oldExp {
		if lvUpTimes == 0 {
			addExp = remainExp - oldExp
		} else {
			addExp += remainExp
		}

		if newLevel != oldLevel {
			// threading.RunSafe(func() {
			//	lilith.WriteDataLog(&lilith.LevelUp{
			//		HeadInfo: lilith.BuildHeadInfo(lilith.LogType_LevelUp, h.actor.uid, h.actor.Account.CliDeviceInfo),
			//		RoleId:   strconv.FormatUint(h.actor.roleId, 10),
			//		Level:    int32(newLevel),
			//		VipLevel: 0,
			//		Recharge: 0,
			//	})
			// })
			threading.RunSafe(func() {
				e := &taptap.LevelUp{
					PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
					RoleId:            strconv.FormatUint(h.actor.roleId, 10),
					Level:             int32(newLevel),
					VipLevel:          0,
					Recharge:          0,
				}
				taptap.WriteDataLog(taptap.LogType_LevelUp, h.actor.uid, h.actor.Account.TapUserInfo, e)
			})
		}

		return addExp, h.updateRoleBase(newLevel, remainExp, reward, commonData)
	}
	return 0, nil
}

func (h *LoginHandler) getLoginDay() int32 {
	return h.actor.GetUserData().Common.LoginDay
}

// 直接提升玩家等级等, 主要为GM命令提供
// @level 表示升多少级
func (h *LoginHandler) DirectLevelUpByGM(level uint32, commonData *clidto.Comdata) error {
	oldLevel := h.getRoleLevel()
	oldExp := h.actor.GetUserData().Common.RoleExp
	maxLevel := uint32(excel.GetConfigMgr().GetCfg().PLAYER_MAX_LEVEL)

	maxLevelConfig := excel.GetPlayerLevelMgr().GetById(int32(maxLevel))
	if maxLevelConfig == nil {
		return fmt.Errorf("player max level config not found: %d", maxLevel)
	}

	targetLevel := oldLevel + level

	// 目标等级超过当前最大等级或者目标等级小于现在的等级
	if targetLevel > maxLevel || oldLevel > targetLevel {
		return fmt.Errorf("player level param is invliad,maybe greater than maxlevel(%d) or less than curlevel %d", maxLevel, oldLevel)
	}

	reward := &LevelUpReward{}

	for i := oldLevel + 1; i <= targetLevel; i++ {
		levelConfig := excel.GetPlayerLevelMgr().GetById(int32(i))
		if levelConfig == nil {
			return fmt.Errorf("player level config not found: %d", i)
		}
		reward.stamina += levelConfig.StaminaGainUpgrade
	}
	return h.updateRoleBase(targetLevel, oldExp, reward, commonData)
}

// 重置玩家等级GM
func (h *LoginHandler) ResetLevelByGM(commonData *clidto.Comdata) error {
	return h.updateRoleBase(1, 0, nil, commonData)
}

// 更新离线时间戳
func (h *LoginHandler) UpdateOfflineTS(ts int64) {
	h.actor.GetUserData().Common.OfflineTime = ts
	if err := h.SaveDB(true); err != nil {
		h.Error(err)
	}
	h.actor.RoleDetailHandler.ChangeOfflineTime(ts)
}

// 更新上线时间戳
func (h *LoginHandler) UpdateOnlineTS(ts int64) {
	h.actor.GetUserData().Common.OnlineTime = ts
	if err := h.SaveDB(); err != nil {
		h.Error(err)
	}
	h.actor.RoleDetailHandler.ChangeOnlineTime(ts)
}

// 检查玩家是否达到给定等级
func (h *LoginHandler) checkPlayerLevel(lv int) bool {
	userData := h.actor.GetUserData()
	return userData.Common.RoleLevel >= uint32(lv)
}

// 获取玩家的渠道
func (h *LoginHandler) GetRoleChannel() string {
	return h.actor.Account.Account.Channel
}

// 获取玩家的操作系统
func (h *LoginHandler) GetRoleOS() string {
	return h.actor.Account.CliDeviceInfo.Os
}

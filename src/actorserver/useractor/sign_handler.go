package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

const (
	SIGN_TYPE_DAILY      = 1 // 每日签到
	SIGN_TYPE_NEW_PLAYER = 2 // 新玩家签到
	SIGN_TYPE_OLD_PLAYER = 3 // 老玩家回归
	SIGN_TYPE_ACTIVITY   = 4 // 节日活动签到
)

type SignHandler struct {
	*UABaseHandler
}

func NewSignHandler(actor *UserActor) *SignHandler {
	h := &SignHandler{UABaseHandler: NewUABaseHandler(actor, "SignHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_InitSignReq), h.InitSignReq)
	h.actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_DaySignReq), h.DaySignReq) // 签到

	return h
}

// Init 初始化模块数据
func (h *SignHandler) Init() error {
	// 初始化
	h.actor.Data.Sign = &cmd.PSignData{
		Createtime: time.Now().Unix(),
		Sign:       make(map[int32]*cmd.PCommonSignInfo),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init sign data success.")
	return nil
}

func (h *SignHandler) EnterGame() error {
	return nil
}

func (h *SignHandler) DailyRefresh() error {
	signData := h.actor.GetSignData()
	err := h.tryRefreshSign(signData.Sign)
	if err != nil {
		return err
	}

	return nil
}

func (h *SignHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PSignData); ok {
		h.actor.Data.Sign = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *SignHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserSign(h.actor.ID()), h.actor.Data.Sign
}

func (h *SignHandler) buildSignInfo() []*cmd.PCommonSignInfo {

	signData := h.actor.GetSignData()
	err := h.tryRefreshSign(signData.Sign)
	if err != nil {
		return nil
	}

	signs := make([]*cmd.PCommonSignInfo, 0)
	for _, sign := range signData.Sign {
		signs = append(signs, sign)
	}

	return signs
}

func (h *SignHandler) InitSignReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	commonData := &cmd.CliComData{
		SignGroups: h.buildSignInfo(),
	}

	return &cmd.LS2C_InitSignRes{CommonData: commonData}, nil, 0
}

func (h *SignHandler) DaySignReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.C2LS_DaySignReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	cfg := excel.GetCalendarsMgr().GetById(req.GroupId)
	if cfg == nil {
		return nil, fmt.Errorf("CalendarsCfg not found %d", req.GroupId), int32(cmd.ErrorCode_NotFoundConfig)
	}

	// 参数校验
	if cfg.Category == SIGN_TYPE_DAILY && (req.Params <= 0 || req.Params > 3) {
		return nil, fmt.Errorf("invalid param %d", req.Params), int32(cmd.ErrorCode_InvalidParam)
	}

	signData := h.actor.GetSignData()
	err = h.tryRefreshSign(signData.Sign)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 签到组不存在
	info := signData.Sign[req.GroupId]
	if info == nil {
		return nil, fmt.Errorf("invalid groupId %d", req.GroupId), int32(cmd.ErrorCode_InvalidParam)
	}

	// 是否过期
	now := time.Now().Unix()
	if now > info.End {
		return nil, fmt.Errorf("sign group is end %d", req.GroupId), int32(cmd.ErrorCode_SignGroupIsEnd)
	}

	// 今日是否签到过
	if now < info.NextSign {
		return nil, fmt.Errorf("today had sign %d", req.GroupId), int32(cmd.ErrorCode_DutyHadSign)
	}

	// 处理逻辑
	info.NextSign = common.GetNextDailyRefreshTime()
	info.Signed++
	if cfg.Category == SIGN_TYPE_DAILY {
		cardId := h.actor.DutyHandler.GetCurDutyCard()
		history := &cmd.PSignDayInfo{Params: []int32{cardId, req.Params}}
		info.Sign = append(info.Sign, history)
	}

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 签到奖励
	reward := make(map[int32]int32)
	excel.GetDailySigninMgr().Foreach(func(rewardCfg *excel.DailySigninCfg) bool {
		if rewardCfg.CalendarId == req.GroupId && rewardCfg.Day == info.Signed {
			reward[rewardCfg.Reward.Key] += rewardCfg.Reward.Val
		}
		return true
	}, true)

	if cfg.Category == SIGN_TYPE_DAILY {
		favor := excel.GetConfigMgr().GetCfg().DUTY_SIGNIN_FAVOR
		reward[favor.Key] += favor.Val
	}

	if len(reward) > 0 {
		_, err = GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Daily_Sign)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.DaySign{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_DaySign, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		GroupId:        req.GroupId,                   // 签到组id
	//		Params:         req.Params,                    // 特殊签到参数，无参数则为0
	//		Category:       cfg.Category,                  // 签到类型
	//		Counter:        info.Signed,                   // 当前签到次数
	//		Reward:         lilith.ConvertMap2Str(reward), // 签到奖励
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.DaySign{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			GroupId:           req.GroupId,                   // 签到组id
			Params:            req.Params,                    // 特殊签到参数，无参数则为0
			Category:          cfg.Category,                  // 签到类型
			Counter:           info.Signed,                   // 当前签到次数
			Reward:            taptap.ConvertMap2Str(reward), // 签到奖励
		}
		taptap.WriteDataLog(taptap.LogType_DaySign, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &cmd.LS2C_DaySignRes{SignInfo: info, CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *SignHandler) DaySignByGM(groupId, param int32, commonData *clidto.Comdata) error {

	signData := h.actor.GetSignData()
	err := h.tryRefreshSign(signData.Sign)
	if err != nil {
		return err
	}

	// 签到组不存在
	info := signData.Sign[groupId]
	if info == nil {
		return fmt.Errorf("sign group not found %d", groupId)
	}

	// 过期了
	now := time.Now().Unix()
	if now > info.End {
		return fmt.Errorf("sign group is end %d", groupId)
	}

	cfg := excel.GetCalendarsMgr().GetById(groupId)
	if cfg == nil {
		return fmt.Errorf("CalendarsCfg not found %d", groupId)
	}

	// 处理逻辑
	info.Signed++
	if cfg.Category == SIGN_TYPE_DAILY {
		cardId := h.actor.DutyHandler.GetCurDutyCard()
		history := &cmd.PSignDayInfo{Params: []int32{cardId, param}}
		info.Sign = append(info.Sign, history)
	}

	if err = h.SaveDB(); err != nil {
		return err
	}

	// 签到奖励
	reward := make(map[int32]int32)
	excel.GetDailySigninMgr().Foreach(func(rewardCfg *excel.DailySigninCfg) bool {
		if rewardCfg.CalendarId == groupId && rewardCfg.Day == info.Signed {
			reward[rewardCfg.Reward.Key] += rewardCfg.Reward.Val
		}
		return true
	}, true)

	if cfg.Category == SIGN_TYPE_DAILY {
		favor := excel.GetConfigMgr().GetCfg().DUTY_SIGNIN_FAVOR
		reward[favor.Key] += favor.Val
	}

	if len(reward) > 0 {
		_, err = GetDropMgr(h.actor).DropList2(reward, true, nil, commonData, common.CR_Daily_Sign)
		if err != nil {
			return err
		}
	}

	return nil
}

// 刷新签到组数据
func (h *SignHandler) tryRefreshSign(signs map[int32]*cmd.PCommonSignInfo) error {
	b := false
	now := time.Now()
	// 尝试去除过期的签到数据
	for k, info := range signs {
		if info.End <= now.Unix() {
			delete(signs, k)
			b = true
		}
	}

	// 尝试接取新的签到数据
	excel.GetCalendarsMgr().Foreach(func(cfg *excel.CalendarsCfg) bool {
		// 排除已经存在的
		if _, ok := signs[cfg.Id]; ok {
			return true
		}

		info := h.tryCreateSignInfo(cfg, now)
		if info != nil {
			signs[cfg.Id] = info
			b = true
		}
		return true
	}, true)

	if b {
		if err := h.SaveDB(); err != nil {
			return err
		}
	}

	return nil
}

func (h *SignHandler) tryCreateSignInfo(cfg *excel.CalendarsCfg, now time.Time) *cmd.PCommonSignInfo {

	// 按类型处理
	if cfg.Category == SIGN_TYPE_DAILY || cfg.Category == SIGN_TYPE_ACTIVITY {
		start, err := common.ParseDate(cfg.StartTime)
		if err != nil {
			h.Warn(err)
			return nil
		}
		end, err := common.ParseDate(cfg.EndTime)
		if err != nil {
			h.Warn(err)
			return nil
		}

		if now.Before(start) || now.After(end) {
			return nil
		}

		return &cmd.PCommonSignInfo{
			GroupId:  cfg.Id,
			NextSign: common.GetNextDailyRefreshTime(),
			Begin:    start.Unix(),
			End:      end.Unix(),
			Signed:   0,
			Sign:     nil,
		}
	}

	if cfg.Category == SIGN_TYPE_NEW_PLAYER || cfg.Category == SIGN_TYPE_OLD_PLAYER {
		// fixme 等需求出来了在处理
	}

	return nil
}

//func (h *SignHandler) dailyRefresh() error {
//	signData := h.actor.GetSignData()
//	err := h.tryRefreshSign(signData.Sign)
//	if err != nil {
//		return err
//	}
//	return nil
//}

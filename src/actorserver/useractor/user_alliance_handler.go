package useractor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"gitee.com/aniwar2/musae/global"
	timeutil "gitee.com/aniwar2/musae/utils/time"
	"strconv"
	"time"
	"unicode/utf8"

	"gitee.com/aniwar2/musae/base"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	"google.golang.org/protobuf/proto"
)

type UserAllianceHandler struct {
	*UABaseHandler
}

func NewUserUserAllianceHandler(actor *UserActor) *UserAllianceHandler {
	h := &UserAllianceHandler{UABaseHandler: NewUABaseHandler(actor, "UserAllianceHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_GetAllianceInfoReq), h.GetAllianceInfoReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CreateAllianceReq), h.CreateAllianceReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_GetAllianceRecommendReq), h.GetAllianceRecommendReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_SearchAllianceReq), h.SearchAllianceReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_JoinAllianceReq), h.JoinAllianceReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_AllianceApplyHandleReq), h.AllianceApplyHandleReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ExitAllianceReq), h.ExitAllianceReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ChangeAllianceInfoReq), h.ChangeAllianceInfoReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ChangeMemberPositionReq), h.ChangeMemberPositionReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_GetAllianceLogReq), h.GetAllianceLogReq)

	return h
}

// Init 初始化模块数据
func (h *UserAllianceHandler) Init() error {
	// 初始化
	h.actor.Data.UserAlliance = &pb.PUserAllianceData{
		Createtime: time.Now().Unix(),
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return err
	}

	logger.Debug("init user alliance data success. player: %s", h.actor.ID())
	return nil
}

func (h *UserAllianceHandler) EnterGame() error {
	return nil
}

func (h *UserAllianceHandler) DailyRefresh() error {
	return nil
}

func (h *UserAllianceHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PUserAllianceData); ok {
		h.actor.Data.UserAlliance = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *UserAllianceHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserAlliance(h.actor.ID()), h.actor.Data.UserAlliance
}

func (h *UserAllianceHandler) tryRefreshWeekDay() error {
	data := h.actor.GetUserAllianceData()

	// 尝试刷新每周登录天数计数
	refreshTime := time.Unix(data.WeekTs, 0)
	now := time.Now()
	if !timeutil.IsCurrentWeek(refreshTime) {
		data.WeekDay = 0
		data.WeekTs = now.Unix()
		return h.SaveDB()
	}
	return nil
}

func (h *UserAllianceHandler) handleAddContribution(typ int32, params []int32) error {
	var add int32
	// var min int32
	// var max int32

	// 处理类型1参数
	// if typ == common.ALLIANCE_CONTRIBUTION_1 {
	// 	data := h.actor.GetUserAllianceData()
	// 	// 同一个联盟多次签到
	// 	signTs := data.SignLog[h.getAllianceId()]
	// 	if common.IsSameDayByOffset(time.Unix(signTs, 0), time.Now(), common.GAME_DAILY_REFRESH_HOUR) {
	// 		return nil
	// 	}
	//
	// 	data.WeekDay++
	// 	data.SignLog[h.getAllianceId()] = time.Now().Unix()
	// 	if err := h.SaveDB(); err != nil {
	// 		return err
	// 	}
	// 	min = data.WeekDay
	// }
	// if typ == common.ALLIANCE_CONTRIBUTION_2 {
	// 	min = params[0]
	// 	max = params[1]
	// }

	// 取增加值
	// excel.GetAllianceExpMgr().Foreach(func(cfg *excel.AllianceExpCfg) bool {
	// 	if cfg.Type != typ {
	// 		return true
	// 	}
	// 	if typ == common.ALLIANCE_CONTRIBUTION_1 {
	// 		if cfg.TypeParm == min {
	// 			add += cfg.Contribution
	// 		}
	// 	}
	// 	if typ == common.ALLIANCE_CONTRIBUTION_2 {
	// 		if min < cfg.TypeParm && cfg.TypeParm <= max {
	// 			add += cfg.Contribution
	// 		}
	// 	}
	// 	return true
	// }, true)

	// 上报到联盟处理
	reqMsg := &pb.S2S_AddContributeReq{AddValue: add, AddType: typ}
	err, _ := h.AllianceInvoke(h.getAllianceId(), int32(pb.Protocols_PS2S_AddContributeReq), reqMsg, nil, "")
	h.Infof("handleAddContribution type:%d, add:%d", typ, add)
	return err
}

func (h *UserAllianceHandler) buildAllianceData(flag bool) *pb.PCommonAllianceInfo {
	data := h.actor.GetUserAllianceData()

	if flag {
		data.RecommendTs = 0
		if err := h.SaveDB(); err != nil {
			h.Error(err)
		}
	}

	userAllianceInfo := h.buildUserAllianceInfo(nil)
	// 没联盟，给个人数据
	if data.AllianceId == 0 {
		return &pb.PCommonAllianceInfo{User: userAllianceInfo}
	}

	info, err := h.getAllianceInfo(data.AllianceId)
	if err != nil {
		h.Error(err)
		return nil
	}
	info.User = userAllianceInfo
	return info
}

func (h *UserAllianceHandler) getAllianceInfo(allianceId int64) (*pb.PCommonAllianceInfo, error) {
	reqMsg := &pb.S2S_GetAllianceInfoReq{}
	rspData := &pb.S2S_GetAllianceInfoRes{}
	err, _ := h.AllianceInvoke(allianceId, int32(pb.Protocols_PS2S_GetAllianceInfoReq), reqMsg, rspData, "")
	if err != nil {
		return nil, err
	}
	return rspData.Info, nil
}

func (h *UserAllianceHandler) toAllianceBaseInfo(base *pb.PServerAllianceBaseInfo) (*pb.PCommonAllianceBaseInfo, error) {
	info, err := h.actor.getRoleBaseDataByRoleId(base.LeaderId)
	if err != nil {
		return nil, err
	}
	return &pb.PCommonAllianceBaseInfo{
		Id:             base.Id,
		Name:           base.Name,
		Profile:        base.Profile,
		Notice:         base.Notice,
		LogoId:         base.LogoId,
		Level:          base.Level,
		Exp:            base.Exp,
		MemberNum:      base.MemberNum,
		WeekContribute: base.WeekContribute,
		LeaderName:     info.Common.RoleName,
	}, nil
}

func (h *UserAllianceHandler) buildUserAllianceInfo(alliance *pb.PCommonAllianceInfo) *pb.PUserAllianceInfo {
	data := h.actor.GetUserAllianceData()
	user := &pb.PUserAllianceInfo{
		AllianceId:  data.AllianceId,
		RecommendTs: data.RecommendTs,
		AllianceTs:  data.AllianceTs,
		JoinTs:      data.JoinTs,
	}
	if alliance != nil {
		alliance.User = user
	}
	return user
}

func (h *UserAllianceHandler) GetAllianceRecommendReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_GetAllianceRecommendReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	data := h.actor.GetUserAllianceData()
	// 是否有联盟
	if data.AllianceId > 0 {
		return nil, fmt.Errorf("illegal operation"), int32(pb.ErrorCode_IllegalOperationError)
	}

	// cd判定
	now := time.Now().Unix()
	if now <= data.RecommendTs {
		return nil, fmt.Errorf("operation cd"), int32(pb.ErrorCode_Alliance_recommend_cd)
	}
	if data.RecommendTs == 0 {
		data.RecommendTs = 1 // 首次打开界面自动刷新一次
	} else {
		// data.RecommendTs = now + int64(excel.GetAllianceParmMgr().GetById(1).AllianceParm)
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 拉取列表
	retList := h.getRecommendList()
	return &pb.LS2C_GetAllianceRecommendRes{List: retList, RecommendTs: data.RecommendTs}, nil, 0
}

func (h *UserAllianceHandler) getRecommendList() []*pb.PCommonAllianceBaseInfo {
	// hitSize := int(0)
	// infos := make([]*pb.PCommonAllianceBaseInfo, 0)
	//
	// err, hitData := h.actor.Srv.ESMultiSearch(common.ES_ALLIANCE_BASE_KEY, nil, nil, nil, hitSize, true)
	// if err != nil {
	// 	h.Errorf("es查询出错了: %v", err)
	// 	return infos
	// }
	// for _, hit := range hitData.Hits {
	// 	temp := &pb.PServerAllianceBaseInfo{}
	// 	if err = json.Unmarshal(hit.Source_, temp); err != nil {
	// 		continue
	// 	}
	// 	baseInfo, err := h.toAllianceBaseInfo(temp)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	infos = append(infos, baseInfo)
	// }
	// return infos
	return nil
}

func (h *UserAllianceHandler) SearchAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_SearchAllianceReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	baseInfo, err := h.getAllianceByName(req.Name)
	if err != nil || baseInfo == nil {
		return nil, err, int32(pb.ErrorCode_Not_found_alliance)
	}

	return &pb.LS2C_SearchAllianceRes{Base: baseInfo}, nil, 0
}

func (h *UserAllianceHandler) JoinAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_JoinAllianceReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 校验
	data := h.actor.GetUserAllianceData()
	if data.AllianceId > 0 {
		return nil, fmt.Errorf("exist alliance"), int32(pb.ErrorCode_Had_exist_alliance)
	}
	// cd中
	if time.Now().Unix() < data.JoinTs {
		return nil, fmt.Errorf("join alliance cd"), int32(pb.ErrorCode_Alliance_join_cd)
	}
	_, err = h.checkAllianceId(req.AllianceId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_Not_found_alliance)
	}

	// 申请加入
	reqMsg := &pb.S2S_JoinAllianceReq{}
	rspData := &pb.S2S_JoinAllianceRes{}
	err, code = h.AllianceInvoke(req.AllianceId, int32(pb.Protocols_PS2S_JoinAllianceReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}
	if rspData.ErrCode > 0 {
		return nil, fmt.Errorf("join alliance failed"), rspData.ErrCode
	}
	return &pb.LS2C_JoinAllianceRes{}, nil, 0
}

func (h *UserAllianceHandler) AllianceApplyHandleReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_AllianceApplyHandleReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 校验
	data := h.actor.GetUserAllianceData()
	if data.AllianceId <= 0 {
		return nil, fmt.Errorf("not in alliance"), int32(pb.ErrorCode_Not_in_alliance)
	}

	// 提交到联盟
	reqMsg := &pb.S2S_AllianceApplyHandleReq{
		RoleIds: req.RoleIds,
		IsAgree: req.IsAgree,
	}
	rspData := &pb.S2S_AllianceApplyHandleRes{}
	err, code = h.AllianceInvoke(data.AllianceId, int32(pb.Protocols_PS2S_AllianceApplyHandleReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}
	h.actor.comData.GetAllianceData().Members = append(h.actor.comData.GetAllianceData().Members, rspData.Members...)
	h.actor.comData.GetAllianceData().Base = rspData.Base
	res := &pb.LS2C_AllianceApplyHandleRes{
		CommonData: h.actor.comData.FixDownComData(),
		IsAgree:    req.IsAgree,
		Result:     rspData.Result,
	}
	return res, nil, 0
}

func (h *UserAllianceHandler) ExitAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_ExitAllianceReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 校验
	data := h.actor.GetUserAllianceData()
	if data.AllianceId <= 0 {
		return nil, fmt.Errorf("not in alliance"), int32(pb.ErrorCode_Not_in_alliance)
	}

	// 提交到联盟
	reqMsg := &pb.S2S_ExitAllianceReq{ExitType: req.ExitType, TargetId: req.TargetId}
	rspData := &pb.S2S_ExitAllianceRes{}
	err, code = h.AllianceInvoke(data.AllianceId, int32(pb.Protocols_PS2S_ExitAllianceReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}
	// 主动退出后续处理
	if req.ExitType == 1 {
		h.HandleExitAlliance(1)
	}
	rsp := &pb.LS2C_ExitAllianceRes{
		ExitType: req.ExitType,
		TargetId: req.TargetId,
		Base:     rspData.Base,
		Members:  rspData.Members,
	}
	return rsp, nil, 0
}

// 加入联盟的后续处理逻辑
func (h *UserAllianceHandler) afterJoinAlliance() {
	// 在线自动签到
	data := h.actor.GetUserData()
	if data.Common.OfflineTime == -1 {
		h.tryRefreshWeekDay()
		userAlliance := h.actor.GetUserAllianceData()
		userAlliance.WeekDay++
		userAlliance.SignLog[h.getAllianceId()] = time.Now().Unix()
	}
}

func (h *UserAllianceHandler) GetAllianceLogReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_GetAllianceLogReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 校验
	data := h.actor.GetUserAllianceData()
	if data.AllianceId <= 0 {
		return nil, fmt.Errorf("not in alliance"), int32(pb.ErrorCode_Not_in_alliance)
	}

	// 提交到联盟
	reqMsg := &pb.S2S_GetAllianceLogReq{Page: req.Page}
	rspData := &pb.S2S_GetAllianceLogRes{}
	err, code = h.AllianceInvoke(data.AllianceId, int32(pb.Protocols_PS2S_GetAllianceLogReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}
	return &pb.LS2C_GetAllianceLogRes{Page: req.Page, Logs: rspData.Logs}, nil, 0
}

func (h *UserAllianceHandler) ChangeMemberPositionReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_ChangeMemberPositionReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 校验
	data := h.actor.GetUserAllianceData()
	if data.AllianceId <= 0 {
		return nil, fmt.Errorf("not in alliance"), int32(pb.ErrorCode_Not_in_alliance)
	}
	if req.TargetId == h.actor.roleId {
		return nil, fmt.Errorf("param error"), int32(pb.ErrorCode_IllegalOperationError)
	}

	// 提交到联盟
	reqMsg := &pb.S2S_ChangeMemberPositionReq{
		TargetId:   req.TargetId,
		PositionId: req.PositionId,
	}
	rspData := &pb.S2S_ChangeMemberPositionRes{}
	err, code = h.AllianceInvoke(data.AllianceId, int32(pb.Protocols_PS2S_ChangeMemberPositionReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}

	rsp := &pb.LS2C_ChangeMemberPositionRes{
		Base:    rspData.Info.Base,
		Members: rspData.Info.Members,
	}
	return rsp, nil, 0
}

func (h *UserAllianceHandler) ChangeAllianceInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}
	var req pb.C2LS_ChangeAllianceInfoReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	// 校验
	data := h.actor.GetUserAllianceData()
	if data.AllianceId <= 0 {
		return nil, fmt.Errorf("not in alliance"), int32(pb.ErrorCode_Not_in_alliance)
	}
	infos := map[string]string{}
	if req.EditType == 1 {
		infos["profile"] = req.Profile
	} else if req.EditType == 2 {
		infos["notice"] = req.Notice
	} else if req.EditType == 3 {
		// headCfg := excel.GetAllianceHeadMgr().GetById(req.LogoId)
		// if headCfg == nil {
		// 	return nil, fmt.Errorf("logo id illegal %v", req.LogoId), int32(pb.ErrorCode_ParamError)
		// }
	} else {
		return nil, fmt.Errorf("edit type illegal"), int32(pb.ErrorCode_ParamError)
	}
	err, code = h.commonInfoCheck(infos)
	if err != nil || code != pb.ErrorCode_Success {
		return nil, err, int32(code)
	}

	// 提交到联盟
	reqMsg := &pb.S2S_ChangeAllianceInfoReq{
		EditType: req.EditType,
		Profile:  req.Profile,
		Notice:   req.Notice,
		LogoId:   req.LogoId,
	}
	rspData := &pb.S2S_ChangeAllianceInfoRes{}
	err, code = h.AllianceInvoke(data.AllianceId, int32(pb.Protocols_PS2S_ChangeAllianceInfoReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}
	h.actor.comData.GetAllianceData().Base = rspData.Info.Base
	return &pb.LS2C_ChangeAllianceInfoRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// 联盟信息通用check
func (h *UserAllianceHandler) commonInfoCheck(infos map[string]string) (error, pb.ErrorCode) {
	// 名称校验
	name, ok := infos["name"]
	if ok {
		nameLen := utf8.RuneCountInString(name)
		if nameLen == 0 || nameLen > 8 {
			return fmt.Errorf("name length is illegal"), pb.ErrorCode_Alliance_name_illegal
		}
		if h.actor.Srv.CheckSpecialLetters(name, false) {
			return fmt.Errorf("name is illegal"), pb.ErrorCode_Alliance_name_illegal
		}
		check, err := h.actor.Srv.CheckSensitiveWord(common.CHECK_TYPE_PLAYERNAME, name)
		if err != nil || !check {
			return err, pb.ErrorCode_Alliance_name_illegal
		}
		if h.checkAllianceName(name) {
			return err, pb.ErrorCode_Alliance_name_exist
		}
	}

	// 简介校验
	profile, ok := infos["profile"]
	if ok {
		if utf8.RuneCountInString(profile) > 50 {
			return fmt.Errorf("profile length is illegal"), pb.ErrorCode_Alliance_profile_illegal
		}
		if profile != "" {
			if h.actor.Srv.CheckSpecialLetters(profile, true) {
				return fmt.Errorf("profile is illegal"), pb.ErrorCode_Alliance_profile_illegal
			}
			check, err := h.actor.Srv.CheckSensitiveWord(common.CHECK_TYPE_PLAYERNAME, profile)
			if err != nil || !check {
				return err, pb.ErrorCode_Alliance_profile_illegal
			}
		}
	}

	// 公告校验
	notice, ok := infos["notice"]
	if ok {
		if utf8.RuneCountInString(notice) > 80 {
			return fmt.Errorf("notice length is illegal"), pb.ErrorCode_Alliance_notice_illegal
		}
		if notice != "" {
			if h.actor.Srv.CheckSpecialLetters(notice, true) {
				return fmt.Errorf("notice is illegal"), pb.ErrorCode_Alliance_notice_illegal
			}
			check, err := h.actor.Srv.CheckSensitiveWord(common.CHECK_TYPE_PLAYERNAME, notice)
			if err != nil || !check {
				return err, pb.ErrorCode_Alliance_notice_illegal
			}
		}
	}

	return nil, 0
}

func (h *UserAllianceHandler) GetAllianceInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}

	var req pb.C2LS_GetAllianceInfoReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// cd判定
	data := h.actor.GetUserAllianceData()
	now := time.Now().Unix()
	if now < data.AllianceTs {
		return &pb.LS2C_GetAllianceInfoRes{}, nil, 0 // 隐式cd，不给错误码
	}

	data.AllianceTs = now + 15
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}
	return &pb.LS2C_GetAllianceInfoRes{Alliance: h.buildAllianceData(false)}, nil, 0
}

func (h *UserAllianceHandler) CreateAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_ALLIANCE)
	if err != nil {
		return nil, err, int32(code)
	}

	var req pb.C2LS_CreateAllianceReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	data := h.actor.GetUserAllianceData()
	if data.AllianceId > 0 {
		return nil, fmt.Errorf("exist alliance"), int32(pb.ErrorCode_Had_exist_alliance)
	}
	// logo id校验
	// headCfg := excel.GetAllianceHeadMgr().GetById(req.LogoId)
	// if headCfg == nil {
	// 	return nil, fmt.Errorf("logo id illegal %v", req.LogoId), int32(pb.ErrorCode_ParamError)
	// }

	infos := map[string]string{
		"name":    req.Name,
		"profile": req.Profile,
	}
	err, code = h.commonInfoCheck(infos)
	if err != nil || code != pb.ErrorCode_Success {
		return nil, err, int32(code)
	}

	// cost := excel.GetConfigMgr().GetCfg().ALLIANCE_CREATE_COST
	// if !GetConsumeMgr(h.actor).CheckMapEnough(map[int32]int32{cost[0]: cost[1]}) {
	// 	return nil, fmt.Errorf("currency not enough"), int32(pb.ErrorCode_CurrencyNotEnough)
	// }

	// 创建联盟数据
	allianceId := guid.GenIntUuid()
	if allianceId == 0 {
		return nil, fmt.Errorf("allianceId generate failed"), int32(pb.ErrorCode_InternalError)
	}
	reqMsg := &pb.S2S_CreateAllianceReq{
		Name:    req.Name,
		Profile: req.Profile,
		LogoId:  req.LogoId,
	}
	rspData := &pb.S2S_CreateAllianceRes{}
	err, code = h.AllianceInvoke(int64(allianceId), int32(pb.Protocols_PS2S_CreateAllianceReq), reqMsg, rspData, in.GetTopic())
	if err != nil {
		return nil, err, int32(code)
	}

	// 保存玩家联盟个人数据
	data.AllianceId = int64(allianceId)
	h.afterJoinAlliance()
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	err = GetConsumeMgr(h.actor).ConsumeList(map[int32]int32{}, h.actor.comData, common.CR_ALLIANCE_CREATE)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	h.buildUserAllianceInfo(rspData.Info)
	h.actor.comData.Data.Alliance = rspData.Info
	return &pb.LS2C_CreateAllianceRes{CommonData: h.actor.comData.FixDownComData()}, nil, int32(pb.ErrorCode_Success)
}

// 检查给定联盟id是否存在
func (h *UserAllianceHandler) checkAllianceId(allianceId int64) (*pb.PServerAllianceBaseInfo, error) {
	// err, info := h.actor.Srv.ESGet(common.ES_ALLIANCE_BASE_KEY, strconv.Itoa(int(allianceId)))
	// if err != nil {
	// 	return nil, err
	// }

	// data := &pb.PServerAllianceBaseInfo{}
	// if err = json.Unmarshal(info, data); err != nil {
	// 	return nil, err
	// }
	// return data, nil
	return nil, nil
}

// 检查给定联盟名称是否使用
func (h *UserAllianceHandler) checkAllianceName(name string) bool {
	info, err := h.getAllianceByName(name)
	if err != nil {
		return true
	}
	return info != nil
}

func (h *UserAllianceHandler) getAllianceByName(name string) (*pb.PCommonAllianceBaseInfo, error) {
	// 索引不存在
	// if !h.actor.Srv.ESCheckIndex(common.ES_ALLIANCE_BASE_KEY) {
	// 	return nil, nil
	// }

	// matchMap := map[string]string{"name.keyword": name}
	// err, hitData := h.actor.Srv.ESMultiSearch(common.ES_ALLIANCE_BASE_KEY, matchMap, nil, nil, 1, false)
	// if err != nil {
	// 	h.Errorf("es查询出错了: %v", err)
	// 	return nil, err
	// }
	// var baseInfo *pb.PCommonAllianceBaseInfo
	// for _, hit := range hitData.Hits {
	// 	temp := &pb.PServerAllianceBaseInfo{}
	// 	if err = json.Unmarshal(hit.Source_, temp); err != nil {
	// 		h.Error(err)
	// 		continue
	// 	}
	// 	baseInfo, err = h.toAllianceBaseInfo(temp)
	// 	if err != nil {
	// 		h.Error(err)
	// 	}
	// }
	// h.Debugf("联盟名称查询结果: %+v", baseInfo)
	// return baseInfo, nil
	return nil, nil
}

func (h *UserAllianceHandler) getAllianceId() int64 {
	return h.actor.GetUserAllianceData().AllianceId
}

func (h *UserAllianceHandler) HandleJoinAlliance(alliance *pb.PCommonAllianceInfo) error {
	data := h.actor.GetUserAllianceData()
	data.AllianceId = alliance.Base.Id
	data.ApplyOutdateTs = time.Now().Unix()
	h.pushChangeAllianceIdNtf(alliance)
	h.afterJoinAlliance()
	return h.SaveDB()
}

// 离线事件处理：被踢出联盟
func (h *UserAllianceHandler) HandleExitAlliance(exitType int32) error {
	data := h.actor.GetUserAllianceData()
	data.AllianceId = 0
	data.RecommendTs = 0
	// 踢出没有cd
	if exitType == 1 {
		// cfg := excel.GetAllianceParmMgr().GetById(7)
		// if cfg != nil {
		// 	data.JoinTs = time.Now().Add(time.Hour * time.Duration(cfg.AllianceParm)).Unix()
		// }
	}
	h.Debug("HandleExitAlliance success")
	if exitType == 2 {
		h.pushChangeAllianceIdNtf(nil)
	}
	h.afterExitAlliance()
	return h.SaveDB()
}

// 退出联盟的后续处理逻辑
func (h *UserAllianceHandler) afterExitAlliance() {
	// 清理签到进度
	data := h.actor.GetUserAllianceData()
	data.WeekDay = 0
	data.SignLog = make(map[int64]int64)
}

// 封装联盟actor调用方法
func (h *UserAllianceHandler) AllianceInvoke(allianceId int64, msgId int32, reqMsg proto.Message, rspData proto.Message, topic string) (error, pb.ErrorCode) {
	callData, err := proto.Marshal(reqMsg)
	if err != nil {
		return err, pb.ErrorCode_SerializeError
	}
	protoMsg := &base.ProtoMsg{
		MsgId:  msgId,
		AppId:  global.ACTOR_SVC,
		UserId: h.actor.uid,
		RoleId: h.actor.roleId,
		UAID:   h.actor.ID(),
		Data:   callData,
		// GUID:    utils.GenIntUUID(),
		Topic:        topic,
		ServerReqIdx: guid.GenIntUuid(),
	}
	rspMsg, err := h.actor.Srv.ActorInvoke(global.AllianceActorType, strconv.Itoa(int(allianceId)), protoMsg)
	if err != nil {
		h.Error(err)
	}
	if rspMsg.ErrCode > 0 {
		return err, pb.ErrorCode(rspMsg.ErrCode)
	}

	// 返回数据解析
	if rspData != nil {
		err = proto.Unmarshal(rspMsg.Data, rspData)
		if err != nil {
			return err, pb.ErrorCode_DeSerializeError
		}
	}
	return nil, pb.ErrorCode_Success
}

// 推送玩家联盟id变更信息
func (h *UserAllianceHandler) pushChangeAllianceIdNtf(alliance *pb.PCommonAllianceInfo) {
	user := h.buildUserAllianceInfo(alliance)
	if alliance == nil {
		alliance = &pb.PCommonAllianceInfo{User: user}
	}
	ntf := &pb.LS2C_ChangeAllianceIdNtf{
		Alliance: alliance,
	}

	err := h.actor.Srv.Send2Gate(h.actor.ID(), h.actor.UserMap[h.actor.GetUID()], ntf)
	if err != nil {
		h.Error(err)
		return
	}
	h.Infof("玩家联盟id变化了 info: %+v", ntf)
}

func (h *UserAllianceHandler) PushTopic2Alliance(opt pb.GateTopicOperator, topic string) {

	allianceId := h.getAllianceId()
	if allianceId <= 0 {
		return
	}
	req := &pb.S2S_BindMemberGateTopicReq{
		Opt:    opt,
		Uid:    h.actor.GetUID(),
		GateId: topic,
	}
	res := &pb.S2S_BindMemberGateTopicRes{}
	err, _ := h.actor.UserAllianceHandler.AllianceInvoke(int64(allianceId), int32(pb.Protocols_PS2S_BindMemberGateTopicReq), req, res, topic)
	if err != nil {
		h.Debugf("玩家[%d],向allianceActor,转发消息失败:[%v]", allianceId, err)
	}
}

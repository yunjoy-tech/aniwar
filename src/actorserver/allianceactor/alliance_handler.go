package allianceactor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/yunjoy-tech/aniwar/src/common/db"
	"github.com/yunjoy-tech/aniwar/src/common/gmeta"
	"github.com/yunjoy-tech/aniwar/src/meta"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/service"
	"github.com/yunjoy-tech/musae/utils"
	"google.golang.org/protobuf/proto"
	"strconv"
	"time"
)

type AllianceHandler struct {
	*USBaseHandler
	wChan chan interface{} // 输出管道
}

type AllianceMessage struct {
	AllianceId int64
	Message    *pb.BroadMessage
}

func NewAllianceMessage(allianceId int64, message *pb.BroadMessage) *AllianceMessage {
	return &AllianceMessage{
		AllianceId: allianceId,
		Message:    message,
	}
}

func NewAllianceHandler(actor *AllianceActor) *AllianceHandler {
	h := &AllianceHandler{USBaseHandler: NewUSBaseHandler(actor, "AllianceHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_CreateAllianceReq), h.CreateAllianceReq)             // 创建联盟
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_JoinAllianceReq), h.JoinAllianceReq)                 // 加入联盟
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_ChangeAllianceInfoReq), h.ChangeAllianceInfoReq)     // 修改联盟信息
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_AllianceApplyHandleReq), h.AllianceApplyHandleReq)   // 审核申请
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_ChangeMemberPositionReq), h.ChangeMemberPositionReq) // 修改职位
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_GetAllianceLogReq), h.GetAllianceLogReq)             // 拉取日志
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_ExitAllianceReq), h.ExitAllianceReq)                 // 退出（踢出）联盟

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_GetAllianceInfoReq), h.GetAllianceInfoReq)           // 拉取联盟基本信息
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_AddContributeReq), h.AddContributeReq)               // 增加联盟贡献度
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_BindMemberGateTopicReq), h.BindMemberGateTopicReq)   // 绑定/解绑成员Topic
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_SendMessage2AllianceReq), h.SendMessage2AllianceReq) // 向联盟发送消息
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_GetAllianceMessageReq), h.GetAllianceMessageReq)     // 获取联盟聊天消息

	return h
}

// Init 初始化模块数据
func (h *AllianceHandler) Init() error {
	return nil
}

func (h *AllianceHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PServerAllianceInfo); ok {
		h.actor.Data = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}
	return nil
}

func (h *AllianceHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyAllianceData(h.actor.ID()), h.actor.Data
}

func (h *AllianceHandler) EnterGame() error {
	// 聊天消息异步存ES
	h.ProcessMessage()
	return nil
}

func (h *AllianceHandler) DailyRefresh() error {
	return nil
}

func (h *AllianceHandler) GetAllianceInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	return &pb.S2S_GetAllianceInfoRes{Info: h.toCommonAllianceInfo(h.actor.Data)}, nil, int32(pb.ErrorCode_Success)
}

func (h *AllianceHandler) MessageWrite2Channel(allianceId int64, message *pb.BroadMessage) {
	chatMessage := NewAllianceMessage(allianceId, message)
	h.wChan <- chatMessage
	h.Debug("chat write message to channel:", allianceId, message)
}

func (h *AllianceHandler) ProcessMessage() {

	h.wChan = make(chan interface{}, 2048)
	utils.GoSafeRun(func() {
		for {
			select {
			case tmpMessage, _ := <-h.wChan:
				if message, ok := tmpMessage.(*AllianceMessage); ok {
					if err := h.SaveAllianceChatMessage(message.AllianceId, message.Message); err != nil {
						h.Debug("联盟聊天信息存储ES失败", err)
					}
				} else {
					h.Debugf("联盟聊天信息数据类型输错")
				}
			}
		}
	}, nil)
}

func (h *AllianceHandler) AddContributeReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_AddContributeReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	err, code := h.handleAddContribute(req.AddType, req.AddValue, in.RoleId)
	if err != nil {
		return nil, err, code
	}
	return &pb.S2S_AddContributeRes{}, nil, int32(pb.ErrorCode_Success)
}

func (h *AllianceHandler) handleAddContribute(addType, addValue int32, targetId uint64) (error, int32) {
	if addType == 1 {
		h.addAllianceLog(targetId, pb.AllianceLogType_LogType_Sign)
	}
	if addValue == 0 {
		if err := h.SaveDB(true); err != nil {
			return err, int32(pb.ErrorCode_SaveDBError)
		}
		return nil, 0
	}
	// 是否领取过签到奖励
	data := h.GetAllianceData()
	_, ok := data.Base.SignLog[targetId]
	if addType == 1 && ok {
		if err := h.SaveDB(true); err != nil {
			return err, int32(pb.ErrorCode_SaveDBError)
		}
		return nil, 0
	}
	data.Base.SignLog[targetId] = 0

	// 增加个人贡献度
	member, ok := data.Member[targetId]
	if ok {
		member.Contribute += addValue
	}

	// 联盟总贡献度/经验
	h.handleAddTotalContribute(addValue)

	if err := h.SaveDB(true); err != nil {
		return err, int32(pb.ErrorCode_SaveDBError)
	}
	return nil, 0
}

func (h *AllianceHandler) SendMessage2AllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_SendMessage2AllianceReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 判断发送者是否是联盟成员
	if !h.IsAllianceMember(req.Message.FromRoleId) {
		return nil, errors.New("member is not found"), int32(pb.ErrorCode_Member_not_found)
	}
	// 存储联盟消息
	data := h.GetAllianceData()
	h.MessageWrite2Channel(data.Base.Id, req.GetMessage())

	if err := h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 向联盟在想成员推送消息
	h.BroadcastMessages(req.GetMessage(), req.Message.FromRoleId, pb.ChatChannel_Channel_alliance)

	return &pb.S2S_SendMessage2AllianceRes{}, nil, int32(pb.ErrorCode_Success)

}

// GetAllianceMessageReq 获取联盟消息
func (h *AllianceHandler) GetAllianceMessageReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_GetAllianceMessageReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	// 判断发送者是否是联盟成员
	if !h.IsAllianceMember(uint64(req.GetRoleId())) {
		return nil, errors.New("member is not found"), int32(pb.ErrorCode_Member_not_found)
	}

	// 获取联盟消息
	data := h.GetAllianceData()
	message := h.GetAllianceChatMessage(data.Base.GetId(), time.Now().Unix(), req.GetFromSize(), req.GetSize())
	for _, m := range message {
		baseInfo, err := h.actor.getRoleBaseDataByRoleId(m.FromRoleId)
		if err != nil {
			continue
		}
		// 获取发送消息的玩家信息
		info := &pb.AllianceMemInfo{
			RoleName: baseInfo.Common.RoleName,
			RoleHead: baseInfo.Common.RoleHead,
		}
		temp, _ := json.Marshal(info)
		m.Data = append(m.Data, string(temp))
	}
	// 绑定Topic
	h.actor.AddGateTopic(in.GetTopic(), in.GetUserId())
	h.Debugf("alliance addGateTopic[%s],user[%s]", in.GetTopic(), in.GetUserId())
	res := &pb.S2S_GetAllianceMessageRes{
		Message: message,
	}

	return res, nil, int32(pb.ErrorCode_Success)
}

// BindMemberGateTopicReq 联盟成员上线下线更新Topic
func (h *AllianceHandler) BindMemberGateTopicReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_BindMemberGateTopicReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	if !h.IsAllianceMember(in.GetRoleId()) {
		return nil, errors.New("member is not found"), int32(pb.ErrorCode_Member_not_found)
	}

	switch req.Opt {
	case pb.GateTopicOperator_GTO_bind: // 建立绑定

		h.actor.AddGateTopic(req.GetGateId(), req.GetUid())
		h.Infof("alliance addGateTopic[%s],user[%s]", req.GetGateId(), req.GetUid())
	case pb.GateTopicOperator_GTO_unbound: // 解除绑定
		h.actor.DelGateTopic(req.GetUid())
		h.Infof("alliance delGateTopic[%s],user[%s]", req.GetGateId(), req.GetUid())

	default:
		h.Warnf("BindMemberGateTopicReq, 未支持的操作类型: %+v", req.GetOpt())
	}
	h.Debugf("玩家登录或离线, 更新topic, uaid:%s, req:%+v", req.GetUid(), req)
	return &pb.S2S_BindMemberGateTopicRes{}, nil, int32(pb.ErrorCode_Success)
}

// 联盟增加总贡献度
func (h *AllianceHandler) handleAddTotalContribute(addValue int32) {
	data := h.GetAllianceData()
	// 增加总贡献度
	data.Base.WeekContribute += addValue

	// 增加总经验 暂定1:1
	data.Base.Exp += addValue

	allianceTable := gmeta.GetMetaMgr().AllianceTable
	// 尝试升级
	var maxLevel int32
	for _, meta := range allianceTable.GetDataList() {
		if meta.Id > maxLevel {
			maxLevel = meta.Id
		}
	}

	for i := data.Base.Level; i < maxLevel; i++ {
		cfg := allianceTable.Get(i)
		if cfg == nil {
			break
		}
		if data.Base.Exp < cfg.UpgradeCost {
			break
		}
		data.Base.Exp -= cfg.UpgradeCost
		data.Base.Level++
	}
	h.tryUploadToES()
}

func (h *AllianceHandler) ExitAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_ExitAllianceReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	data := h.GetAllianceData()
	if req.ExitType == 1 {
		h.exitAlliance(req.TargetId, req.ExitType, 0, in.GetUserId())
	} else if req.ExitType == 2 {
		// 权限判定
		if !h.checkPermission(in.RoleId, MEMBER_PERMISSION_KICKOUT) {
			return nil, fmt.Errorf("member permission illegal"), int32(pb.ErrorCode_Member_permission_illegal)
		}
		// 是否级别足够
		mine := data.Member[in.RoleId]
		target := data.Member[req.TargetId]
		if mine == nil || target == nil {
			return nil, fmt.Errorf("member not found"), int32(pb.ErrorCode_Member_not_found)
		}
		if target.Position >= mine.Position {
			return nil, fmt.Errorf("illegal operation"), int32(pb.ErrorCode_Member_permission_illegal)
		}

		// 踢出
		h.exitAlliance(req.TargetId, req.ExitType, in.RoleId, in.GetUserId())
	} else {
		return nil, fmt.Errorf("unrealized exit type %d", req.ExitType), int32(pb.ErrorCode_UnrealizedTypeError)
	}

	if err = h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.S2S_ExitAllianceRes{
		Base:    h.toAllianceBaseInfo(data.Base),
		Members: h.toCommonAllianceMembers(data.Member),
	}
	return rsp, nil, 0
}

func (h *AllianceHandler) GetAllianceLogReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_GetAllianceLogReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	data := h.GetAllianceData()
	// 按页取数据
	allianceTable := gmeta.GetMetaMgr().ParamTable
	size := allianceTable.Get(4).AllianceParm
	start := (req.Page - 1) * size
	end := req.Page * size
	if int32(len(data.Log)) < end {
		end = int32(len(data.Log))
	}
	logs := make([]*pb.PCommonAllianceLog, 0)
	for i := start; i < end; i++ {
		logs = append(logs, data.Log[i])
	}
	return &pb.S2S_GetAllianceLogRes{Logs: logs}, nil, 0
}

func (h *AllianceHandler) ChangeMemberPositionReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_ChangeMemberPositionReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	// 校验
	data := h.GetAllianceData()
	target := data.Member[req.TargetId]
	mine := data.Member[in.RoleId]
	if target == nil || mine == nil {
		return nil, fmt.Errorf("member not found"), int32(pb.ErrorCode_Member_not_found)
	}
	if target.Position == req.PositionId {
		return &pb.S2S_ChangeMemberPositionRes{}, nil, 0
	}

	// 职位有效性判定
	if req.PositionId <= int32(pb.MemberPositionType_None) || req.PositionId >= int32(pb.MemberPositionType_Max) {
		return nil, fmt.Errorf("position illegal"), int32(pb.ErrorCode_ParamError)
	}
	if req.PositionId > mine.Position {
		return nil, fmt.Errorf("illegal operation"), int32(pb.ErrorCode_IllegalOperationError)
	}

	var f bool
	var typ pb.AllianceLogType
	if req.PositionId == int32(pb.MemberPositionType_Leader) {
		// 委任头目
		if !h.checkPermission(in.RoleId, MEMBER_PERMISSION_ABDICATE) {
			return nil, fmt.Errorf("member permission illegal"), int32(pb.ErrorCode_Member_permission_illegal)
		}
		if mine.Position != int32(pb.MemberPositionType_Leader) {
			return nil, fmt.Errorf("current member not leader"), int32(pb.ErrorCode_IllegalOperationError)
		}
		changeLeader(data, req.TargetId, data.Base.LeaderId)
		typ = pb.AllianceLogType_LogType_Promote_Position
		f = true
	} else {
		// 检查职位数量是否足够
		if !h.checkPositionNum(req.PositionId) {
			return nil, fmt.Errorf("position num not enough"), int32(pb.ErrorCode_Position_not_enough)
		}
		// 职位升降调动
		oldPosition := target.Position
		target.Position = req.PositionId

		// 日志记录
		if oldPosition < req.PositionId {
			// 升职了
			typ = pb.AllianceLogType_LogType_Promote_Position
		} else {
			// 降职了
			typ = pb.AllianceLogType_LogType_Reduce_Position
		}
	}
	// 记录日志
	h.addAllianceLog(req.TargetId, typ, strconv.Itoa(int(req.PositionId)))
	if err = h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}
	if f {
		h.tryUploadToES()
	}
	info := &pb.PCommonAllianceInfo{
		Base:    h.toAllianceBaseInfo(data.Base),
		Members: h.toCommonAllianceMembers(data.Member),
	}
	return &pb.S2S_ChangeMemberPositionRes{Info: info}, nil, 0
}

func (h *AllianceHandler) AllianceApplyHandleReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_AllianceApplyHandleReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 权限审核
	if !h.checkPermission(in.RoleId, MEMBER_PERMISSION_EXAMINE) {
		return nil, fmt.Errorf("member permission illegal"), int32(pb.ErrorCode_Member_permission_illegal)
	}
	// 审核列表不存在
	data := h.GetAllianceData()
	roleIds := make(map[uint64]int64)
	if req.RoleIds > 0 {
		if _, ok := data.Examines[req.RoleIds]; !ok {
			return nil, fmt.Errorf("examine not exist %d", req.RoleIds), int32(pb.ErrorCode_ParamError)
		}
		roleIds[req.RoleIds] = data.Examines[req.RoleIds]
	} else {
		roleIds = data.Examines
	}

	var f bool
	var ftype int32
	success := make([]uint64, 0)
	delIds := make([]uint64, 0)
	unhandles := make([]uint64, 0)
	outdate := make([]uint64, 0)
	member := make([]*pb.PCommonAllianceMember, 0)
	if req.IsAgree {
		// 同意
		for id, ts := range roleIds {
			// 是否已满
			if len(data.Member) >= h.getMaxMember() {
				ftype = 2
				unhandles = append(unhandles, id)
				continue
			}
			// 是否已经在其他联盟
			reqMsg := &pb.S2S_CheckInAllianceReq{}
			rspMsg := &pb.S2S_CheckInAllianceRes{}
			if err, _ = h.actor.Srv.CallUserActor(true, id, int32(pb.Protocols_PS2S_CheckInAllianceReq), reqMsg, rspMsg); err != nil {
				h.Error(err)
				continue
			}
			// 已经有联盟了
			if rspMsg.Exist {
				delete(data.Examines, id)
				ftype = 3
				delIds = append(delIds, id)
				continue
			}
			// 申请过期了
			if ts < rspMsg.Outdate {
				delete(data.Examines, id)
				ftype = 4
				outdate = append(outdate, id)
				continue
			}

			// 通知到玩家
			reqMsg2 := &pb.S2S_JoinAllianceNtf{Alliance: h.toCommonAllianceInfo(data)}
			rspMsg2 := &pb.S2S_JoinAllianceNtf{}
			if err, _ = h.actor.Srv.CallUserActor(true, id, int32(pb.Protocols_PS2S_JoinAllianceNtf), reqMsg2, rspMsg2); err != nil {
				h.Error(err)
				continue
			}
			// 加入
			m := h.joinAlliance(id, int32(pb.MemberPositionType_Member), in)
			member = append(member, h.toCommonAllianceMember(m))

			// 删除申请信息
			delete(data.Examines, id)
			success = append(success, id)
			f = true
			ftype = 1
		}
	} else {
		// 拒绝
		for id := range roleIds {
			delete(data.Examines, id)
			success = append(success, id)
		}
		ftype = 1
	}

	if err = h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}
	if f {
		h.tryUploadToES()
	}
	var result *pb.ApplyResultItem
	if ftype == 1 {
		result = &pb.ApplyResultItem{FType: ftype, RoleIds: success}
	} else if ftype == 2 {
		result = &pb.ApplyResultItem{FType: ftype, RoleIds: unhandles}
	} else if ftype == 3 {
		result = &pb.ApplyResultItem{FType: ftype, RoleIds: delIds}
	} else if ftype == 4 {
		result = &pb.ApplyResultItem{FType: ftype, RoleIds: outdate}
	}

	res := &pb.S2S_AllianceApplyHandleRes{
		Result:  result,
		Members: member,
		Base:    h.toAllianceBaseInfo(data.Base),
	}
	return res, nil, 0
}

func (h *AllianceHandler) ChangeAllianceInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_ChangeAllianceInfoReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	data := h.GetAllianceData()
	if req.EditType == 1 {
		if !h.checkPermission(in.RoleId, MEMBER_PERMISSION_EDIT_PROFILE) {
			return nil, fmt.Errorf("member permission illegal"), int32(pb.ErrorCode_Member_permission_illegal)
		}
		data.Base.Profile = req.Profile
	} else if req.EditType == 2 {
		if !h.checkPermission(in.RoleId, MEMBER_PERMISSION_EDIT_NOTICE) {
			return nil, fmt.Errorf("member permission illegal"), int32(pb.ErrorCode_Member_permission_illegal)
		}
		data.Base.Notice = req.Notice
	} else if req.EditType == 3 {
		if !h.checkPermission(in.RoleId, MEMBER_PERMISSION_EDIT_LOGO) {
			return nil, fmt.Errorf("member permission illegal"), int32(pb.ErrorCode_Member_permission_illegal)
		}
		data.Base.LogoId = req.LogoId
	}

	if err = h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}
	h.tryUploadToES()

	info := &pb.PCommonAllianceInfo{
		Base: h.toAllianceBaseInfo(data.Base),
	}
	return &pb.S2S_ChangeAllianceInfoRes{Info: info}, nil, 0
}

func (h *AllianceHandler) JoinAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	data := h.GetAllianceData()
	// 是否申请过了
	if _, ok := data.Examines[in.RoleId]; ok {
		h.Warnf("had apply alliance. allianceId: %v, roleId: %v", data.Base.Id, in.RoleId)
		return &pb.S2S_JoinAllianceRes{ErrCode: int32(pb.ErrorCode_Had_apply_alliance)}, nil, 0
	}

	// 是否到上限
	paramTable := gmeta.GetMetaMgr().ParamTable
	limit := int(paramTable.Get(6).AllianceParm)
	if len(data.Examines) >= limit {
		h.Warnf("alliance apply limit. allianceId: %v, roleId: %v", data.Base.Id, in.RoleId)
		return &pb.S2S_JoinAllianceRes{ErrCode: int32(pb.ErrorCode_Alliance_apply_limit)}, nil, 0
	}
	h.Debugf("联盟加入上限判定: cur: %v, limit: %v", len(data.Examines), limit)

	// 处理申请
	data.Examines[in.RoleId] = time.Now().Unix()
	if err := h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	return &pb.S2S_JoinAllianceRes{}, nil, 0
}

func (h *AllianceHandler) CreateAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_CreateAllianceReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	allianceId, err := strconv.ParseInt(h.actor.ID(), 10, 64)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	h.Debugf("---------玩家:%s, 创建联盟 %v--------", in.UserId, allianceId)

	h.actor.Data = &pb.PServerAllianceInfo{
		Base: &pb.PServerAllianceBaseInfo{
			Id:             allianceId,
			Name:           req.Name,
			Profile:        req.Profile,
			Notice:         "",
			LogoId:         req.LogoId,
			Level:          1,
			Exp:            0,
			WeekContribute: 0,
			LeaderId:       in.RoleId,
			MemberNum:      0,
		},
		Member:   make(map[uint64]*pb.PAllianceMember),
		Log:      make([]*pb.PCommonAllianceLog, 0),
		Examines: make(map[uint64]int64),
	}
	h.joinAlliance(in.RoleId, int32(pb.MemberPositionType_Leader), in)

	if err = h.SaveDB(true); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}
	h.tryUploadToES()

	return &pb.S2S_CreateAllianceRes{Info: h.toCommonAllianceInfo(h.actor.Data)}, nil, int32(pb.ErrorCode_Success)
}

// 获取联盟最大成员数
func (h *AllianceHandler) getMaxMember() int {
	cfg := h.getAllianceLevelCfg()
	if cfg == nil {
		return 0
	}
	return int(cfg.MemberNum)
}

// 获取联盟等级配置
func (h *AllianceHandler) getAllianceLevelCfg() *meta.AlliancePkgAllianceMeta {
	return gmeta.GetMetaMgr().AllianceTable.Get(h.actor.Data.Base.Level)
}

// IsAllianceMember 是否是联盟成员
func (h *AllianceHandler) IsAllianceMember(roleId uint64) bool {
	data := h.GetAllianceData()
	if _, ok := data.Member[roleId]; ok {
		return true
	}
	return false
}

func (h *AllianceHandler) BroadcastMessages(message *pb.BroadMessage, fromRoleId uint64, channelId pb.ChatChannel) {
	baseInfo, err := h.actor.getRoleBaseDataByRoleId(fromRoleId)
	if err != nil {
		return
	}
	// 获取发送消息的玩家信息
	info := &pb.AllianceMemInfo{
		RoleName: baseInfo.Common.RoleName,
		RoleHead: baseInfo.Common.RoleHead,
	}
	temp, _ := json.Marshal(info)
	message.Data = append(message.Data, string(temp))

	data := h.GetAllianceData()
	h.Infof("联盟成员:%v", data.Member)
	for _, v := range data.Member {
		if v.RoleId == fromRoleId {
			continue
		}
		h.Infof("联盟成员[%d]是否在线[%v]", v.RoleId, h.IsOnline(v.RoleId))
		if h.IsOnline(v.RoleId) { // 在线，发送消息
			if err := h.NotifyMem(fromRoleId, v.RoleId, message, channelId); err != nil {
				h.Infof("联盟向成员[%d]推送消息失败", v.RoleId)
			}
		}
	}
}

func (h *AllianceHandler) NotifyMem(formRoleId, toRoleId uint64, message *pb.BroadMessage, channelId pb.ChatChannel) error {
	baseInfo, err := h.actor.getRoleBaseDataByRoleId(formRoleId)
	if err != nil {
		return err
	}
	notify := &pb.LS2C_NotifyPrivateMessage{
		ChannelId: channelId,
		Message:   message,
		RoleInfo:  baseInfo.Common,
	}
	// 推送到客户端
	uaid, _ := h.actor.Srv.GetUAIDByRoleId(uint64(toRoleId))
	// uaid, err := h.actor.Srv.GetUAIDByRoleId()
	uid, _ := h.actor.Srv.ConvUAID(uaid)
	pid, ok := h.actor.CommonActor.UserMap[uid]
	if !ok {
		h.Infof("alliance --- notify client get pid from UserMap is err:%+v", h.actor.CommonActor.UserMap)
		return errors.New("")
	}
	h.Infof("alliance --- notify client:%+v, gateTopic:%s", h.actor.CommonActor.UserMap, pid.GateId)
	err = h.actor.Srv.Send2Gate(strconv.Itoa(int(toRoleId)), pid, notify)
	if err != nil {
		return err
	}
	return nil
}

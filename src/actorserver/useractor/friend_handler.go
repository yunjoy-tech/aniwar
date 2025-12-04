package useractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type FriendHandler struct {
	*UABaseHandler
}

func NewFriendHandler(actor *UserActor) *FriendHandler {
	h := &FriendHandler{UABaseHandler: NewUABaseHandler(actor, "FriendHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_GetFriendListReq), h.GetFriendListReq)           // 好友数据
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_AddFriendApplyReq), h.AddFriendApplyReq)         // 添加申请
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_FriendApplyHandleReq), h.FriendApplyHandleReq)   // 处理申请（同意/拒绝）
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_DelFriendReq), h.DelFriendReq)                   // 删除
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_GetFriendRecommendReq), h.GetFriendRecommendReq) // 好友推荐
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_BlackOperateReq), h.BlackOperateReq)             // 黑名单操作（加入/移除）
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_GetFriendCampReq), h.GetFriendCampReq)           // 查看露营地
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_OperateFriendPointReq), h.OperateFriendPointReq) // 友情点操作（赠送/收取）

	return h
}

func (h *FriendHandler) Init() error {
	h.actor.Data.FriendData = &cmd.PFriendData{
		Createtime: time.Now().Unix(),
		Friends:    make(map[uint64]int32),
		Applys:     make(map[uint64]int64),
		Examinesx:  make(map[uint64]int64),
		Blacks:     make(map[uint64]int32),
		Sends:      make(map[uint64]int32),
		Receives:   make(map[uint64]int32),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init user friend data success. player: %s", h.actor.ID())
	return nil
}

func (h *FriendHandler) EnterGame() error {
	data := h.actor.GetFriendData()
	h.tryClearExaminesList(data.Examinesx)
	h.tryClearApplyList(data.Applys)
	return h.SaveDB()
}

func (h *FriendHandler) DailyRefresh() error {
	return h.tryRefreshData()
}

func (h *FriendHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PFriendData); ok {
		h.actor.Data.FriendData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *FriendHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserFriend(h.actor.ID()), h.actor.Data.FriendData
}

// 跨天好友相关数据刷新
func (h *FriendHandler) tryRefreshData() error {
	data := h.actor.GetFriendData()
	data.Sends = make(map[uint64]int32)
	data.Receives = make(map[uint64]int32)
	return h.SaveDB()
}

func (h *FriendHandler) buildFriendData(flag bool) *cmd.PClientFriendInfo {
	data := h.actor.GetFriendData()

	// 尝试重置数据
	if flag {
		data.RecommendTs = 0
		data.FriendTs = 0
		if err := h.SaveDB(); err != nil {
			h.Error(err)
		}
	}

	friends := make([]*cmd.PCommonRoleBaseInfo, 0)
	for id := range data.Friends {
		info, err := h.actor.getRoleBaseDataByRoleId(id)
		if err != nil {
			continue
		}
		friends = append(friends, info.Common)
	}

	applys := make([]*cmd.PFriendApplyInfo, 0)
	for id, ts := range data.Applys {
		info, err := h.actor.getRoleBaseDataByRoleId(id)
		if err != nil {
			continue
		}
		applys = append(applys, &cmd.PFriendApplyInfo{
			Info:    info.Common,
			ApplyTs: ts,
		})
	}

	examines := make([]*cmd.PCommonRoleBaseInfo, 0)
	for id := range data.Examinesx {
		info, err := h.actor.getRoleBaseDataByRoleId(id)
		if err != nil {
			continue
		}
		examines = append(examines, info.Common)
	}

	blacks := make([]*cmd.PCommonRoleBaseInfo, 0)
	for id := range data.Blacks {
		info, err := h.actor.getRoleBaseDataByRoleId(id)
		if err != nil {
			continue
		}
		blacks = append(blacks, info.Common)
	}

	sends := make([]uint64, 0)
	for id := range data.Sends {
		sends = append(sends, id)
	}

	receives := make([]uint64, 0)
	var rNum int32
	for id, v := range data.Receives {
		// 过滤一下已经领取的
		if v == 1 {
			rNum++
			continue
		}
		receives = append(receives, id)
	}
	return &cmd.PClientFriendInfo{
		Friends:  friends,
		Applys:   applys,
		Examines: examines,
		Blacks:   blacks,
		Sends:    sends,
		Receives: receives,
		Common: &cmd.PFriendCommonInfo{
			ReceiveNum:  rNum,
			RecommendTs: data.RecommendTs,
			FriendTs:    data.FriendTs,
		},
	}
}

func (h *FriendHandler) GetFriendListReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}

	var req cmd.C2LS_GetFriendListReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// cd判定
	data := h.actor.GetFriendData()
	now := time.Now().Unix()
	if now < data.FriendTs {
		return &cmd.LS2C_GetFriendListRes{}, nil, 0 // 隐式cd，不给错误码
	}

	data.FriendTs = now + 15
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	return &cmd.LS2C_GetFriendListRes{Friends: h.buildFriendData(false)}, nil, 0
}

func (h *FriendHandler) AddFriendApplyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_AddFriendApplyReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 非自己
	if req.RoleId == h.actor.roleId {
		return nil, fmt.Errorf("role id is illegal %d", req.RoleId), int32(cmd.ErrorCode_InvalidParam)
	}

	data := h.actor.GetFriendData()
	h.tryClearApplyList(data.Applys)
	// 已经是好友了
	if _, ok := data.Friends[req.RoleId]; ok {
		return nil, fmt.Errorf("friend exist %d", req.RoleId), int32(cmd.ErrorCode_AlreadyFriendShip)
	}
	// 在黑名单中
	if _, ok := data.Blacks[req.RoleId]; ok {
		return nil, fmt.Errorf("role in black %d", req.RoleId), int32(cmd.ErrorCode_AlreadyInBlack)
	}
	// 好友上限
	if int32(len(data.Friends)) >= excel.GetConfigMgr().GetCfg().FRIEND_MAX_NUM {
		return nil, fmt.Errorf("friend num is limit"), int32(cmd.ErrorCode_SelfFriendNumLimit)
	}
	// 已经申请过了, 判定申请cd
	if _, ok := data.Applys[req.RoleId]; ok {
		return nil, fmt.Errorf("apply exist %d", req.RoleId), int32(cmd.ErrorCode_ApplyFriendCD)
	}
	// 对方是否存在
	info, err := h.actor.getRoleBaseDataByRoleId(req.RoleId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_NotFoundPlayer)
	}
	// 对方好友是否达上限
	friend, err := h.actor.getFriendDataByRoleId(req.RoleId)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_NotFoundPlayer)
	}
	if int32(len(friend.Friends)) >= excel.GetConfigMgr().GetCfg().FRIEND_MAX_NUM {
		return nil, fmt.Errorf("friend num is limit"), int32(cmd.ErrorCode_FriendNumLimit)
	}

	// 发送申请
	reqMsg := &cmd.S2S_AddFriendApplyReq{RoleId: h.actor.roleId}
	err, code = h.actor.Srv.CallUserActor(true, req.RoleId, int32(cmd.Protocols_PS2S_AddFriendApplyReq), reqMsg, nil)
	if err != nil {
		return nil, err, int32(code)
	}

	// 已申请处理
	data.Applys[req.RoleId] = time.Now().Unix() + int64(excel.GetConfigMgr().GetCfg().FRIEND_ADD_CD)
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	h.actor.comData.GetFriendData().Applys = append(h.actor.comData.GetFriendData().Applys, &cmd.PFriendApplyInfo{
		Info:    info.Common,
		ApplyTs: data.Applys[req.RoleId],
	})
	// 返回消息
	return &cmd.LS2C_AddFriendApplyRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// 尝试清理过期申请记录
func (h *FriendHandler) tryClearApplyList(applys map[uint64]int64) {
	now := time.Now().Unix()
	for roleId, ts := range applys {
		if now >= ts {
			delete(applys, roleId)
		}
	}
}

// 尝试清理过期审核列表
func (h *FriendHandler) tryClearExaminesList(examines map[uint64]int64) {
	now := time.Now()
	for roleId, ts := range examines {
		// 申请时间超过30天
		if time.Unix(ts, 0).AddDate(0, 0, 30).Before(now) {
			delete(examines, roleId)
		}
	}
	// 总数量超过上限,从老到新删除
	if int32(len(examines)) > excel.GetConfigMgr().GetCfg().FRIEND_REQUEST_MAX_NUM {
		keys := myUtils.SortMapKeyByVal(examines, myUtils.SORT_ORDER_ASC)
		costNum := len(keys) - int(excel.GetConfigMgr().GetCfg().FRIEND_REQUEST_MAX_NUM)
		for i := 0; i < costNum; i++ {
			delete(examines, keys[i])
		}
	}
}

func (h *FriendHandler) FriendApplyHandleReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_FriendApplyHandleReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	// 审核列表不存在
	data := h.actor.GetFriendData()
	if req.RoleIds > 0 {
		if _, ok := data.Examinesx[req.RoleIds]; !ok {
			return nil, fmt.Errorf("examine not exist %d", req.RoleIds), int32(cmd.ErrorCode_ParamError)
		}
	}

	// 判断处理数据
	var roleIds []uint64
	if req.RoleIds > 0 {
		roleIds = append(roleIds, req.RoleIds)
	} else {
		for k := range data.Examinesx {
			roleIds = append(roleIds, k)
		}
	}

	var allFailed = true         // 全部处理失败的标记，用于给前端返回错误提示
	var errCode int32            // 失败的原因
	success := make([]uint64, 0) // 处理成功的玩家
	if req.IsAgree {
		// 同意
		reqMsg := &cmd.S2S_AgreeFriendApplyReq{RoleId: h.actor.roleId}
		for _, roleId := range roleIds {
			// 判定自己的好友是否上限
			if int32(len(data.Friends)) >= excel.GetConfigMgr().GetCfg().FRIEND_MAX_NUM {
				errCode = int32(cmd.ErrorCode_SelfFriendNumLimit)
				continue
			}
			// 对方好友是否上限
			friendData, err := h.actor.getFriendDataByRoleId(roleId)
			if err != nil {
				h.Errorf("FriendApplyHandleReq got err: %s", err.Error())
				errCode = int32(cmd.ErrorCode_InternalError)
				continue
			}
			if int32(len(friendData.Friends)) >= excel.GetConfigMgr().GetCfg().FRIEND_MAX_NUM {
				errCode = int32(cmd.ErrorCode_FriendNumLimit)
				continue
			}
			baseInfo, err := h.actor.getRoleBaseDataByRoleId(roleId)
			if err != nil {
				h.Errorf("FriendApplyHandleReq got err: %s", err.Error())
				errCode = int32(cmd.ErrorCode_InternalError)
				continue
			}

			// 可以组成好友关系
			if err, _ = h.actor.Srv.CallUserActor(true, roleId, int32(cmd.Protocols_PS2S_AgreeFriendApplyReq), reqMsg, nil); err != nil {
				h.Error(err)
				errCode = int32(cmd.ErrorCode_InternalError)
				continue
			}
			// 自己的数据处理
			data.Friends[roleId] = 0
			delete(data.Examinesx, roleId)
			success = append(success, roleId)
			h.actor.comData.GetFriendData().Friends = append(h.actor.comData.GetFriendData().Friends, baseInfo.Common)
			allFailed = false // 有处理成功的玩家，不给错误提示
		}
	} else {
		// 拒绝, 对方不可见
		for _, roleId := range roleIds {
			delete(data.Examinesx, roleId)
		}
		success = roleIds
		allFailed = false // 有处理成功的玩家，不给错误提示
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	if allFailed {
		return nil, fmt.Errorf("friend apply handle failed"), errCode
	}
	// 返回消息
	return &cmd.LS2C_FriendApplyHandleRes{RoleIds: success, IsAgree: req.IsAgree, CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

func (h *FriendHandler) DelFriendReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_DelFriendReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	// 参数校验
	if h.actor.roleId == req.RoleId {
		return nil, fmt.Errorf("param error %d", req.RoleId), int32(cmd.ErrorCode_ParamError)
	}
	// 是否为好友
	data := h.actor.GetFriendData()
	if _, ok := data.Friends[req.RoleId]; !ok {
		return nil, fmt.Errorf("not friend. roleId: %d", req.RoleId), int32(cmd.ErrorCode_NotFriendShip)
	}
	err, code = h.tryHandleDelFriend(req.RoleId)
	if err != nil {
		return nil, err, int32(code)
	}
	// 返回消息
	return &cmd.LS2C_DelFriendRes{RoleId: req.RoleId}, nil, 0
}

// 删除好友逻辑
func (h *FriendHandler) tryHandleDelFriend(roleId uint64) (error, cmd.ErrorCode) {
	// 发送删除
	reqMsg := &cmd.S2S_DelFriendReq{RoleId: h.actor.roleId}
	if err, code := h.actor.Srv.CallUserActor(true, roleId, int32(cmd.Protocols_PS2S_DelFriendReq), reqMsg, nil); err != nil {
		return err, code
	}

	// 删除自己的好友数据
	data := h.actor.GetFriendData()
	delete(data.Friends, roleId)
	// 有未领取的友情点，尝试删除
	if v, ok := data.Receives[roleId]; ok && v == 0 {
		delete(data.Receives, roleId)
	}
	if err := h.SaveDB(); err != nil {
		return err, cmd.ErrorCode_SaveDBError
	}
	// 删除聊天
	err := h.actor.UserChatHandler.DeleteFriendChatMessage(h.actor.roleId, roleId)
	if err != nil {
		return err, cmd.ErrorCode_InternalError
	}
	return nil, cmd.ErrorCode_Success
}

func (h *FriendHandler) GetFriendRecommendReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_GetFriendRecommendReq
	if err = in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	data := h.actor.GetFriendData()
	// cd判定
	now := time.Now().Unix()
	if now <= data.RecommendTs {
		return nil, fmt.Errorf("refresh cd %d", data.RecommendTs), int32(cmd.ErrorCode_OperateCdError)
	}
	// 好友上限判定
	if int32(len(data.Friends)) >= excel.GetConfigMgr().GetCfg().FRIEND_MAX_NUM {
		return &cmd.LS2C_GetFriendRecommendRes{}, nil, 0
	}

	// 拉取推荐数据
	infos := h.getRecommendList()

	// cd处理
	if data.RecommendTs == 0 {
		data.RecommendTs = 1 // 首次打开界面自动刷新一次
	} else {
		data.RecommendTs = now + 30
	}
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 返回消息
	return &cmd.LS2C_GetFriendRecommendRes{RoleList: infos, RecommendTs: data.RecommendTs}, nil, 0
}

// 尝试拉取当前在线的，足够了直接返回，否则尝试拉取今日在线的
func (h *FriendHandler) getRecommendList() []*cmd.PCommonRoleBaseInfo {
	userData := h.actor.GetUserData()
	hitSize := int(excel.GetConfigMgr().GetCfg().FRIEND_RECOMMENDATION_NUM)
	infos := make([]*cmd.PCommonRoleBaseInfo, 0)

	minLv := uint32(1)
	if userData.Common.RoleLevel > 5 {
		minLv = userData.Common.RoleLevel - 5
	}

	// 范围条件
	rangeMap := map[string]service.RangeItem{
		"common.offline_time": {
			Min: float64(-1),
			Max: float64(0),
		},
		"common.role_level": {
			Min: float64(minLv),
			Max: float64(userData.Common.RoleLevel + 5),
		},
	}
	// 排除条件
	ids := make([]string, 0)
	ids = append(ids, strconv.Itoa(int(h.actor.roleId))) // 排除自己
	friendData := h.actor.GetFriendData()
	for id := range friendData.Blacks {
		ids = append(ids, strconv.Itoa(int(id))) // 排除黑名单
	}
	for id := range friendData.Friends {
		ids = append(ids, strconv.Itoa(int(id))) // 排除好友
	}
	filterMap := map[string][]string{
		"common.role_id": ids,
	}

	err, hitData := h.actor.Srv.ESMultiSearch(common.ES_ROLE_DETAIL_KEY, nil, rangeMap, filterMap, hitSize, true)
	if err != nil {
		h.Errorf("es查询出错了: %v", err)
		return infos
	}
	for _, hit := range hitData.Hits {
		temp := &cmd.PServerRoleDetailInfo{}
		if err = json.Unmarshal(hit.Source_, temp); err != nil {
			continue
		}
		infos = append(infos, temp.Common)
	}

	// 尝试拉取今日在线
	if len(infos) < hitSize {
		now := time.Now().Unix()
		rangeMap["common.offline_time"] = service.RangeItem{
			Min: float64(now - 24*60*60),
			Max: float64(now),
		}
		err, hitData = h.actor.Srv.ESMultiSearch(common.ES_ROLE_DETAIL_KEY, nil, rangeMap, filterMap, hitSize, true)
		if err != nil {
			h.Errorf("es查询出错了: %v", err)
			return infos
		}
		for _, hit := range hitData.Hits {
			temp := &cmd.PServerRoleDetailInfo{}
			if err = json.Unmarshal(hit.Source_, temp); err != nil {
				continue
			}
			infos = append(infos, temp.Common)
		}
	}
	return infos
}

func (h *FriendHandler) BlackOperateReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_BlackOperateReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	// 非自己
	if req.RoleId == h.actor.roleId {
		return nil, fmt.Errorf("role id is illegal %d", req.RoleId), int32(cmd.ErrorCode_InvalidParam)
	}
	data := h.actor.GetFriendData()
	var (
		info = &cmd.PServerRoleBaseInfo{}
	)

	// 校验
	if req.Operate == 1 {
		// 加入黑名单
		info, err = h.actor.getRoleBaseDataByRoleId(req.RoleId)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_NotFoundPlayer)
		}
		// 上限
		if int32(len(data.Blacks)) >= excel.GetConfigMgr().GetCfg().FRIEND_BLACKLIST_MAX_NUM {
			return nil, fmt.Errorf("black is limit"), int32(cmd.ErrorCode_BlackNumLimit)
		}
		// 好友？删除好友
		if _, ok := data.Friends[req.RoleId]; ok {
			if err, code = h.tryHandleDelFriend(req.RoleId); err != nil {
				return nil, err, int32(code)
			}
		}
		// 加入
		data.Blacks[req.RoleId] = 0
		// 尝试清理申请列表
		delete(data.Examinesx, req.RoleId)
	} else if req.Operate == 2 {
		// 移除黑名单
		delete(data.Blacks, req.RoleId)
	} else {
		return nil, fmt.Errorf("operate is illegal %d", req.Operate), int32(cmd.ErrorCode_ParamError)
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	return &cmd.LS2C_BlackOperateRes{Info: info.Common, Operate: req.Operate, RoleId: req.RoleId}, nil, 0
}

func (h *FriendHandler) GetFriendCampReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_GetFriendCampReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	data := h.actor.GetFriendData()
	// 非好友
	if _, ok := data.Friends[req.RoleId]; !ok {
		return &cmd.LS2C_GetFriendCampRes{RoleId: req.RoleId, ErrCode: 1}, nil, 0
	}
	uaid, err := h.actor.Srv.GetUAIDByRoleId(req.RoleId)
	if err != nil {
		return nil, fmt.Errorf("role not found %d", req.RoleId), int32(cmd.ErrorCode_NotFoundPlayer)
	}
	camp := h.actor.CampHandler.getFriendCampInfo(uaid)
	if camp == nil {
		return nil, nil, int32(cmd.ErrorCode_FriendCampNotUnlock)
	}
	return &cmd.LS2C_GetFriendCampRes{Camp: camp}, nil, 0
}

func (h *FriendHandler) OperateFriendPointReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_Friend)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_OperateFriendPointReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	dels := make([]uint64, 0)
	recvList := make([]uint64, 0)
	sendList := make([]uint64, 0)
	dropChange := &cmd.DropChange{}
	data := h.actor.GetFriendData()
	// 先判断roleIds,大于0为单个操作,否则为一键操作，先尝试领取，再尝试赠送
	if req.RoleIds > 0 {
		// 单个操作
		dels, recvList, sendList, dropChange, err, code = h.handleSinglePoint(data, req.Operate, req.RoleIds)
	} else {
		// 一键操作
		dels, recvList, sendList, dropChange, err, code = h.handleMultiPoint(data)
	}
	if err != nil {
		return nil, err, int32(code)
	}

	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}
	rsp := &cmd.LS2C_OperateFriendPointRes{
		DelIds:     dels,
		SendIds:    sendList,
		ReceiveIds: recvList,
		DropChange: dropChange,
		CommonData: h.actor.comData.FixDownComData(),
	}
	return rsp, nil, 0
}

// 单个友情点操作
func (h *FriendHandler) handleSinglePoint(data *cmd.PFriendData, operate int32, targetId uint64) (dels, recvList, sendList []uint64, change *cmd.DropChange, err error, code cmd.ErrorCode) {
	if operate == 1 {
		// 赠送
		dels, sendList, err, code = h.handleSendGift(data, []uint64{targetId})
		return
	} else if operate == 2 {
		// 收取
		dels, recvList, change, err, code = h.handleReceiveGift(data, []uint64{targetId})
		return
	} else {
		err = fmt.Errorf("illegal operate %d", operate)
		code = cmd.ErrorCode_ParamError
		return
	}
}

// 多个友情点操作
func (h *FriendHandler) handleMultiPoint(data *cmd.PFriendData) (dels, recvList, sendList []uint64, change *cmd.DropChange, err error, code cmd.ErrorCode) {
	// 尝试一键领取
	dels, recvList, change, err, code = h.handleReceiveGift(data, nil)
	if err != nil {
		return
	}
	var tempDels []uint64
	tempDels, sendList, err, code = h.handleSendGift(data, nil)
	dels = append(dels, tempDels...)
	return
}

// 赠送友情点逻辑
func (h *FriendHandler) handleSendGift(data *cmd.PFriendData, targetIds []uint64) (dels, sendList []uint64, err error, code cmd.ErrorCode) {
	// 一键操作判定
	onlines := make([]uint64, 0) // 在线的时间戳为0，特殊处理
	if len(targetIds) == 0 {
		// 筛选赠送目标
		temp := make(map[int64]uint64)
		for id := range data.Friends {
			if _, ok := data.Sends[id]; ok {
				continue
			}
			// 查询离线时间
			baseInfo, err := h.actor.getRoleBaseDataByRoleId(id)
			if err != nil {
				continue
			}
			if baseInfo.Common.OfflineTime == -1 {
				onlines = append(onlines, id)
			} else {
				temp[baseInfo.Common.OfflineTime] = id
			}
		}
		// 排序
		targetIds = myUtils.SortMapValByKeys(temp, myUtils.SORT_ORDER_DESC)
		targetIds = append(onlines, targetIds...)
	}

	reqMsg := &cmd.S2S_SendFriendPointReq{RoleId: h.actor.roleId}
	for _, roleId := range targetIds {
		// 是否好友
		if _, ok := data.Friends[roleId]; !ok {
			dels = append(dels, roleId)
			continue
		}
		// 已经赠送过了
		if _, ok := data.Sends[roleId]; ok {
			continue
		}
		// 次数上限
		if int32(len(data.Sends)) >= excel.GetConfigMgr().GetCfg().FRIEND_GIVE_RECEIVE_MAX_NUM {
			continue
		}

		// 赠送给对方
		if err, code = h.actor.Srv.CallUserActor(true, roleId, int32(cmd.Protocols_PS2S_SendFriendPointReq), reqMsg, nil); err != nil {
			h.Error(err)
			continue
		}

		// 处理自己的数据
		data.Sends[roleId] = 0
		sendList = append(sendList, roleId)
	}
	h.Infof("赠送友情点 targetIds: %v, send: %v", targetIds, sendList)
	return
}

// 领取友情点逻辑
func (h *FriendHandler) handleReceiveGift(data *cmd.PFriendData, targetIds []uint64) (dels, recvList []uint64, change *cmd.DropChange, err error, code cmd.ErrorCode) {
	// 一键操作判定
	if len(targetIds) == 0 {
		for id, status := range data.Receives {
			if status == 0 {
				targetIds = append(targetIds, id)
			}
		}
	}

	var sum int32
	for _, roleId := range targetIds {
		// 是否好友
		if _, ok := data.Friends[roleId]; !ok {
			dels = append(dels, roleId)
			continue
		}
		// 没有可领取，或者领取过了
		if v, ok := data.Receives[roleId]; !ok || v == 1 {
			continue
		}
		// 次数上限
		if calReceivesNum(data.Receives) >= excel.GetConfigMgr().GetCfg().FRIEND_GIVE_RECEIVE_MAX_NUM {
			continue
		}

		// 处理逻辑
		data.Receives[roleId] = 1
		sum += excel.GetConfigMgr().GetCfg().FRIEND_GIFT_NUM
		recvList = append(recvList, roleId)
	}
	if sum > 0 {
		change, err = GetDropMgr(h.actor).DropList2(map[int32]int32{common.CURRENCY_ITEM_ID_2013: sum}, true, nil, h.actor.comData, common.CR_ADD_FRIEND_POINT)
		if err != nil {
			code = cmd.ErrorCode_InternalError
		}
	}
	h.Infof("收取友情点 targetIds: %v, receive: %v", targetIds, recvList)
	return
}

// 计算领取过友情点的次数
func calReceivesNum(data map[uint64]int32) int32 {
	var ret int32
	for _, v := range data {
		if v == 1 {
			ret++
		}
	}
	return ret
}

// 离线事件处理：删除好友处理
func (h *FriendHandler) HandleDelFriend(delRoleIds map[int32]int32) error {
	data := h.actor.GetFriendData()
	for k := range delRoleIds {
		delete(data.Friends, uint64(k))
		// 有未领取的友情点，尝试删除
		if v, ok := data.Receives[uint64(k)]; ok && v == 0 {
			delete(data.Receives, uint64(k))
		}
	}
	h.Infof("HandleDelFriend roleIds: %v", delRoleIds)
	return h.SaveDB()
}

// 离线事件处理：添加申请
func (h *FriendHandler) HandleAddFriendApply(roleIds map[int32]int32) error {
	var refresh bool
	data := h.actor.GetFriendData()
	now := time.Now().Unix()
	for k := range roleIds {
		// 判定是否在黑名单中
		if _, ok := data.Blacks[uint64(k)]; ok {
			continue
		}
		data.Examinesx[uint64(k)] = now
		refresh = true

		//// 通知
		//baseInfo, err := h.actor.getRoleBaseDataByRoleId(uint64(k))
		//if err != nil {
		//	h.Errorf("HandleAddFriendApply got err: %v", err)
		//	continue
		//}
		//h.actor.comData.GetFriendData().Examines = append(h.actor.comData.GetFriendData().Examines, baseInfo.Common)
	}
	if refresh {
		h.tryClearExaminesList(data.Examinesx)
	}

	h.Infof("HandleAddFriendApply roleIds: %v", roleIds)
	return h.SaveDB()
}

// 离线事件处理：同意申请
func (h *FriendHandler) HandleAgreeFriendApply(roleIds map[int32]int32) error {
	data := h.actor.GetFriendData()
	for k := range roleIds {
		// 尝试清除黑名单
		delete(data.Blacks, uint64(k))
		data.Friends[uint64(k)] = 0

		//// 通知
		//baseInfo, err := h.actor.getRoleBaseDataByRoleId(uint64(k))
		//if err != nil {
		//	h.Errorf("HandleAgreeFriendApply got err: %v", err)
		//	continue
		//}
		//h.actor.comData.GetFriendData().Friends = append(h.actor.comData.GetFriendData().Friends, baseInfo.Common)
	}

	h.Infof("HandleAgreeFriendApply roleIds: %v", roleIds)
	return h.SaveDB()
}

// 离线事件处理：友情点赠送
func (h *FriendHandler) HandleSendFriendPoint(roleIds map[int32]int32) error {
	data := h.actor.GetFriendData()
	for k := range roleIds {
		data.Receives[uint64(k)] = 0
		//h.actor.comData.GetFriendData().Receives = append(h.actor.comData.GetFriendData().Receives, uint64(k))
	}

	h.Infof("HandleSendFriendPoint roleIds: %v", roleIds)
	return h.SaveDB()
}

// 判断指定角色id的玩家是否为好友
func (h *FriendHandler) IsFriend(roleId uint64) bool {
	data := h.actor.GetFriendData()
	_, ok := data.Friends[roleId]
	return ok
}

// 判断指定角色id的玩家是否在黑名单
func (h *FriendHandler) InBlackList(roleId uint64) bool {
	data := h.actor.GetFriendData()
	_, ok := data.Blacks[roleId]
	return ok
}

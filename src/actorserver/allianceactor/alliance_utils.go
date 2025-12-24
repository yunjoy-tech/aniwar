package allianceactor

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	randutil "gitee.com/aniwar2/musae/utils/rand"
	timeutil "gitee.com/aniwar2/musae/utils/time"
	"time"
)

func (h *AllianceHandler) GetAllianceData() *pb.PServerAllianceInfo {
	data := h.actor.Data
	if data.Examines == nil {
		data.Examines = make(map[uint64]int64)
	}
	if data.Base.SignLog == nil {
		data.Base.SignLog = make(map[uint64]int32)
	}

	// 尝试清空每周活跃度
	refreshTime := time.Unix(data.Base.WeekTs, 0)
	now := time.Now()
	if !timeutil.IsSameDay(refreshTime, now) {
		data.Base.WeekContribute = 0
		data.Base.WeekTs = now.Unix()
		// 清空每个成员的记录
		for _, member := range data.Member {
			member.Contribute = 0
		}
		h.Infof("联盟本周贡献值清理了...")
	}

	// 尝试清空每日签到奖励记录
	refreshTime = time.Unix(data.Base.SignTs, 0)
	if !timeutil.IsSameDay(refreshTime, now) {
		data.Base.SignLog = make(map[uint64]int32)
		data.Base.SignTs = now.Unix()
		h.Infof("联盟每日签到奖励记录清理了...")
	}
	return data
}

// 加入联盟处理
func (h *AllianceHandler) joinAlliance(targetId uint64, positionId int32, in *base.ProtoMsg) *pb.PAllianceMember {
	data := h.GetAllianceData()
	member := &pb.PAllianceMember{
		RoleId:     targetId,
		Position:   positionId,
		Contribute: 0,
	}
	data.Member[targetId] = member
	data.Base.MemberNum = int32(len(data.Member))
	h.addAllianceLog(targetId, pb.AllianceLogType_LogType_Join)
	// 在线自动签到
	base, err := h.actor.getRoleBaseDataByRoleId(targetId)
	if err == nil && base.Common.OfflineTime == -1 {
		var add int32
		// excel.GetAllianceExpMgr().Foreach(func(cfg *excel.AllianceExpCfg) bool {
		// 	if cfg.Type != 1 {
		// 		return true
		// 	}
		// 	if cfg.TypeParm == 1 {
		// 		add += cfg.Contribution
		// 	}
		// 	return true
		// }, true)
		h.handleAddContribute(1, add, targetId)
	}
	// 绑定Topic
	h.actor.AddGateTopic(in.GetTopic(), in.GetUserId())
	h.Debugf("alliance addGateTopic[%s],user[%s]", in.GetTopic(), in.GetUserId())
	h.Infof("新成员加入联盟了 id: %v, position: %v", targetId, positionId)
	return member
}

// 退出联盟处理
func (h *AllianceHandler) exitAlliance(targetId uint64, typ int32, operator uint64, uid string) {
	data := h.GetAllianceData()
	_, ok := data.Member[targetId]
	if !ok {
		// 不存在，就不走后续逻辑了
		return
	}

	// 处理联盟数据
	delete(data.Member, targetId)
	data.Base.MemberNum = int32(len(data.Member))

	// 通知被踢人
	if typ == 2 {
		reqMsg := &pb.S2S_ExitAllianceNtf{}
		if err, code := h.actor.Srv.CallUserActor(true, targetId, int32(pb.Protocols_PS2S_ExitAllianceNtf), reqMsg, nil); err != nil {
			h.Errorf("exit alliance got error: %s, code: %v", err, code)
			return
		}
	}

	// 日志记录
	if typ == 1 { // 退出
		h.addAllianceLog(targetId, pb.AllianceLogType_LogType_Exit)
	} else { // 踢出
		info, _ := h.actor.getRoleBaseDataByRoleId(operator)
		if info != nil {
			h.addAllianceLog(targetId, pb.AllianceLogType_LogType_Kickout, info.Common.RoleName)
		}
	}
	// 盟主特殊处理
	f := h.handleLeaderExit(targetId)
	if !f {
		h.tryUploadToES()
	}
	// 解绑topic
	h.actor.DelGateTopic(uid)
	h.Debugf("alliance delGateTopic[%s]", uid)
	h.Infof("成员退出联盟了 id: %v, type: %v", targetId, typ)
}

// 盟主退出处理, 返回联盟是否解散的标记, true为已解散，false为未解散
func (h *AllianceHandler) handleLeaderExit(targetId uint64) bool {
	data := h.GetAllianceData()
	// 不是盟主
	if data.Base.LeaderId != targetId {
		return false
	}
	h.Debugf("盟主退出联盟了...")
	// 还有成员
	if len(data.Member) > 0 {
		// 职位最高并且贡献度最高的成员，如果都相同则随机
		var max int32
		var newLeaders = make([]*pb.PAllianceMember, 0)
		for _, member := range data.Member {
			// 初始化
			if len(newLeaders) == 0 {
				max = member.Position*100000 + member.Contribute
				newLeaders = append(newLeaders, member)
				continue
			}
			temp := member.Position*100000 + member.Contribute
			if temp < max {
				continue
			} else if temp == max {
				newLeaders = append(newLeaders, member)
			} else {
				newLeaders = []*pb.PAllianceMember{member}
				max = temp
			}
		}
		// 随机一个
		r := randutil.RangeInt(0, len(newLeaders))
		newLeader := newLeaders[r]
		h.Infof("盟主自动选举结果 new: %v, old: %v", newLeader.RoleId, targetId)

		// 修正新数据
		changeLeader(data, newLeader.RoleId, targetId)
		return false
	} else {
		// 没有成员了,解散联盟,暂不删除db数据
		// // 删除ES数据
		// err := h.actor.Srv.ESDel(common.ES_ALLIANCE_BASE_KEY, strconv.Itoa(int(data.Base.Id)))
		// if err != nil {
		// 	h.Error(err)
		// }
		h.Infof("联盟解散了 id: %v", data.Base.Id)
		return true
	}
}

// 更换盟主
func changeLeader(data *pb.PServerAllianceInfo, newLeader, oldLeader uint64) {
	data.Base.LeaderId = newLeader
	leader := data.Member[newLeader]
	leader.Position = int32(pb.MemberPositionType_Leader)
	// 老盟主是否还在
	old := data.Member[oldLeader]
	if old != nil {
		old.Position = int32(pb.MemberPositionType_Member)
	}
	logger.Infof("更换盟主成功...")
}

// 增加联盟日志
func (h *AllianceHandler) addAllianceLog(targetId uint64, typ pb.AllianceLogType, params ...string) {
	data := h.GetAllianceData()
	info, _ := h.actor.getRoleBaseDataByRoleId(targetId)
	if info == nil {
		return
	}

	log := &pb.PCommonAllianceLog{
		Name:   info.Common.RoleName,
		Ts:     time.Now().Unix(),
		Typ:    typ,
		Params: params,
	}
	data.Log = append(data.Log, log)
	// 尝试清理日志
	needDel := len(data.Log) /*- int(excel.GetAllianceParmMgr().GetById(5).AllianceParm)*/
	if needDel > 0 {
		data.Log = data.Log[needDel:]
	}
	h.Debugf("增加联盟日志 id: %v, type: %v, params: %v", targetId, typ, params)
}

// 检查权限是否合法
func (h *AllianceHandler) checkPermission(roleId uint64, typ PermissionType) bool {
	// data := h.GetAllianceData()
	// 获取成员的职位
	// member, ok := data.Member[roleId]
	// if !ok {
	// 	return false
	// }

	// 查找权限配置
	// cfg := excel.GetAlliancePostMgr().GetById(member.Position)
	// if cfg == nil {
	// 	return false
	// }
	// for _, id := range cfg.Permission {
	// 	if id == int32(typ) {
	// 		return true
	// 	}
	// }
	return false
}

// 同步联盟基础信息到ES
func (h *AllianceHandler) tryUploadToES() {
	data := h.GetAllianceData()
	// 同步
	// err := h.actor.Srv.ESPut(common.ES_ALLIANCE_BASE_KEY, strconv.Itoa(int(data.Base.Id)), data.Base)
	// if err != nil {
	// 	h.Error(err)
	// 	return
	// }
	h.Infof("同步联盟到es中... allianceId: %d", data.Base.Id)
}

// 获取指定玩家的base数据块
func (a *AllianceActor) getRoleBaseDataByRoleId(roleId uint64) (*pb.PServerRoleBaseInfo, error) {
	uaid, err := a.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return nil, fmt.Errorf("roleId not found %v", roleId)
	}

	data := &pb.PServerRoleBaseInfo{}
	_, err = a.GetCache(service.MongoDbType_MongoGame, db.KeyUserBaseInfo(uaid), data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (h *AllianceHandler) toAllianceBaseInfo(base *pb.PServerAllianceBaseInfo) *pb.PCommonAllianceBaseInfo {
	info, _ := h.actor.getRoleBaseDataByRoleId(base.LeaderId)
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
	}
}

func (h *AllianceHandler) toCommonAllianceMembers(members map[uint64]*pb.PAllianceMember) []*pb.PCommonAllianceMember {
	ret := make([]*pb.PCommonAllianceMember, 0)
	for _, member := range members {
		ret = append(ret, h.toCommonAllianceMember(member))
	}
	return ret
}

// 成员数据填充
func (h *AllianceHandler) toCommonAllianceMember(member *pb.PAllianceMember) *pb.PCommonAllianceMember {
	info, _ := h.actor.getRoleBaseDataByRoleId(member.RoleId)
	if info == nil {
		return nil
	}
	return &pb.PCommonAllianceMember{
		Role:       info.Common,
		Position:   member.Position,
		Contribute: member.Contribute,
	}
}

// 联盟信息填充
func (h *AllianceHandler) toCommonAllianceInfo(data *pb.PServerAllianceInfo) *pb.PCommonAllianceInfo {
	// 成员数据
	members := make([]*pb.PCommonAllianceMember, 0)
	for _, member := range data.Member {
		members = append(members, h.toCommonAllianceMember(member))
	}

	// 审核数据
	examines := make([]*pb.PCommonRoleBaseInfo, 0)
	for id := range data.Examines {
		info, _ := h.actor.getRoleBaseDataByRoleId(id)
		if info != nil {
			examines = append(examines, info.Common)
		}
	}
	return &pb.PCommonAllianceInfo{
		Base:     h.toAllianceBaseInfo(data.Base),
		Members:  members,
		Examines: examines,
	}
}

// 检查职位剩余数量是否足够, 足够返回true, 否则返回false
func (h *AllianceHandler) checkPositionNum(positionId int32) bool {
	var max int32
	var cur int32
	// 获取最大数量
	cfg := h.getAllianceLevelCfg()
	if cfg == nil {
		return false
	}
	switch positionId {
	case int32(pb.MemberPositionType_Member):
		max = cfg.MemberNum
	case int32(pb.MemberPositionType_Admin):
		max = cfg.AdminNum
	case int32(pb.MemberPositionType_Leader):
		max = 1
	}

	// 查询当前任职数量
	data := h.GetAllianceData()
	for _, member := range data.Member {
		if member.Position == positionId {
			cur++
		}
	}
	return max > cur
}

// IsOnline 联盟成员是否在线
func (h *AllianceHandler) IsOnline(roleId uint64) bool {
	info, _ := h.actor.getRoleBaseDataByRoleId(roleId)
	if info == nil {
		return false
	}
	return info.Common.OfflineTime == -1
}

// func (h *AllianceHandler) GetAllianceMessageKey(allianceId int64) string {
// 	return fmt.Sprintf("alliance_chat_%d", allianceId)
// }

// SaveAllianceChatMessage 存储联盟聊天消息到es
func (h *AllianceHandler) SaveAllianceChatMessage(allianceId int64, message *pb.BroadMessage) error {
	// esIndex := h.GetAllianceMessageKey(allianceId)
	// if esIndex == "" {
	// 	return errors.New("获取索引失败")
	// }
	// if err := h.actor.Srv.ESPutNoId(esIndex, message); err != nil {
	// 	h.Error(err)
	// 	return err
	// }
	return nil
}

// GetAllianceChatMessage 获取联盟消息
func (h *AllianceHandler) GetAllianceChatMessage(allianceId int64, endTime int64, form, size int32) []*pb.BroadMessage {
	// esIndex := h.GetAllianceMessageKey(allianceId)
	// infos := make([]*pb.BroadMessage, 0)
	// rangeMap := map[string]service.RangeItem{
	// 	"timeStamp": {
	// 		Min: float64(0),
	// 		Max: float64(endTime),
	// 	},
	// }
	//
	// err, hitData := h.actor.Srv.ESMultiSearchPage(esIndex, rangeMap, int(size), &sortorder.Desc, int(form))
	// if err != nil {
	// 	h.Warnf("es查询出错了: %v", err.Error())
	// 	return infos
	// }
	// for _, hit := range hitData.Hits {
	// 	temp := &pb.BroadMessage{}
	// 	if err = json.Unmarshal(hit.Source_, temp); err != nil {
	// 		continue
	// 	}
	// 	infos = append(infos, temp)
	// }
	// return infos
	return nil
}

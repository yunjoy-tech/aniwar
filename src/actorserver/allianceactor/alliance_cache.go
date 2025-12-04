package allianceactor

//type CacheMgr struct {
//	Actor *AllianceActor
//
//	Members  sync.Map // 成员角色基础信息
//	ExpireTS int64    // 失效时间戳
//}
//
//func NewCacheMgr(actor *AllianceActor) *CacheMgr {
//	return &CacheMgr{
//		Actor:    actor,
//		Members:  sync.Map{},
//		ExpireTS: time.Now().Unix() + 60,
//	}
//}
//
//func (m *CacheMgr) GetMemberDataById(roleId uint64) *cmd.PServerRoleBaseInfo {
//	m.tryRefreshData()
//	var baseInfo *cmd.PServerRoleBaseInfo
//	// 获取缓存数据
//	if value, ok := m.Members.Load(roleId); ok {
//		baseInfo = value.(*cmd.PServerRoleBaseInfo)
//	} else {
//		// 新数据加载
//		info, err := m.Actor.getRoleBaseDataByRoleId(roleId)
//		if err != nil {
//			m.Actor.Errorf("GetMemberDataByMap got err: %s", err.Error())
//		} else {
//			m.Members.Store(roleId, info)
//		}
//		baseInfo = info
//	}
//	return baseInfo
//}
//
//// 获取一批成员缓存数据
//func (m *CacheMgr) GetMemberDataByIds(roleIds []uint64) []*cmd.PServerRoleBaseInfo {
//	m.tryRefreshData()
//	ret := make([]*cmd.PServerRoleBaseInfo, 0)
//	for _, roleId := range roleIds {
//		// 获取缓存数据
//		if value, ok := m.Members.Load(roleId); ok {
//			baseInfo := value.(*cmd.PServerRoleBaseInfo)
//			ret = append(ret, baseInfo)
//		} else {
//			// 新数据加载
//			baseInfo, err := m.Actor.getRoleBaseDataByRoleId(roleId)
//			if err != nil {
//				m.Actor.Errorf("GetMemberDataByMap got err: %s", err.Error())
//				continue
//			} else {
//				m.Members.Store(roleId, baseInfo)
//			}
//			ret = append(ret, baseInfo)
//		}
//	}
//	return ret
//}
//
//func (m *CacheMgr) GetMemberData(member *cmd.PAllianceMember) *cmd.PCommonAllianceMember {
//	m.tryRefreshData()
//	var ret *cmd.PCommonAllianceMember
//	// 获取缓存数据
//	if value, ok := m.Members.Load(member.RoleId); ok {
//		baseInfo := value.(*cmd.PServerRoleBaseInfo)
//		ret = toCommonMember(member, baseInfo)
//	} else {
//		// 新数据加载
//		baseInfo, err := m.Actor.getRoleBaseDataByRoleId(member.RoleId)
//		if err != nil {
//			m.Actor.Errorf("GetMemberDataByMap got err: %s", err.Error())
//		} else {
//			m.Members.Store(member.RoleId, baseInfo)
//			ret = toCommonMember(member, baseInfo)
//		}
//	}
//
//	return ret
//}
//
////获取一批成员缓存数据
//func (m *CacheMgr) GetMemberDataByMap(members map[uint64]*cmd.PAllianceMember) []*cmd.PCommonAllianceMember {
//	m.tryRefreshData()
//	ret := make([]*cmd.PCommonAllianceMember, 0)
//	for roleId, member := range members {
//		// 获取缓存数据
//		if value, ok := m.Members.Load(roleId); ok {
//			baseInfo := value.(*cmd.PServerRoleBaseInfo)
//			ret = append(ret, toCommonMember(member, baseInfo))
//		} else {
//			// 新数据加载
//			baseInfo, err := m.Actor.getRoleBaseDataByRoleId(roleId)
//			if err != nil {
//				m.Actor.Errorf("GetMemberDataByMap got err: %s", err.Error())
//				continue
//			} else {
//				m.Members.Store(roleId, baseInfo)
//			}
//			ret = append(ret, toCommonMember(member, baseInfo))
//		}
//	}
//	return ret
//}
//
//// 删除一批成员缓存数据
//func (m *CacheMgr) DelMemberData(members []uint64) {
//	for _, id := range members {
//		m.Members.Delete(id)
//	}
//}
//
//// 尝试刷新过期数据
//func (m *CacheMgr) tryRefreshData() {
//	now := time.Now().Unix()
//	if now < m.ExpireTS {
//		return
//	}
//
//	// 过期了
//	m.ExpireTS = now + 40 // 两次心跳的时间
//	m.Members.Range(func(key, value any) bool {
//		roleId := key.(uint64)
//		baseInfo, err := m.Actor.getRoleBaseDataByRoleId(roleId)
//		if err != nil {
//			m.Actor.Errorf("tryRefreshData got err: %s", err.Error())
//		} else {
//			m.Members.Store(roleId, baseInfo)
//		}
//		return true
//	})
//	logger.Debugf("刷新联盟缓存的成员数据了...")
//}

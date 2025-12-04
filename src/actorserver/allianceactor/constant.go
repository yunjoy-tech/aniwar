package allianceactor

type PermissionType int32

const (
	MEMBER_PERMISSION_EXAMINE      PermissionType = 1 // 审批成员
	MEMBER_PERMISSION_KICKOUT      PermissionType = 2 // 踢出成员
	MEMBER_PERMISSION_APPOINT      PermissionType = 3 // 任命/卸任元老
	MEMBER_PERMISSION_ABDICATE     PermissionType = 4 // 委任头目
	MEMBER_PERMISSION_EDIT_LOGO    PermissionType = 5 // 编辑头像
	MEMBER_PERMISSION_EDIT_NOTICE  PermissionType = 6 // 编辑公告
	MEMBER_PERMISSION_EDIT_PROFILE PermissionType = 7 // 编辑简介
)

package logic

type VerifyReq struct {
	ReqType int32 `json:"type"`   // 固定值 “send_mail”
	Number  int32 `json:"number"` // 审批编号
}

type GetOrderReq struct {
	Uid    string `json:"uid"`
	RoleId uint64 `json:"roleId"` // roleId
}

type GetDropOrderReq struct {
	OrderId string `json:"orderId"`
	Url     string `json:"url"` // 所属集群
	Uid     string `json:"uid"`
	RoleId  uint64 `json:"roleId"` // roleId
}

type ReduceItem struct {
	Uid    string `json:"uid"`
	RoleId string `json:"roleId"` // roleId
	ItemId int32  `json:"itemId"`
	Num    int32  `json:"num"`
}

type GetVersion struct {
	Ops string `json:"ops"` // 1 服务端版本 2 客户端版本
}

type ServerVersion struct {
	VersionRecord            []*VersionRecord `json:"version_record"`
	CurVersion               string           `json:"cur_version"`                 // 服务端版本
	CurVersionAndroid        string           `json:"cur_version_android"`         //当前版本android
	CurVersionIos            string           `json:"cur_version_ios"`             //当前版本ios
	MinVersionAndroid        string           `json:"min_version_android"`         //最低版本android
	MinVersionIos            string           `json:"min_version_ios"`             //最低版本ios
	CurJenkinsVersionAndroid string           `json:"cur_jenkins_version_android"` // 客户端线上版本=> jenkins 流水号映射
	CurJenkinsVersionIos     string           `json:"cur_jenkins_version_ios"`     // 客户端线上版本=> jenkins 流水号映射
	MinJenkinsVersionAndroid string           `json:"min_jenkins_version_android"` // 客户端线上版本=> jenkins 流水号映射
	MinJenkinsVersionIos     string           `json:"min_jenkins_version_ios"`     // 客户端线上版本=> jenkins 流水号映射
}

type GetClientMaxVersionReq struct {
	Platform string `json:"platform"` //安卓还是IOS
}
type GetClientMaxVersion struct {
	Version string `json:"version"`
}

type VersionRecord struct {
	Version      string `json:"version"`
	VersionType  int32  `json:"version_type"`
	VersionNotes string `json:"version_notes"`
	UploadTime   string `json:"upload_time"`
	PkgName      string `json:"pkg_name"`
	State        int32  `json:"state"`
}

type ChangeVersionState struct {
	ServiceType string `json:"service_type"` // 1 服务端 2 客户端
	Version     string `json:"version"`
	State       int32  `json:"state"`
	Channel     string `json:"channel"`     //渠道
	NewVersion  string `json:"new_version"` // 这个字段传给IDIP用,服务端维护的自增id
}

type SetMinVersion struct {
	Platform string `json:"platform"` // 操作系统ios ,android
	Version  string `json:"version"`
	Branch   string `json:"branch"`
}

type GoLive struct {
	OptType int32  `json:"typ"`
	PkgName string `json:"pkg_name"`
	Version string `json:"version"`
}

type ExcelExpired struct {
	Day     int32  `json:"day"`     // 支持天
	Version string `json:"version"` // 版本号
}

type ExcelExpiredRes struct {
	ExpiredTime int32  `json:"expired_time"`
	Version     string `json:"version"` // 版本号
}

type GetExcelListRes struct {
	FileNames []string `json:"file_names"`
}

type ClientVersionPublishReq struct {
	Version    string `json:"version"`
	State      int32  `json:"state"`
	Branch     string `json:"branch"`
	Platform   string `json:"platform"`    //渠道
	NewVersion string `json:"new_version"` // 这个字段传给IDIP用,服务端维护的自增id
}
type ClientVersionPublishRes struct {
	CurVersion string `json:"cur_version"`
}

type GetExcelList struct {
	OptType   int32  `json:"typ"`
	PkgName   string `json:"pkg_name"`
	NameSpace string `json:"name_space"`
	Group     string `json:"group"`
	Version   string `json:"version"`
}

type GetRollingVersionRes struct {
	Version string `json:"version"`
}

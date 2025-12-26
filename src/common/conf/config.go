package conf

import (
	"gitee.com/aniwar2/musae/baseconf"
	"gopkg.in/yaml.v3"
	"os"
)

var gConf = &ServerConf{}

// BaseConf 实现了musae的IConf接口，用于将gConf反向注册到框架的Iconf全局变量
// 为了保持调用简洁，业务层一般不调用这个方法
func (s *ServerConf) BaseConf() *baseconf.BaseConf {
	return s.Base
}

// 整个server的配置项，包含了musae的baseconf和业务层自定义的conf
type ServerConf struct {
	Base *baseconf.BaseConf `yaml:"BaseConf"` // 基础配置
	// 服务相关配置
	Guide *GuideConf `yaml:"GuideConf"` // GuideServer 配置
	Login *LoginConf `yaml:"LoginConf"` // LoginServer 配置
	Gate  *GateConf  `yaml:"GateConf"`  // GateServer 配置
	Actor *ActorConf `yaml:"ActorConf"` // ActorServer 配置
	Idip  *IdipConf  `yaml:"IdipConf"`  // IdipServer 配置
	Bill  *BillConf  `yaml:"BillConf"`  // BillServer 配置
	// 业务相关配置
	GMT     *GMTConf     `yaml:"GMTConf"`     // gmt配置
	DDos    *DDosConf    `yaml:"DDosConf"`    // 负载配置
	SrvAddr *SrvAddrConf `yaml:"SrvAddrConf"` // 服务地址
	OSS     *OSSConf     `yaml:"OSSConf"`     // OSS配置
}

// 加载配置文件
func LoadConf(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &gConf)
	if err != nil {
		return err
	}
	baseconf.Iconf = gConf
	return nil
}

func Base() *baseconf.BaseConf {
	return gConf.Base
}

// ==========业务层自定义的配置==========

// DDosConf DDosConf配置
type DDosConf struct {
	TimeInterval  uint32 `yaml:"timeInterval"`  // 检测时间间隔，单位秒  0表示不检测
	LoginLimitNum uint32 `yaml:"loginLimitNum"` // 每秒登陆频率上限，0表示不限
	TotalUserNum  uint32 `yaml:"totalUserNum"`  // 每秒登录用户数量上限 0表示不限
	LimitPktNum   uint32 `yaml:"limitPktNum"`   // 每间隔限制数据包量, 0表示不限
	LimitByteNum  uint32 `yaml:"limitByteNum"`  // 每间隔限制数据流量, 0表示不限
}

func DDos() *DDosConf {
	return gConf.DDos
}

// GMTConf 后台管理工具相关配置
type GMTConf struct {
	ApiSecret   string   `yaml:"apiSecret"`   // 验签密钥
	IsIpWhite   bool     `yaml:"isIpWhite"`   // 是否开启IP白名单
	IpWhiteList []string `yaml:"ipWhiteList"` // ip白名单
}

func GMT() *GMTConf {
	return gConf.GMT
}

// SrvAddrConf 客户端更新地址配置
type SrvAddrConf struct {
	UpdateAddrARD []string `yaml:"updateAddrARD"`
	UpdateAddrIOS []string `yaml:"updateAddrIOS"`
	TCPAddr       []string `yaml:"tcpAddr"`
	HTTPAddr      []string `yaml:"httpAddr"`
}

func SrvAddr() *SrvAddrConf {
	return gConf.SrvAddr
}

// OSSConf oss相关配置
type OSSConf struct {
	Endpoint      string `yaml:"endpoint"`
	AccessKey     string `yaml:"accessKey"`
	AccessSecret  string `yaml:"accessSecret"`
	VersionBucket string `yaml:"versionBucket"`
	DownPath      string `yaml:"downPath"`
}

func OSS() *OSSConf {
	return gConf.OSS
}

// ==========各个服务的配置==========

// GuideConf 引导服配置项
type GuideConf struct {
}

func Guide() *GuideConf {
	return gConf.Guide
}

// LoginConf 登录服配置项
type LoginConf struct {
	LoginReqRate   int32  `yaml:"loginReqRate"`  // loginReq处理频率每秒
	LoginReqQueue  int32  `yaml:"loginReqQueue"` // login最大请求队列
	GateUrl        string `yaml:"gateUrl"`
	GateUrlDev     string `yaml:"gateUrlDev"`
	UserActorAhead bool   `yaml:"userActorAhead"`
}

func Login() *LoginConf {
	return gConf.Login
}

// GateConf 网关服配置项
type GateConf struct {
	GatePendingNumLimit int32 `yaml:"gatePendingNumLimit"` // gate排队人数限制
	GateLoginRateLimit  int32 `yaml:"gateLoginRateLimit"`  // gate登录频率限制
	GateLoginThreadNum  int32 `yaml:"gateLoginThreadNum"`  // gate登录协成数量
	GateUserNumLimit    int32 `yaml:"gateUserNumLimit"`    // 每个gate上的登录用户上限
}

func Gate() *GateConf {
	return gConf.Gate
}

// ActorConf Actor服配置项
type ActorConf struct {
	RoomTokenTTL     int    `yaml:"roomTokenTTL"`     // room的Token有效时长
	MailActorMin     int32  `yaml:"mailActorMin"`     // 邮件actor最小启用数量
	MailActorPercent int32  `yaml:"mailActorPercent"` // 邮件Actor启用数量万分比
	DirtyWords       string `yaml:"dirtyWords"`       // 屏蔽字库
}

func Actor() *ActorConf {
	return gConf.Actor
}

// IdipConf Idip服配置项
type IdipConf struct {
}

func Idip() *IdipConf {
	return gConf.Idip
}

// BillConf 充值服相关配置
type BillConf struct {
	CanPay        int32    `yaml:"canPay"`        // 是否开启充值, 1:是, 0:否
	CanVirtualPay int32    `yaml:"canVirtualPay"` // 是否支持模拟充值, 1:是, 0:否
	ApiSecret     string   `yaml:"apiSecret"`     // 验签密钥
	IsIpWhite     bool     `yaml:"isIpWhite"`     // 是否开启IP白名单
	IpWhiteList   []string `yaml:"ipWhiteList"`   // ip白名单
}

func Bill() *BillConf {
	return gConf.Bill
}

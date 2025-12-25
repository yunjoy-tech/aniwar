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
	Base    *baseconf.BaseConf `yaml:"BaseConf"`    // 基础配置
	DDos    *DDosConf          `yaml:"DDosConf"`    // 负载配置
	Login   *LoginConf         `yaml:"LoginConf"`   // 版本校验
	GMT     *GMTConf           `yaml:"GMTConf"`     // gmt配置
	Bill    *BillConf          `yaml:"BillConf"`    // bill配置
	SrvAddr *SrvAddrConf       `yaml:"SrvAddrConf"` // 服务地址
	OSS     *OSSConf           `yaml:"OSSConf"`     // OSS配置
}

// 加载配置文件
func LoadConf(path string, params ...string) error {
	var data []byte
	var err error

	if len(params) > 0 {
		data = []byte(params[0])
	} else {
		data, err = os.ReadFile(path)
		if err != nil {
			return err
		}
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

// LoginConf 登录服配置项
type LoginConf struct {
	GateUrl        string `yaml:"gateUrl"`
	GateUrlDev     string `yaml:"gateUrlDev"`
	UserActorAhead bool   `yaml:"userActorAhead"`
}

func Login() *LoginConf {
	return gConf.Login
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

// BillConf 支付相关配置
type BillConf struct {
	ApiSecret   string   `yaml:"apiSecret"`   // 验签密钥
	IsIpWhite   bool     `yaml:"isIpWhite"`   // 是否开启IP白名单
	IpWhiteList []string `yaml:"ipWhiteList"` // ip白名单
}

func Bill() *BillConf {
	return gConf.Bill
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

// TODO 具体某个server的配置，拆分出来
// type LoginServerConf struct {
// 	ServerConf
// }
//
// type GateServerConf struct {
// 	ServerConf
// }
//
// type BillServerConf struct {
// 	ServerConf
// }

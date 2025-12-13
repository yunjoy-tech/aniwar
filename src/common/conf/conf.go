package conf

import (
	"encoding/json"
	"io/ioutil"

	"gitee.com/bychannel/musae/framework/baseconf"
	"gitee.com/bychannel/musae/framework/logger"
)

var gConf ServerConf

func GConf() *ServerConf {
	return &gConf
}

func Base() *baseconf.BaseConf {
	return &gConf.Base
}

func DDos() *DDosConf {
	return &gConf.DDos
}

func UGC() *UGCConf {
	return &gConf.UGC
}

func Login() *LoginConf {
	return &gConf.Login
}

func GMT() *GMTConf {
	return &gConf.GMT
}

func Question() *QuestionConf {
	return &gConf.Question
}

func SrvAddr() *SrvAddrConf {
	return &gConf.SrvAddr
}

type ServerConf struct {
	Base     baseconf.BaseConf `json:"BaseConf"`     // 基础配置
	DDos     DDosConf          `json:"DDosConf"`     // 负载配置
	UGC      UGCConf           `json:"UGCConf"`      // ugc机器审核配置
	Login    LoginConf         `json:"LoginConf"`    // 版本校验
	GMT      GMTConf           `json:"GMTConf"`      // gmt配置
	Bill     BillConf          `json:"BillConf"`     // bill配置
	Question QuestionConf      `json:"QuestionConf"` // 问卷配置
	SrvAddr  SrvAddrConf       `json:"SrvAddrConf"`  // 服务地址
	Sdk      SdkConf           `json:"SdkConf"`      // sdk配置
	TapTap   TaptapConf        `json:"TaptapConf"`   // taptap配置
	OSS      OSSConf           `json:"OSSConf"`      // OSS配置
}

func (s *ServerConf) BaseConf() *baseconf.BaseConf {
	return &s.Base
}

func LoadConf(path string, params ...string) error {
	var data []byte
	var err error

	if len(params) > 0 {
		data = []byte(params[0])
	} else {
		data, err = ioutil.ReadFile(path)
		if err != nil {
			logger.Error("[LoadConf] read fail ", path, ",", err)
			return err
		}
	}

	err = json.Unmarshal(data, &gConf)
	if err != nil {
		logger.Error("[LoadConf] parse fail ", path, ",", err)
		return err
	}
	baseconf.Iconf = &gConf

	return nil
}

type DDosConf struct {
	TimeInterval  uint32 `json:"timeInterval"`  // 检测时间间隔，单位秒  0表示不检测
	LoginLimitNum uint32 `json:"loginLimitNum"` // 每秒登陆频率上限，0表示不限
	TotalUserNum  uint32 `json:"totalUserNum"`  // 每秒登录用户数量上限 0表示不限
	LimitPktNum   uint32 `json:"limitPktNum"`   // 每间隔限制数据包量, 0表示不限
	LimitByteNum  uint32 `json:"limitByteNum"`  // 每间隔限制数据流量, 0表示不限
}

type UGCConf struct {
	BaseUrl   string `json:"baseUrl"`   // 机审url
	ApiPath   string `json:"apiPath"`   // 接口path
	SecretId  string `json:"secretId"`  // appid
	SecretKey string `json:"secretKey"` // appkey
	AccessKey string `json:"accessKey"` // accessKey
	Switch    int    `json:"switch"`    // 开关
}

type GMTConf struct {
	ApiSecret   string   `json:"apiSecret"`   // 验签密钥
	IsIpWhite   bool     `json:"isIpWhite"`   // 是否开启IP白名单
	IpWhiteList []string `json:"ipWhiteList"` // ip白名单
}

type BillConf struct {
	ApiSecret   string   `json:"apiSecret"`   // 验签密钥
	IsIpWhite   bool     `json:"isIpWhite"`   // 是否开启IP白名单
	IpWhiteList []string `json:"ipWhiteList"` // ip白名单
}

type QuestionConf struct {
	BaseUrl   string `json:"baseUrl"`   // 基础url
	ClientKey string `json:"clientKey"` // url签名key
	SecretKey string `json:"secretKey"` // 发奖接口签名key
}

type SdkConf struct {
	GameId       string `json:"gameId"`       // 游戏id
	LilithAppId  string `json:"lilithAppId"`  // 游戏应用id
	PlatName     string `json:"platName"`     // 发行渠道
	ServerRegion string `json:"serverRegion"` // 研发服务器大区
	Phase        string `json:"phase"`        // 产品阶段
}

type TaptapConf struct {
	BaseUrl  string `json:"baseUrl"`  // url
	ClientId string `json:"clientId"` // 应用id
}

type OSSConf struct {
	Endpoint      string `json:"endpoint"`
	AccessKey     string `json:"accessKey"`
	AccessSecret  string `json:"accessSecret"`
	VersionBucket string `json:"versionBucket"`
	DownPath      string `json:"downPath"`
}

type LoginConf struct {
	LoginBaseConf
}

type LoginBaseConf struct {
	GateUrl        string `json:"gateUrl"`
	GateUrlDev     string `json:"gateUrl_dev"`
	UserActorAhead bool   `json:"userActorAhead"`
}

type LoginServerConf struct {
	ServerConf
}

type GateServerConf struct {
	ServerConf
}

type BillServerConf struct {
	ServerConf
}

type SrvAddrConf struct {
	UpdateAddrARD []string `json:"updateAddrARD"`
	UpdateAddrIOS []string `json:"updateAddrIOS"`
	TCPAddr       []string `json:"tcpAddr"`
	HTTPAddr      []string `json:"httpAddr"`
}

package db

import "fmt"

const (
	// cfg:type:name
	// server.cfg
	KeyCfgServerOpenTime      = "cfg:server:opentime"      // 服务器开发时间
	KeyCfgServerNotice        = "cfg:server:notice"        // 服务器公告
	KeyCfgServerRollingNotice = "cfg:server:rollingnotice" // 跑马灯
	//KeyCfgReloadConf      = "cfg:server:conf"      // server.conf配置文件热更新
	KeyCfgReloadExcel         = "cfg:reload:excel"         // excel配置文件热更新
	KeyCfgReloadDirtyWord     = "cfg:reload:dirtyword"     // 静态脏字文件热更新
	KeyCfgServerRegisterLimit = "cfg:server:registerlimit" // 注册人数上限

	// version
	KeyCfgCVersionAndroid     = "cfg:version:client:android"      // client android 版本号 当前版本
	KeyCfgCVersionIOS         = "cfg:version:client:ios"          // client ios 版本号 当前版本
	KeyCfgCVersionAndroidMini = "cfg:version:client:android:mini" // client android 最低版本号
	KeyCfgCVersionIOSMini     = "cfg:version:client:ios:mini"     // client ios 最低版本号

	KeyCfgCVersionOnline  = "cfg:version:client:online"  // client android 版本号 对应的jenkins 流水号cfg:version:client:android:online::线上版本=> jenkins 流水号
	KeyCfgCVersionJenkins = "cfg:version:client:jenkins" // client android 版本号 对应的jenkins 流水号cfg:version:client:android:jenkins::jenkins 流水号=> 线上版本

	KeyCfgCVersionAndroidMax = "cfg:version:client:android:max" // 服务端维护的真正版本递增版本号
	KeyCfgCVersionIOSMax     = "cfg:version:client:ios:max"     // 服务端维护的真正版本递增版本号

	// login
	KeyCfgOpenWhiteList   = "cfg:login:openwhitelist"   // 账号白名单开关
	KeyCfgWhiteList       = "cfg:login:whitelist"       // 账号白名单
	KeyCfgOpenBlackList   = "cfg:login:openblacklist"   // 账号黑名单开关
	KeyCfgBlackList       = "cfg:login:blacklist"       // 账号黑名单
	KeyCfgLoginSwitch     = "cfg:login:loginswitch"     // 登录开关
	KeyCfgRegisterSwitch  = "cfg:login:registerswitch"  // 注册开关
	KeyCfgLoginBlackIp    = "cfg:login:loginblackip"    // 登录IP黑名单
	KeyCfgRegisterBlackIp = "cfg:login:registerblackip" // 注册IP黑名单

	// global
	KeyCfgGlobalDirtyWord     = "cfg:global:dirtyword"     // 新增脏字文本
	KeyCfgGlobalDeprecatedMsg = "cfg:global:deprecatedmsg" // 新增临时关闭协议
	KeyCfgGlobalCloseFunc     = "cfg:global:closefunc"     // 新增临时关闭功能

)

var KeyExcelCfg = func(excel string) string {
	return fmt.Sprintf("cfg:excel:%v", excel)
}

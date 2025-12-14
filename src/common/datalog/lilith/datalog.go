package lilith

// import (
//	"encoding/json"
//	"fmt"
//	"time"
//
//	"gitee.com/aniwar2/aniwar/src/common/conf"
//	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"
//	"gitee.com/aniwar2/aniwar/src/proto/pb"
//	"gitee.com/aniwar2/musae/framework/dlog"
//	"gitee.com/aniwar2/musae/framework/logger"
//	"gitee.com/aniwar2/musae/framework/threading"
//	"gitee.com/aniwar2/musae/framework/utils"
// )
//
// const (
//	FORMAT_DATETIME_LOG = "2006-01-02 15:04:05" // 日志上报统一时间格式
//	VERSION             = 1                     // 版本号
//	CHANNEL_STR_IOS     = "self-lilith-0.7"
//	CHANNEL_STR_PC      = "self-lilith-1"
//	OS_TYPE_ANDROID     = "android"
//	OS_TYPE_IOS         = "ios"
// )
//
// func WriteDataLog(data any) {
//	var (
//		startT = time.Now()
//	)
//
//	b, err := json.Marshal(data)
//	if err != nil {
//		logger.Warnf("[datalog] json marshal failed! err: %v , data: %+v", err, data)
//		return
//	}
//	dlog.Write(string(b))
//
//	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")
// }
//
// // 构建通用日志公共头信息
// func BuildHeadInfo(logType, uid string, device *pb.CliDeviceInfo) *SystemFieldInfo {
//	head := &SystemFieldInfo{
//		LogType:     logType,
//		Version:     VERSION,
//		EventTime:   time.Now().Format(FORMAT_DATETIME_LOG),
//		GameId:      conf.GConf().Sdk.GameId, // 游戏id
//		Pkg:         device.GetPkg(),         // 包名 带有-cbtN的后缀
//		Ip:          device.GetIp(),          // 玩家设备ip
//		Os:          device.GetOs(),          // 玩家设备系统
//		OsVersion:   device.GetOsVersion(),   // 系统版本号
//		AppVersion:  device.GetAppVersion(),  // 应用版本号
//		DeviceModel: device.GetDeviceModel(), // 设备产品型号
//		OpenId:      GetOpenId(uid),          // 平台账号,一个登录账号对应的唯一id
//		UserId:      uid,                     // 游戏内账号id
//		ServerId:    GetServerId(),           // 服务器id, 通服架构填0就行了
//	}
//	// 判定os类型再进行处理
//	if head.Os == OS_TYPE_ANDROID {
//		head.Channel = device.GetChannel() // 客户端提供
//		head.Idfa = ""                     // 明确不填值
//		head.AndroidId = device.GetAndroidId()
//		head.GoogleAid = device.GetGoogleAid()
//	} else if head.Os == OS_TYPE_IOS {
//		head.Channel = CHANNEL_STR_IOS
//		head.Idfa = device.GetIdfa()
//		head.AndroidId = "" // 明确不填值
//		head.GoogleAid = "" // 明确不填值
//	} else {
//		// 当pc处理
//		head.Channel = CHANNEL_STR_PC
//		head.Idfa = device.GetIdfa()
//		head.AndroidId = "" // 明确不填值
//		head.GoogleAid = "" // 明确不填值
//	}
//	logger.Debugf("埋点日志公共数据：%+v", head)
//	return head
// }
//
// // GetOpenId 拼接规则：appid@{platname}:{server_region}:{phase}-appuid
// func GetOpenId(uid string) string {
//	_, appid, appuid, err := myUtils.ScanfUID(uid)
//	if err != nil {
//		return ""
//	}
//	return fmt.Sprintf("%d@%s:%s:%s-%s", appid, conf.GConf().Sdk.PlatName, conf.GConf().Sdk.ServerRegion, conf.GConf().Sdk.Phase, appuid)
// }
//
// // GetServerId 拼接规则：{platname}:{server_region}:{phase}-server_id
// func GetServerId() string {
//	return fmt.Sprintf("%s:%s:%s-%d", conf.GConf().Sdk.PlatName, conf.GConf().Sdk.ServerRegion, conf.GConf().Sdk.Phase, 0)
// }
//
// func BuildCustomHeadInfo(logType, uid string, device *pb.CliDeviceInfo) *PropertyFieldInfo {
//	headInfo := BuildHeadInfo(logType, uid, device)
//	headInfo.LogType = "aniwar_" + logType
//	customHead := &PropertyFieldInfo{
//		SystemFieldInfo: headInfo,
//		UniqueId: utils.GenStrGUID(),
//	}
//	logger.Debugf("埋点自定义日志公共数据: %+v", customHead)
//	return customHead
// }
//
// // 服务启动
// func ServiceStart(appId, appVersion, clientVersion, rollingVersion, serverName string) error {
//
//	threading.RunSafe(func() {
//		WriteDataLog(&ServerStart{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_ServiceStart, serverName, nil),
//			AppId:          appId,          // 服务类型标识
//			AppVersion:     appVersion,     // 程序版本号
//			ClientVersion:  clientVersion,  // 客户端版本
//			RollingVersion: rollingVersion, // 滚动更新版本
//		})
//	})
//
//	return nil
// }
//
// // 服务退出
// func ServiceStop(appId, appVersion, clientVersion, rollingVersion, serverName string, liveTime int64) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&ServerStop{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_ServiceStop, serverName, nil),
//			AppId:          appId,          // 服务类型标识
//			AppVersion:     appVersion,     // 程序版本号
//			ClientVersion:  clientVersion,  // 客户端版本
//			RollingVersion: rollingVersion, // 滚动更新版本
//			LiveTime:       liveTime,       // 生存时间
//		})
//	})
//
//	return nil
// }
//
// // 服务定时器
// func ServerHourComm(appId, appVersion, clientVersion, rollingVersion, serverName string, liveTime int64) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&ServerHour{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_ServerHour, serverName, nil),
//			AppId:          appId,          // 服务类型标识
//			AppVersion:     appVersion,     // 程序版本号
//			ClientVersion:  clientVersion,  // 客户端版本
//			RollingVersion: rollingVersion, // 滚动更新版本
//			LiveTime:       liveTime,       // 生存时间
//		})
//	})
//
//	return nil
// }
//
// // 服务器配置事件
// func ConfeventComm(appId, appVersion, clientVersion, rollingVersion, serverName, eventId, eventData string) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&Confevent{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_Confevent, serverName, nil),
//			AppId:          appId,          // 服务类型标识
//			AppVersion:     appVersion,     // 程序版本号
//			ClientVersion:  clientVersion,  // 客户端版本
//			RollingVersion: rollingVersion, // 滚动更新版本
//			EventId:        eventId,        // 事件id
//			EventData:      eventData,      // 配置事件内容
//		})
//	})
//
//	return nil
// }
//
// // 服务热更新
// func ServeReloadComm(appId, appVersion, clientVersion, rollingVersion, serverName, files, fails string) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&ServeReload{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_ServeReload, serverName, nil),
//			AppId:          appId,          // 服务类型标识
//			AppVersion:     appVersion,     // 程序版本号
//			ClientVersion:  clientVersion,  // 客户端版本
//			RollingVersion: rollingVersion, // 滚动更新版本
//			Files:          files,          // 热更文件列表
//			Fails:          fails,          // 热更失败文件列表
//		})
//	})
//
//	return nil
// }
//
// // GM指令
// func GmCmdComm(cmd, param, gmuser string, user int, ip, serverName, result string, resultStatus int) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&GmCmd{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_GmCmd, serverName, nil),
//			Cmd:            cmd,          // 命令字符串
//			Param:          param,        // 命令参数 []string
//			GmUser:         gmuser,       // gm用户
//			User:           user,         // useractor id
//			Ip:             ip,           // 请求源IP地址
//			ResultStatus:   resultStatus, // 结果状态
//			Result:         result,       // 结果
//		})
//	})
//
//	return nil
// }
//
// // 用户容器上线下线
// func UserActorComm(id, serverName string, typeC, liveTime int64) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&UserActor{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_UserActor, serverName, nil),
//			Id:             id,       // id
//			Type:           typeC,    // 1：active; 0: deactive
//			LiveTime:       liveTime, // 生存时间
//		})
//	})
//
//	return nil
// }
//
// // DB读写失效
// func DbFailComm(key, db, typeC, serverName string) error {
//	threading.RunSafe(func() {
//		WriteDataLog(&DbFail{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_DbFail, serverName, nil),
//			Key:            key,   // 数据key
//			Db:             db,    // redis/mongo
//			Type:           typeC, // read/write
//		})
//	})
//
//	return nil
// }
//
// // login网络延迟
// func LoginDelayComm(uid string, device *pb.CliDeviceInfo, msgId int32, delay int64) {
//	if delay < conf.GConf().Base.DelayLogLimit {
//		return
//	}
//	threading.RunSafe(func() {
//		WriteDataLog(&LoginDelay{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_LoginDelay, uid, device),
//			MsgId:          msgId,
//			Delay:          delay,
//		})
//	})
// }
//
// // gate网络延迟
// func GateDelayComm(uid string, device *pb.CliDeviceInfo, msgId int32, delay int64) {
//	if delay < conf.GConf().Base.DelayLogLimit {
//		return
//	}
//	threading.RunSafe(func() {
//		WriteDataLog(&GateDelay{
//			PropertyFieldInfo: BuildCustomHeadInfo(LogType_GateDelay, uid, device),
//			MsgId:          msgId,
//			Delay:          delay,
//		})
//	})
// }

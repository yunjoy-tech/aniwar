package taptap

import (
	"encoding/json"
	"strconv"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/dlog"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/threading"
)

const (
	TAPTAP_CLIENT_ID = "queizhcadav89gshoq"
	CHANNEL_STR_PC   = "pc"
	OS_TYPE_ANDROID  = "android"
	OS_TYPE_IOS      = "ios"
	SDK_VERSION      = "3.16.0"
)

func WriteDataLog(eventName, uid string, tapUser *pb.TaptapUserInfo, event any) {
	var (
		startT = time.Now()
	)

	data := BuildSystemFieldInfo(eventName, uid, tapUser, event)

	b, err := json.Marshal(data)
	if err != nil {
		logger.Warnf("[datalog] json marshal failed! err: %v , data: %+v", err, data)
		return
	}
	dlog.Write(string(b))
	logger.WarnDelayf(time.Since(startT).Milliseconds(), "")
}

func BuildPropertyFieldInfo(device *pb.CliDeviceInfo) *PropertyFieldInfo {
	properties := &PropertyFieldInfo{
		OS:          device.GetOs(),
		Width:       device.GetWidth(),
		Height:      device.GetHeight(),
		DeviceModel: device.GetDeviceModel(),
		OsVersion:   device.GetOsVersion(),
		Network:     strconv.Itoa(int(device.GetNetwork())),
		Channel:     device.GetChannel(),
		AppVersion:  device.GetAppVersion(),
		SdkVersion:  SDK_VERSION,
		ServerId:    conf.GConf().Base.ServerId,
		ServerName:  conf.GConf().Base.ServerName,
		CreateTS:    time.Now().Unix(),
	}

	// 判定os类型处理
	if properties.OS == OS_TYPE_ANDROID {
		properties.DeviceId1 = ""                    // 传IMEI
		properties.DeviceId2 = device.GetGoogleAid() // 传google广告id
		properties.DeviceId3 = device.GetAndroidId() // 传android Id
		properties.DeviceId4 = ""                    // 传OAID
	} else if properties.OS == OS_TYPE_IOS {
		properties.DeviceId1 = device.GetIdfa() // 传IDFA
	}
	// logger.Debugf("埋点日志公共数据：%+v", properties)
	return properties
}

func BuildSystemFieldInfo(logName, uid string, tapUser *pb.TaptapUserInfo, event interface{}) *SystemFieldInfo {
	systemFields := &SystemFieldInfo{
		ClientId:   TAPTAP_CLIENT_ID,
		Type:       "track",
		DeviceId:   tapUser.GetDeviceId(),
		UserId:     uid,
		Name:       logName,
		Properties: event,
	}
	// logger.Debugf("埋点自定义日志公共数据: %+v", systemFields)
	return systemFields
}

// 新增全局邮件
func GlobalMailAdd(appId, appVersion, clientVersion, rollingVersion, serverName string, mailId int64) {
	threading.RunSafe(func() {
		e := &GlobalMailAddInfo{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
			MailId:            mailId,         // 新增的全局邮件id
		}
		WriteDataLog(LogType_GlobalMailAdd, serverName, nil, e)
	})
}

// 删除全局邮件
func GlobalMailDel(appId, appVersion, clientVersion, rollingVersion, serverName string, mailId int64) {
	threading.RunSafe(func() {
		e := &GlobalMailDelInfo{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
			MailId:            mailId,         // 删除的全局邮件id
		}
		WriteDataLog(LogType_GlobalMailDel, serverName, nil, e)
	})
}

// 玩家领取到系统邮件
func AddGlobalMail(appId, appVersion, clientVersion, rollingVersion, serverName string, addMails []*pb.PSysMailInfo) {
	mailIds := make([]int64, 0)
	for _, mail := range addMails {
		mailIds = append(mailIds, mail.Id)
	}

}

// 服务启动
func ServiceStart(appId, appVersion, clientVersion, rollingVersion, serverName string) {
	threading.RunSafe(func() {
		e := &ServerStart{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
		}
		WriteDataLog(LogType_ServiceStart, serverName, nil, e)
	})
}

// 服务退出
func ServiceStop(appId, appVersion, clientVersion, rollingVersion, serverName string, liveTime int64) {
	threading.RunSafe(func() {
		e := &ServerStop{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
			LiveTime:          liveTime,       // 生存时间
		}
		WriteDataLog(LogType_ServiceStop, serverName, nil, e)
	})
}

// 服务定时器
func ServerHourComm(appId, appVersion, clientVersion, rollingVersion, serverName string, liveTime int64) {
	threading.RunSafe(func() {
		e := &ServerHour{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
			LiveTime:          liveTime,       // 生存时间
		}
		WriteDataLog(LogType_ServerHour, serverName, nil, e)
	})
}

// 服务器配置事件
func ConfeventComm(appId, appVersion, clientVersion, rollingVersion, serverName, eventId, eventData string) {
	threading.RunSafe(func() {
		e := &Confevent{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
			EventId:           eventId,        // 事件id
			EventData:         eventData,      // 配置事件内容
		}
		WriteDataLog(LogType_Confevent, serverName, nil, e)
	})
}

// 服务热更新
func ServeReloadComm(appId, appVersion, clientVersion, rollingVersion, serverName, files, fails string) {
	threading.RunSafe(func() {
		e := &ServeReload{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			AppId:             appId,          // 服务类型标识
			AppVersion:        appVersion,     // 程序版本号
			ClientVersion:     clientVersion,  // 客户端版本
			RollingVersion:    rollingVersion, // 滚动更新版本
			Files:             files,          // 热更文件列表
			Fails:             fails,          // 热更失败文件列表
		}
		WriteDataLog(LogType_ServeReload, serverName, nil, e)
	})
}

// GM指令
func GmCmdComm(cmd, param, gmuser string, user int, ip, serverName, result string, resultStatus int) {
	threading.RunSafe(func() {
		e := &GmCmd{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			Cmd:               cmd,          // 命令字符串
			Param:             param,        // 命令参数 []string
			GmUser:            gmuser,       // gm用户
			User:              user,         // useractor id
			Ip:                ip,           // 请求源IP地址
			ResultStatus:      resultStatus, // 结果状态
			Result:            result,       // 结果
		}
		WriteDataLog(LogType_GmCmd, serverName, nil, e)
	})
}

// 用户容器上线下线
func UserActorComm(id, serverName string, typeC, liveTime int64) {
	threading.RunSafe(func() {
		e := &UserActor{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			Id:                id,       // id
			Type:              typeC,    // 1：active; 0: deactive
			LiveTime:          liveTime, // 生存时间
		}
		WriteDataLog(LogType_UserActor, serverName, nil, e)
	})
}

// DB读写失效
func DbFailComm(key, db, typeC, serverName string) {
	threading.RunSafe(func() {
		e := &DbFail{
			PropertyFieldInfo: BuildPropertyFieldInfo(nil),
			Key:               key,   // 数据key
			Db:                db,    // redis/mongo
			Type:              typeC, // read/write
		}
		WriteDataLog(LogType_DbFail, serverName, nil, e)
	})
}

// login网络延迟
func LoginDelayComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo, msgId int32, delay int64) {
	if delay < conf.GConf().Base.DelayLogLimit {
		return
	}
	threading.RunSafe(func() {
		e := &LoginDelay{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
			MsgId:             msgId,
			Delay:             delay,
		}
		WriteDataLog(LogType_LoginDelay, uid, tapUser, e)
	})
}

// gate网络延迟
func GateDelayComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo, msgId int32, delay int64) {
	if delay < conf.GConf().Base.DelayLogLimit {
		return
	}
	threading.RunSafe(func() {
		e := &GateDelay{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
			MsgId:             msgId,
			Delay:             delay,
		}
		WriteDataLog(LogType_GateDelay, uid, tapUser, e)
	})
}

// 版本获取
func GetVersionComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo, version string, platform string) {
	threading.RunSafe(func() {
		e := &GetVersion{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
			Version:           version,
			Platform:          platform,
		}
		WriteDataLog(LogType_GetVersion, uid, tapUser, e)
	})
}

// loginserver登录
func AccountLoginComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo, channel int32, extra string) {
	threading.RunSafe(func() {
		e := &AccountLogin{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
			Channel:           channel,
			Extra:             extra,
		}
		WriteDataLog(LogType_AccountLogin, uid, tapUser, e)
	})
}

// 长链进入
func TcpEnterComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo) {
	threading.RunSafe(func() {
		e := &TcpEnter{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
		}
		WriteDataLog(LogType_TcpEnter, uid, tapUser, e)
	})
}

// 断线重连
func ReconnectComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo) {
	threading.RunSafe(func() {
		e := &Reconnect{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
		}
		WriteDataLog(LogType_Reconnect, uid, tapUser, e)
	})
}

// 重复登录
func RepeatedLoginComm(uid string, tapUser *pb.TaptapUserInfo, device *pb.CliDeviceInfo) {
	threading.RunSafe(func() {
		e := &RepeatedLogin{
			PropertyFieldInfo: BuildPropertyFieldInfo(device),
		}
		WriteDataLog(LogType_RepeatedLogin, uid, tapUser, e)
	})
}

package logic

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"strings"
)

// GetUpdateVersionByChannel 获取当前版本号
func (s *GuideServer) GetUpdateVersionByChannel(channel string) (string, error) {
	key := db.KeyCfgCVersionAndroid
	if strings.ToLower(channel) == "ios" {
		key = db.KeyCfgCVersionIOS
	}
	return s.GetFromConfigCenter(key)
}

// GetMinVersionByChannel 获取最低版本号
func (s *GuideServer) GetMinVersionByChannel(channel string) (string, error) {
	key := db.KeyCfgCVersionAndroidMini
	if strings.ToLower(channel) == "ios" {
		key = db.KeyCfgCVersionIOSMini
	}
	return s.GetFromConfigCenter(key)
}

// GetJenkinsVersionByOnlineVersion 获取线上版本号对应的Jenkins 流水号
func (s *GuideServer) GetJenkinsVersionByOnlineVersion(version string) (string, error) {
	key := fmt.Sprintf("%s:%s", db.KeyCfgCVersionOnline, version)
	// value := s.Redis.Get(context.Background(), key)
	// return value.Result()
	return s.Server.GetFromConfigCenter(key)
}

func (s *GuideServer) GetPlatform(platform int32) string {
	if platform == 2 {
		return "ios"
	}
	return "android"
}

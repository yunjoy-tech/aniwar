package server

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"strconv"
	"strings"
)

// VerReq 请求json数据结构
type VerReq struct {
	AccountId string `json:"accountId"`
	Platform  int32  `json:"platform"` // 渠道， android,ios
	IsHttp    bool   `json:"isHttp"`
}

type VersionSupport struct {
	Channel     uint64 `json:"-"` // 渠道
	GameVersion uint64 `json:"-"` // 大版本号
	// Tag         uint32 `json:"-"` // 小版本号
	Latest uint32 `json:"-"` // 热更版本号
}

// guide http post test
// curl -H "Content-Type: application/json" -X POST -d '{"accountId": "wxw001", "isHttp": true }' "https://aniwar-ce.lilithgame.com:20001/api/version"

// Version 返回json数据结构
type Version struct {
	UpdateVersion string   `json:"updateVersion"` // 当前版本号
	DownloadPath  string   `json:"downloadPath"`  // jenkins 对应的版本号
	ServerAddr    []string `json:"serverAddr"`    // tcp 192.168.xxx.xxx http: [http/https]://aniwar-govtest.lilithgame.com
	TcpAddr       []string `json:"tcpAddr"`
	UpdateAddr    []string `json:"updateAddr"`
}

func ParseVersion(version string) *VersionSupport {

	forceStr := strings.Split(version, ".")
	if len(forceStr) != 3 {
		logger.Errorf("parse LoginConf err Version :%v", version)
		return nil
	}

	ch, err := strconv.ParseUint(forceStr[0], 10, 32)
	if err != nil {
		logger.Errorf("parse channel err channel :%v", forceStr[0])
		return nil
	}

	gameVersion, err := strconv.ParseUint(forceStr[1], 10, 32)
	if err != nil {
		logger.Errorf("parse gameVersion err gameVersion :%v", forceStr[1])
		return nil
	}

	// tag, err := strconv.ParseUint(forceStr[2], 10, 32)
	// if err != nil {
	//	logger.Newf("parse tag err tag :%v", forceStr[2])
	//	return nil
	// }
	//
	// latest, err := strconv.ParseUint(forceStr[3], 10, 32)
	// if err != nil {
	//	logger.Newf("parse latest err latest :%v", forceStr[3])
	//	return nil
	// }

	latest, err := strconv.ParseUint(forceStr[2], 10, 32)
	if err != nil {
		logger.Errorf("parse latest err latest :%v", forceStr[2])
		return nil
	}

	return &VersionSupport{
		Channel:     ch,
		GameVersion: gameVersion,
		// Tag:         uint32(tag),
		Latest: uint32(latest),
	}
}

// Verify true  需要更新
func (v *VersionSupport) Verify(des *VersionSupport) bool {
	if v.Channel != des.Channel {
		return true
	}
	if v.GameVersion != des.GameVersion {
		return true
	}
	if v.Latest > des.Latest {
		return true
	}

	// if v.Channel != des.Channel || v.GameVersion != des.GameVersion || v.Latest > des.Latest { // 90 < 89
	//	return false
	// }
	// return true
	return false // 不需要更新
}

func (s *Server) VersionCheck(version string) error {

	v := ParseVersion(version)
	if v == nil {
		return fmt.Errorf("GateServer:HandleLoginGame ParseVersion err version:%v", version)
	}

	if s.version.Verify(v) {
		return fmt.Errorf("version check Failed game Version :%v, input Version :%v", s.version, v)
	}

	return nil
}

func (s *Server) VersionCheckExt(platform, clientVersion string) error {
	var miniVersion string
	var err error
	if global.Platform_IOS == platform {
		miniVersion, err = s.GetFromConfigCenter(db.KeyCfgCVersionIOSMini)
		if err != nil {
			return err
		}
	} else {
		miniVersion, err = s.GetFromConfigCenter(db.KeyCfgCVersionAndroidMini)
		if err != nil {
			return err
		}
	}

	mini := ParseVersion(miniVersion)
	if mini == nil {
		return errorx.Newf("ParseVersion miniVersion err version:%v", miniVersion)
	}
	client := ParseVersion(clientVersion)
	if client == nil {
		return errorx.Newf("ParseVersion clientVersion err version:%v", clientVersion)
	}
	logger.Debugf("VersionCheckExt,clientVersion:%+v, miniVersion:%+v", client, mini)
	if mini.Verify(client) {
		return fmt.Errorf("version check failed mini:%v client:%v", miniVersion, clientVersion)
	}
	return nil
}

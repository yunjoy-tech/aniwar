package sdkconstant

import (
	"fmt"
	"gitee.com/bychannel/aniwar/src/common/conf"
)

const (
	Lilith_channel    = "lilith"
	Lilith_url_prefix = "https://apptest-develop.farlightgames.com"
	Lilith_login_url  = Lilith_url_prefix + "/api/sdk/verify_session"
)

// GenLilithUid
//
//	@Description: 生成莉莉丝渠道uid
//	@param uid 原uid
//	@return string 莉莉丝专用uid
func GenLilithUid(uid int) string {
	return fmt.Sprintf("%s_%s_%d", Lilith_channel, conf.GConf().Sdk.LilithAppId, uid)
}

// GetLilithChannel
//
//	@Description:  生成莉莉丝渠道
//	@return string
func GenLilithChannel() string {
	return fmt.Sprintf("%s_%s", Lilith_channel, conf.GConf().Sdk.LilithAppId)
}

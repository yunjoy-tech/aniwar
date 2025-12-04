package sdkconstant

import (
	"fmt"
)

const (
	KuaiBao_app_id    = 31567
	KuaiBao_channel   = "kuaibao"
	KuaiBao_Url_login = "https://api.3839app.com/kuaibao/android/devsdk.php"

	KuaiBao_c = "authgame"
	KuaiBao_a = "checkAccessToken"
	KuaiBao_v = "1546"

	LoginCheck_Code_Success = 100 // 登陆验证成功
)

func GenKuaiBaoUid(uid string) string {
	return fmt.Sprintf("%s_%s", KuaiBao_channel, uid)
}

func GenKuaiBaoChannel() string {
	return KuaiBao_channel
}

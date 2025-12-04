package logic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/http/request"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"net/http"
	"testing"
)

// 废弃  用gm指令实现
func TestReqUserInfo(t *testing.T) {
	// 构造数据
	reqData := UserInfoReq{
		ReqType:  "query_user",
		SvrId:    0,
		UserId:   0,
		UserName: "",
		OpenId:   "",
	}
	dataStr, err := json.Marshal(reqData)
	if err != nil {
		t.Error(err)
	}
	// base64加密
	baseStr := base64.StdEncoding.EncodeToString(dataStr)
	tempStr := fmt.Sprintf("%s%s", baseStr, conf.GConf().GMT.ApiSecret)
	signStr := utils.Md5Str(tempStr)
	data := []byte(fmt.Sprintf("%s%s", signStr, baseStr))

	// 构建请求
	req := request.New("http://localhost:19001")
	resp, err := req.Method(http.MethodPost).JSONBytesBody(data).Send("/api/sdk/gm/block/query")
	defer resp.Body.Close()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(resp.StatusCode)
}

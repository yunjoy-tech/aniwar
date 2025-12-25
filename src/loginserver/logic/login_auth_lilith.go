package logic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"gitee.com/aniwar2/aniwar/src/idipserver/logic"

	"gitee.com/aniwar2/musae/logger"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
)

// LilithLoginResp 莉莉丝登陆验证结果
type LilithLoginResp struct {
	Result      string         `json:"result"`
	Bind        bool           `json:"bind"`
	BindAccount []int          `json:"bind_account"`
	Identity    LilithIdentity `json:"identity"`
}

type LilithIdentity struct {
	IsAdult bool `json:"is_adult"`
	IsRn    bool `json:"is_rn"`
}

func (s *LoginServer) handleAuthLilith(appUid int, appToken string) (*LilithLoginResp, pb.ErrorCode) {
	if appToken == "" {
		logger.Warnf("lilith, 登陆请求验证, 无请求参数")
		return nil, pb.ErrorCode_Account_auth_fail
	}

	client := &http.Client{}
	reqParam := fmt.Sprintf("app_uid=%d&app_token=%s", appUid, appToken)
	logger.Warnf("lilith, 登陆请求验证:" + reqParam)
	lilithReq, err := http.NewRequest(http.MethodPost, "", strings.NewReader(reqParam))

	if err != nil {
		logger.Errorf(err.Error())
		return nil, pb.ErrorCode_InternalError
	}
	lilithReq.Header.Add("User-Agent", "apifox/1.0.0 (https://www.apifox.cn)")
	lilithReq.Header.Add("Accept", "*/*")
	lilithReq.Header.Add("Host", "apptest-develop.farlightgames.com")
	lilithReq.Header.Add("Connection", "keep-alive")
	lilithReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(lilithReq)
	if err != nil {
		logger.Errorf(err.Error())
		return nil, pb.ErrorCode_InternalError
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Errorf(err.Error())
		return nil, pb.ErrorCode_InternalError
	}
	logger.Warnf("lilith, 请求验证返回:" + string(body)) // {"bind":false,"bind_account":[0],"identity":{"is_adult":false,"is_rn":false},"result":"success"}

	resp := &LilithLoginResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		err = errors.Wrap(err, "验证结果失败")
		logger.Errorf(err.Error())
		return nil, pb.ErrorCode_Account_auth_fail
	}

	if resp.Result != logic.SUCCESS {
		err = errors.New(fmt.Sprintf("lilith, 验证结果失败, resp:%v", resp))
		logger.Errorf(err.Error())
		return nil, pb.ErrorCode_Account_auth_fail
	}
	return resp, pb.ErrorCode_Success
}

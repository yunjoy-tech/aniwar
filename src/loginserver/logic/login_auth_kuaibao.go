package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/pkg/errors"

	"github.com/yunjoy-tech/aniwar/src/common/sdkconstant"

	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/logger"
)

type KuaiBaoLoginReq struct {
	C           string `json:"c"`
	A           string `json:"a"`
	V           string `json:"v"`
	AppId       int32  `json:"app_id"`
	Uid         int32  `json:"uid"`
	AccessToken string `json:"access_token"`
}

// KuaiBaoLoginResp 好游快爆登陆验证结果
type KuaiBaoLoginResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (s *LoginServer) handleAuthKuaiBao(unionId, accessToken string) pb.ErrorCode {
	client := http.Client{}
	// reqParam := fmt.Sprintf("c=%s&a=%s&v=%s&app_id=%d&uid=%s&access_token=%s",
	//	sdkconstant.KuaiBao_c, sdkconstant.KuaiBao_a, sdkconstant.KuaiBao_v, sdkconstant.KuaiBao_app_id, unionId, accessToken)

	kuaiBaoUid, err := strconv.Atoi(unionId)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("好游快爆的账号不合法, unionId=%s", unionId))
		logger.Errorf(err.Error())
		return pb.ErrorCode_InvalidParam
	}

	reqParam := &KuaiBaoLoginReq{
		C:           sdkconstant.KuaiBao_c,
		A:           sdkconstant.KuaiBao_a,
		V:           sdkconstant.KuaiBao_v,
		AppId:       sdkconstant.KuaiBao_app_id,
		Uid:         int32(kuaiBaoUid),
		AccessToken: accessToken,
	}
	logger.Infof("好游快爆, 登陆请求验证: reqParam:%v", reqParam)

	reqParamBytes, err := json.Marshal(reqParam)
	if err != nil {
		logger.Errorf(err.Error())
		return pb.ErrorCode_InternalError
	}

	kuaiBaoReq, err := http.NewRequest(http.MethodPost, sdkconstant.KuaiBao_Url_login, bytes.NewBuffer(reqParamBytes))
	if err != nil {
		logger.Errorf(err.Error())
		return pb.ErrorCode_InternalError
	}

	res, err := client.Do(kuaiBaoReq)
	if err != nil {
		logger.Errorf(err.Error())
		return pb.ErrorCode_InternalError
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Errorf(err.Error())
		return pb.ErrorCode_InternalError
	}
	logger.Warnf("好游快爆, 请求验证返回:" + string(body))

	resp := &KuaiBaoLoginResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		err = errors.Wrap(err, "验证结果失败")
		logger.Errorf(err.Error())
		return pb.ErrorCode_InternalError
	}
	if resp.Code != sdkconstant.LoginCheck_Code_Success {
		err = errors.New(fmt.Sprintf("好游快爆, 验证结果失败, resp:%v", resp))
		logger.Errorf(err.Error())
		return pb.ErrorCode_Account_auth_fail
	}
	return pb.ErrorCode_Success
}

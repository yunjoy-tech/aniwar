package test

import (
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gitee.com/aniwar2/aniwar/src/common/sdkconstant"

	"gitee.com/aniwar2/musae/logger"
)

func Test_checkLogin(t *testing.T) {
	params := fmt.Sprintf("app_id=%s&app_uid=%d&app_token=%s", conf.SDK().LilithAppId, 10001, "ld3H2ITnI8nRXPeKldKl1jCNPTQDkM9m")
	rsp, err := http.Post(sdkconstant.Lilith_login_url, "application/x-www-form-urlencoded", strings.NewReader(url.QueryEscape(params)))
	defer rsp.Body.Close()
	if err != nil {
		logger.Errorf(err.Error())
		// return pb.ErrorCode_URL_GOT_ERROR
	}

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		logger.Errorf(err.Error())
		// return pb.ErrorCode_URL_GOT_ERROR
	}

	logger.Warnf("莉莉丝 校验登陆返回值:%s", string(body))
}

func Test_apiFox(t *testing.T) {
	client := &http.Client{}
	payload := strings.NewReader(fmt.Sprintf("app_id=%s&app_uid=%d&app_token=%s", conf.SDK().LilithAppId, 10001, "ld3H2ITnI8nRXPeKldKl1jCNPTQDkM9m"))
	req, err := http.NewRequest(http.MethodPost, sdkconstant.Lilith_login_url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("User-Agent", "apifox/1.0.0 (https://www.apifox.cn)")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Host", "apptest-develop.farlightgames.com")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

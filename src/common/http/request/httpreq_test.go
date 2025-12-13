package request_test

import (
	"fmt"
	"gitee.com/bychannel/aniwar/src/common/http/ctype"
	"gitee.com/bychannel/aniwar/src/common/http/request"
	"gitee.com/bychannel/aniwar/src/common/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHttpReq_Send(t *testing.T) {
	resp, err := request.New("https://httpbin.org").
		StringBody("hi").
		ContentType(ctype.JSON).
		WithHeaders(map[string]string{"coustom1": "value1"}).
		Send("/get")

	assert.NoError(t, err)
	sc := resp.StatusCode
	assert.True(t, request.IsOK(sc))
	assert.True(t, request.IsSuccessful(sc))
	assert.False(t, request.IsRedirect(sc))
	assert.False(t, request.IsForbidden(sc))
	assert.False(t, request.IsNotFound(sc))
	assert.False(t, request.IsClientError(sc))
	assert.False(t, request.IsServerError(sc))

	retMp := make(map[string]interface{})
	err = utils.DecodeReader(resp.Body, &retMp)
	assert.NoError(t, err)
	fmt.Println(retMp)
}

func TestHttpReq_MustSend(t *testing.T) {
	resp := request.New().
		BaseURL("https://httpbin.org").
		BytesBody([]byte("hi")).
		Method("POST").
		MustSend("/post")

	sc := resp.StatusCode
	assert.True(t, request.IsOK(sc))
	assert.True(t, request.IsSuccessful(sc))
}

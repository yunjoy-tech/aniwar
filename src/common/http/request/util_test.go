package request_test

import (
	"fmt"
	"gitee.com/bychannel/aniwar/src/common/http/request"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBuildBasicAuth(t *testing.T) {
	val := request.BuildBasicAuth("inhere", "abcd&123")

	assert.Contains(t, val, "Basic ")
}

func TestAddHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "inhere.xyz", nil)
	assert.NoError(t, err)

	request.AddHeaders(req, http.Header{
		"key0": []string{"val0"},
	})

	assert.Equal(t, "val0", req.Header.Get("key0"))
}

func TestRequestToString(t *testing.T) {
	req, err := http.NewRequest("GET", "inhere.xyz", nil)
	assert.NoError(t, err)

	request.AddHeaders(req, http.Header{
		"custom-key0": []string{"val0"},
	})

	vs := request.ToQueryValues(map[string]string{"field1": "value1", "field2": "value2"})

	req.Body = io.NopCloser(strings.NewReader(vs.Encode()))

	str := request.RequestToString(req)
	fmt.Println(str)

	assert.Contains(t, str, "GET inhere.xyz")
	assert.Contains(t, str, "Custom-Key0: val0")
	assert.Contains(t, str, "field1=value1")
}

func TestResponseToString(t *testing.T) {
	res := &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: 50,
		Header: http.Header{
			"Foo": []string{"Bar"},
		},
		Body: io.NopCloser(strings.NewReader("foo...bar")),
	}

	str := request.ResponseToString(res)
	fmt.Println(str)

	assert.Contains(t, str, "HTTP/1.1 200 OK")
	assert.Contains(t, str, "Foo: Bar")
	assert.Contains(t, str, "foo...bar")
}

func TestHeaderToStringMap(t *testing.T) {
	assert.Nil(t, request.HeaderToStringMap(nil))
	assert.Nil(t, request.HeaderToStringMap(http.Header{}))

	want := map[string]string{"key": "value; more"}
	assert.Equal(t, want, request.HeaderToStringMap(http.Header{
		"key": {"value", "more"},
	}))
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

func HttpPost(url string, buf []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("auth-token", "")
	client := &http.Client{Timeout: time.Duration(Timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 判定是否返回http错误
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HttpPost error status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

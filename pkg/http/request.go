package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"show-live/pkg/log"
)

func Request(url, method string, postBody interface{}, out interface{}) error {
	req, err := http.NewRequest(method, url, nil)
	if method != "GET" {
		v, _ := json.Marshal(postBody)
		body := bytes.NewBuffer(v)
		req, err = http.NewRequest(method, url, body)
	}
	if err != nil {
		return fmt.Errorf("new request error %v", err)
	}
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Content-Type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request error %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Errorf("读取响应body出错 %v", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("unmarshal response body error %v", err)
	}
	return nil
}

package controller

import (
	"io"
	"net/http"
	"strings"

	"github.com/roseforljh/opencrab/model"
	"github.com/roseforljh/opencrab/service"
)

func GetAuthHeader(key string) http.Header {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
	return header
}

func GetClaudeAuthHeader(key string) http.Header {
	header := http.Header{}
	header.Set("x-api-key", strings.TrimSpace(key))
	header.Set("anthropic-version", "2023-06-01")
	return header
}

func GetResponseBody(method string, url string, channel *model.Channel, headers http.Header) ([]byte, error) {
	proxy := ""
	if channel != nil {
		proxy = channel.GetSetting().Proxy
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

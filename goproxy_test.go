// Package goproxy 提供了一个简单易用的HTTP代理客户端实现
package goproxy

import (
	"net/http"
	"testing"
)

func TestGoProxy_SetProxy(t *testing.T) {
	c := New()
	err := c.SetProxy("http://127.0.0.1:8088")
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	resp, err := c.GetClient().Do(req)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp.StatusCode)
	}
	err = c.SetProxy("http://127.0.0.1:8081")
	if err != nil {
		t.Fatal(err)
	}
	req, _ = http.NewRequest("GET", "https://www.baidu.com", nil)
	resp, err = c.GetClient().Do(req)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp.StatusCode)
	}

	err = c.SetProxy("socks5://127.0.0.1:7891")
	if err != nil {
		t.Fatal(err)
	}
	req, _ = http.NewRequest("GET", "https://www.baidu.com", nil)
	resp, err = c.GetClient().Do(req)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp.StatusCode)
	}
	err = c.SetProxy("socks5://127.0.0.1:7890")
	if err != nil {
		t.Fatal(err)
	}
	req, _ = http.NewRequest("GET", "https://www.baidu.com", nil)
	resp, err = c.GetClient().Do(req)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp.StatusCode)
	}
}

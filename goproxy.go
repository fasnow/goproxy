// Package goproxy 提供了一个简单易用的HTTP代理客户端实现
package goproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// DefaultUA 定义默认的User-Agent字符串
const DefaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"

// DefaultTimeout 定义默认的HTTP请求超时时间
const DefaultTimeout = 20 * time.Second

// DefaultHTTPClient 创建一个默认的HTTP客户端实例
var DefaultHTTPClient = &http.Client{
	Timeout: DefaultTimeout,
}

// GoProxy 结构体定义了代理客户端的主要属性和方法
type GoProxy struct {
	client   *http.Client // HTTP客户端实例
	proxyUrl string       // 代理服务器URL
	mu       sync.Mutex   // 互斥锁，用于保护并发操作
}

func New() *GoProxy {
	return &GoProxy{
		client: &http.Client{
			Transport: &CustomTransport{
				GlobalHeader: http.Header{"User-Agent": []string{DefaultUA}},
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			},
			Timeout: DefaultTimeout,
			// 禁止重定向
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// CustomTransport 自定义传输层，用于处理HTTP请求的传输
type CustomTransport struct {
	// GlobalHeader 用于存储自定义的HTTP请求头
	// 在发送请求时会自动添加到每个请求中，对于单
	GlobalHeader http.Header     // 自定义请求头
	Transport    *http.Transport // 底层传输实现
}

// SetHeader 设置自定义请求头
// 参数:
//   - key: 请求头的键名
//   - value: 请求头的值
//
// 注意: 如果设置User-Agent，将会覆盖默认的User-Agent
func (c *CustomTransport) SetHeader(key, value string) {
	if c.GlobalHeader == nil {
		c.GlobalHeader = make(http.Header)
	}
	c.GlobalHeader.Set(key, value)
}

// AddHeader 添加自定义请求头
// 参数:
//   - key: 请求头的键名
//   - value: 请求头的值
//
// 注意: 如果添加User-Agent，将会覆盖默认的User-Agent
func (c *CustomTransport) AddHeader(key, value string) {
	if c.GlobalHeader == nil {
		c.GlobalHeader = make(http.Header)
	}
	c.GlobalHeader.Add(key, value)
}

// DelHeader 删除指定的请求头
// 参数:
//   - key: 要删除的请求头键名
func (c *CustomTransport) DelHeader(key string) {
	if c.GlobalHeader != nil {
		c.GlobalHeader.Del(key)
	}
}

// ClearHeaders 清除所有自定义请求头
func (c *CustomTransport) ClearHeaders() {
	c.GlobalHeader = make(http.Header)
}

// RoundTrip 实现了http.RoundTripper接口，用于处理HTTP请求
// 自动添加User-Agent和其他自定义请求头
func (c *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 复制原始请求头，避免修改原始请求
	req.Header = req.Header.Clone()

	// 以下请求头只能有单个值,使用Set
	singleValueHeaders := map[string]bool{
		"Authorization":     true,
		"Content-Type":      true,
		"Content-Length":    true,
		"Content-Encoding":  true,
		"Host":              true,
		"User-Agent":        true,
		"If-Match":          true,
		"If-None-Match":     true,
		"If-Modified-Since": true,
		"If-Range":          true,
		"Range":             true,
	}

	// 遍历自定义请求头
	for key, values := range c.GlobalHeader {
		for _, value := range values {
			if singleValueHeaders[key] {
				// req中的优先级更高
				if _, ok := req.Header[key]; ok {
					continue
				}
				// 对于单值请求头,使用Set覆盖
				req.Header.Set(key, value)
				break // 只使用第一个值
			} else {
				// 对于可以多值的请求头,使用Add追加
				// 如果key已存在则使用Add追加,否则使用Set设置
				if _, ok := req.Header[key]; ok {
					req.Header.Add(key, value)
				} else {
					req.Header.Set(key, value)
				}
				continue
			}
		}
	}

	return c.Transport.RoundTrip(req)
}

// SetProxy 设置代理服务器
// 支持HTTP、HTTPS和SOCKS5代理
// 参数s为空字符串时表示不使用代理
func (r *GoProxy) SetProxy(s string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ct := r.client.Transport.(*CustomTransport)
	if s == "" {
		r.proxyUrl = ""
		return nil // 不使用代理，设置成功
	}

	proxyURL, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("代理地址解析失败: %w", err)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		ct.Transport.Proxy = http.ProxyURL(proxyURL)
		ct.Transport.DialContext = nil
	case "socks5":
		var auth *proxy.Auth
		if proxyURL.User != nil {
			auth = &proxy.Auth{
				User:     proxyURL.User.Username(),
				Password: "",
			}
			if password, ok := proxyURL.User.Password(); ok {
				auth.Password = password
			}
		}
		dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
		if err != nil {
			return fmt.Errorf("创建SOCKS5代理失败: %w", err)
		}
		ct.Transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		ct.Transport.Proxy = nil
	default:
		return fmt.Errorf("不支持的代理协议: %s", proxyURL.Scheme)
	}
	r.proxyUrl = s
	return nil // 设置成功
}

// SetTimeout 设置HTTP请求的超时时间
func (r *GoProxy) SetTimeout(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.client.Timeout = timeout
}

// GetTimeout 获取HTTP请求的超时时间
func (r *GoProxy) GetTimeout() time.Duration {
	return r.client.Timeout
}

// GetClient 获取HTTP客户端实例
func (r *GoProxy) GetClient() *http.Client {
	return r.client
}

// String 返回当前代理服务器的URL字符串
func (r *GoProxy) String() string {
	return r.proxyUrl
}

// 添加一个设置全局请求头的方法
func (r *GoProxy) SetGlobalHeader(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.client.Transport.(*CustomTransport).SetHeader(key, value)
}

// 添加一个删除全局请求头的方法
func (r *GoProxy) DelGlobalHeader(key string) {
	r.client.Transport.(*CustomTransport).DelHeader(key)
}

// 添加一个清除所有全局请求头的方法
func (r *GoProxy) ClearGlobalHeaders() {
	r.client.Transport.(*CustomTransport).ClearHeaders()
}

// 添加一个获取全局请求头的方法
func (r *GoProxy) GetGlobalHeaders() http.Header {
	return r.client.Transport.(*CustomTransport).GlobalHeader
}

// 自动设置UserAgent
func (r *GoProxy) AutoSetUserAgent(autoSet bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.client.Transport.(*CustomTransport).GlobalHeader["User-Agent"]; ok {
		return
	}
	if autoSet {
		r.client.Transport.(*CustomTransport).SetHeader("User-Agent", DefaultUA)
	} else {
		r.client.Transport.(*CustomTransport).DelHeader("User-Agent")
	}
}

func (r *GoProxy) SetCheckRedirect(checkRedirect func(req *http.Request, via []*http.Request) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.client.CheckRedirect = checkRedirect
}

func (r *GoProxy) SetTransport(transport *http.Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ct := r.client.Transport.(*CustomTransport)
	if transport == nil {
		// 如果传入的transport为nil，则使用默认的transport避免panic
		ct.Transport = &http.Transport{}
		return
	}
	ct.Transport = transport
}

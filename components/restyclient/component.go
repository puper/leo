package restyclient

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

type Config struct {
	DisableKeepAlives bool `json:"disableKeepAlives"`

	Timeout time.Duration `json:"timeout"`

	MaxConnsPerHost     int `json:"maxConnsPerHost"`
	MaxIdleConnsPerHost int `json:"maxIdleConnsPerHost"`

	MaxIdleConns       int           `json:"maxIdleConns"`
	IdleConnTimeout    time.Duration `json:"idleConnTimeout"`
	InsecureSkipVerify bool          `json:"insecureSkipVerify"`
}

type Client struct {
	*resty.Client
	transport   *http.Transport
	ratelimiter resty.RateLimiter
}

// SetTimeout 与 Resty 的其他 Set 方法一致，更新当前客户端并保留已有配置。
func (me *Client) SetTimeout(timeout time.Duration) *Client {
	me.Client.SetTimeout(timeout)
	return me
}
func New(cfg *Config) *Client {
	transport := &http.Transport{
		DisableKeepAlives: cfg.DisableKeepAlives,

		MaxIdleConns:    cfg.MaxIdleConns,
		IdleConnTimeout: cfg.IdleConnTimeout,

		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
	}
	if cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	ratelimiter := rate.NewLimiter(1, 10)
	return &Client{
		Client:      resty.New().SetTimeout(cfg.Timeout).SetTransport(transport).SetRateLimiter(ratelimiter),
		transport:   transport,
		ratelimiter: ratelimiter,
	}
}

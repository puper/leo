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

func (me *Client) SetTimeout(timeout time.Duration) *Client {
	return &Client{
		Client:      resty.New().SetTimeout(timeout).SetTransport(me.transport).SetRateLimiter(me.ratelimiter),
		transport:   me.transport,
		ratelimiter: me.ratelimiter,
	}
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

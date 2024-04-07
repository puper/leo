package influxdb

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/influxdb/config"
	"github.com/puper/leo/engine"
)

func Builder(cfg *config.Config, configurers ...func(*Component) error) engine.Builder {
	return func() (any, error) {
		httpClient := &http.Client{
			Timeout: time.Second * time.Duration(60),
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: cfg.DailTimeout,
				}).DialContext,
				TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.InsecureSkipVerify,
				},
				MaxIdleConns:        cfg.MaxIdleConns,
				MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
				IdleConnTimeout:     cfg.IdleConnTimeout,
			},
		}
		c := influxdb2.NewClientWithOptions(cfg.ServerUrl, cfg.Token,
			influxdb2.DefaultOptions().
				SetUseGZip(cfg.UseGzip).
				SetHTTPClient(httpClient).
				SetApplicationName(cfg.AppName),
		)
		_, err := c.BucketsAPI().FindBucketByName(context.TODO(), cfg.Bucket)
		if err != nil {
			return nil, errors.WithMessage(err, "find bucket")
		}
		me := &Component{
			Client:     c,
			WriteApi:   c.WriteAPI(cfg.Org, cfg.Bucket),
			ViQueryApi: NewV1QueryApi(httpClient, c.ServerURL(), cfg.Token, cfg.AppName),
		}
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		return me, nil
	}
}

package influxdb

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"net/http"
	"sync"
)

type Component struct {
	httpClient *http.Client
	closeOnce  sync.Once
	Client     influxdb2.Client
	WriteApi   api.WriteAPI
	V1QueryApi *V1QueryApi
}

func (me *Component) Close() error {
	me.closeOnce.Do(func() {
		// Client.Close 已会 Flush；重复调用已关闭的 WriteAPI.Flush 会 panic。
		me.Client.Close()
		if me.httpClient != nil {
			me.httpClient.CloseIdleConnections()
		}
	})
	return nil
}

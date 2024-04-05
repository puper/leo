package influxdb

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Component struct {
	Client   influxdb2.Client
	WriteApi api.WriteAPI
}

func (me *Component) Close() error {
	me.WriteApi.Flush()
	me.Client.Close()
	return nil
}

package influxdb

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"testing"
)

func TestComponentCloseIsIdempotent(t *testing.T) {
	client := influxdb2.NewClient("http://127.0.0.1:1", "")
	c := &Component{Client: client, WriteApi: client.WriteAPI("org", "bucket")}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

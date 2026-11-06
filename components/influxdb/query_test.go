package influxdb

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryRejectsFailedOrMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		fail       bool
	}{
		{"empty", `{"results":[{"statement_id":0}]}`, false},
		{"normal", `{"results":[{"statement_id":0,"series":[{"name":"m","tags":{"host":"a"},"columns":["value"],"values":[[9007199254740993]]}]}]}`, false},
		{"invalid_json", `{`, true}, {"top_error", `{"error":"bad query"}`, true},
		{"statement_error", `{"results":[{"statement_id":0,"error":"database not found"}]}`, true},
		{"missing_results", `{}`, true}, {"null_statement", `{"results":[null]}`, true},
		{"short_row", `{"results":[{"series":[{"columns":["a","b"],"values":[[1]]}]}]}`, true},
		{"long_row", `{"results":[{"series":[{"columns":["a"],"values":[[1,2]]}]}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			api := NewV1QueryApi(srv.Client(), srv.URL, "", "test")
			_, err := api.Query("db", "select")
			if (err != nil) != tc.fail {
				t.Errorf("Query error=%v", err)
			}
			var records []struct {
				Value int64  `json:"value"`
				Host  string `json:"host"`
			}
			err = api.QueryRecords("db", "select", &records)
			if (err != nil) != tc.fail {
				t.Errorf("QueryRecords error=%v", err)
			}
			if tc.name == "normal" && (len(records) != 1 || records[0].Value != 9007199254740993 || records[0].Host != "a") {
				t.Errorf("records=%+v", records)
			}
		})
	}
}

func TestQueryValidatesTags(t *testing.T) {
	for _, tc := range []struct {
		name, tags string
		fail       bool
		wantHost   any
	}{
		{name: "missing"},
		{name: "null", tags: `null`},
		{name: "empty_object", tags: `{}`},
		{name: "normal", tags: `{"host":"a"}`, wantHost: "a"},
		{name: "empty_string", tags: `{"host":""}`, wantHost: ""},
		{name: "array", tags: `["host","a"]`, fail: true},
		{name: "empty_array", tags: `[]`, fail: true},
		{name: "number", tags: `7`, fail: true},
		{name: "boolean", tags: `false`, fail: true},
		{name: "string", tags: `"host"`, fail: true},
		{name: "number_value", tags: `{"host":7}`, fail: true},
		{name: "boolean_value", tags: `{"host":false}`, fail: true},
		{name: "null_value", tags: `{"host":null}`, fail: true},
		{name: "array_value", tags: `{"host":[]}`, fail: true},
		{name: "object_value", tags: `{"host":{}}`, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tagsField := ""
			if tc.tags != "" {
				tagsField = `,"tags":` + tc.tags
			}
			body := `{"results":[{"series":[{"columns":["value"],"values":[[7]]` + tagsField + `}]}]}`
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()
			api := NewV1QueryApi(srv.Client(), srv.URL, "", "test")
			_, err := api.Query("db", "select")
			if (err != nil) != tc.fail {
				t.Errorf("Query error=%v", err)
			}
			// 使用可接受任意值的目标，确保错误来自协议校验，而非业务字段类型碰巧不兼容。
			records := []map[string]any{{"untouched": true}}
			err = api.QueryRecords("db", "select", &records)
			if (err != nil) != tc.fail {
				t.Fatalf("QueryRecords error=%v records=%v", err, records)
			}
			if tc.fail {
				if len(records) != 1 || len(records[0]) != 1 || records[0]["untouched"] != true {
					t.Fatalf("invalid response changed destination: %v", records)
				}
				return
			}
			if len(records) != 1 || records[0]["value"] != float64(7) || records[0]["host"] != tc.wantHost {
				t.Fatalf("records=%v", records)
			}
		})
	}
}

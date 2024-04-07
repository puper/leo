package influxdb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"reflect"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

func NewV1QueryApi(httpClient *http.Client, url string, authorization string, userAgent string) *V1QueryApi {
	return &V1QueryApi{
		httpClient:    httpClient,
		url:           url,
		authorization: "Token " + authorization,
		userAgent:     userAgent,
	}
}

type V1QueryApi struct {
	httpClient    *http.Client
	url           string
	authorization string
	userAgent     string
}

type QueryResult struct {
	Results []*QueryStatement `json:"results"`
}

type QueryStatement struct {
	StatementId int            `json:"statement_id"`
	Series      []*QuerySeries `json:"series"`
}

type QuerySeries struct {
	Name    string            `json:"name"`
	Tags    map[string]string `json:"tags"`
	Columns []string          `json:"columns"`
	Values  [][]any           `json:"values"`
}

func (me *V1QueryApi) QueryRaw(db string, query string) ([]byte, error) {
	u, err := url.Parse(me.url)
	if err != nil {
		return nil, errors.WithMessage(err, "url.Parse")
	}
	u.Path = path.Join(u.Path, "query")
	q := u.Query()
	q.Set("db", db)
	q.Set("q", query)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, errors.WithMessage(err, "http.NewRequest")
	}
	if len(me.authorization) > 0 {
		req.Header.Set("Authorization", me.authorization)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", me.userAgent)
	}
	resp, err := me.httpClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "http.Do")
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "ReadFrom")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("http status code: %d, response: %v", resp.StatusCode, buf.String())
	}
	return buf.Bytes(), nil
}

func (me *V1QueryApi) Query(db string, query string) (*QueryResult, error) {
	b, err := me.QueryRaw(db, query)
	if err != nil {
		return nil, errors.WithMessage(err, "QueryRaw")
	}
	reply := &QueryResult{}
	if err := json.Unmarshal(b, reply); err != nil {
		return nil, errors.WithMessage(err, "json.Unmarshal")
	}
	return reply, nil
}

func (me *V1QueryApi) QueryRecords(db string, query string, reply any) error {
	rv := reflect.ValueOf(reply)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("reply must be a non-nil pointer")
	}
	b, err := me.QueryRaw(db, query)
	if err != nil {
		return errors.WithMessage(err, "QueryRaw")
	}
	records := []map[string]json.RawMessage{}
	for _, statement := range gjson.GetBytes(b, "results").Array() {
		for _, series := range statement.Get("series").Array() {
			tags := map[string]json.RawMessage{}
			columns := []string{}
			for k, v := range series.Get("tags").Map() {
				tags[k] = json.RawMessage(v.Raw)
			}
			for _, v := range series.Get("columns").Array() {
				columns = append(columns, v.String())
			}
			columnCount := len(columns)
			for _, values := range series.Get("values").Array() {
				record := map[string]json.RawMessage{}
				for k, v := range tags {
					record[k] = v
				}
				for i, v := range values.Array() {
					if i < columnCount {
						record[columns[i]] = json.RawMessage(v.Raw)
					}
				}
				records = append(records, record)
			}
		}
	}
	b, _ = json.Marshal(records)
	if err := json.Unmarshal(b, &reply); err != nil {
		return errors.WithMessage(err, "json.Unmarshal")
	}
	return nil
}

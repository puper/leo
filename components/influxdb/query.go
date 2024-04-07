package influxdb

import (
	"bytes"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
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

func (me *V1QueryApi) Query(db string, query string) ([]byte, error) {
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

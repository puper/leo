package service_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"server/internal/config"
	"server/internal/middleware"
	exampleRoute "server/internal/modules/example/route"
	"server/internal/modules/example/service"
	"server/migrates"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/puper/leo/v2/components/db"
	dbconfig "github.com/puper/leo/v2/components/db/config"
	"github.com/spf13/viper"
)

type integrationContainer struct {
	database *db.Db
	node     *snowflake.Node
}

func (c integrationContainer) GetDb() *db.Db            { return c.database }
func (c integrationContainer) GenerateID() snowflake.ID { return c.node.Generate() }

func TestMySQLIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set TEST_MYSQL_DSN to an isolated MySQL test database")
	}
	v := viper.New()
	v.Set("servers.server.driver", "mysql")
	v.Set("servers.server.master", dsn)
	v.Set("servers.server.migrate.tableName", "schema_migrations")
	var cfg dbconfig.Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal("invalid test database configuration")
	}
	value, err := db.Builder(&cfg, db.WithMigrateFs(migrates.Fs))()
	if err != nil {
		t.Fatal("test database initialization or migration failed")
	}
	database := value.(*db.Db)
	sqlDB, err := database.Write("server").DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.WithMigrateFs(migrates.Fs)(database); err != nil {
		t.Fatal("migration replay failed")
	}
	node, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatal(err)
	}
	sc := service.New(integrationContainer{database, node})
	original := config.Instance()
	t.Cleanup(func() { config.SetInstance(original) })
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	config.SetInstance(&config.Config{App: &config.App{Auth: &config.AuthConfig{JwtSecret: base64.URLEncoding.EncodeToString(secret), JwtExpire: 60}}})
	token, err := middleware.GenerateTokenWithUserID(1, "integration")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := exampleRoute.RouteFunc(sc)(r); err != nil {
		t.Fatal(err)
	}
	request := func(path, body string, authorized bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// 模拟 chunked 请求，不能依据 ContentLength 判断是否存在 JSON。
		req.ContentLength = -1
		if authorized {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		return res
	}
	created := request("/api/example/create", `{"name":" 示例 "}`, true)
	if created.Code != 200 {
		t.Fatalf("create status: %d", created.Code)
	}
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.ID == "" || response.Data.ID == "0" {
		t.Fatal("missing generated ID")
	}
	detail := request("/api/example/detail", `{"id":"`+response.Data.ID+`"}`, true)
	if detail.Code != 200 || !strings.Contains(detail.Body.String(), `"name":"示例"`) {
		t.Fatal("created record was not persisted")
	}
	for _, tc := range []struct {
		path, body string
		auth       bool
		status     int
	}{
		{"/api/example/detail", `{"id":"1"}`, true, 404},
		{"/api/example/create", `{}`, true, 400},
		{"/api/example/create", `{`, true, 400},
		{"/api/example/create", ``, true, 400},
		{"/api/example/create", `{"name":"example"}`, false, 401},
	} {
		if got := request(tc.path, tc.body, tc.auth).Code; got != tc.status {
			t.Errorf("%s: got %d want %d", tc.path, got, tc.status)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := sc.Call(ctx, "example.Get", json.RawMessage(`{"id":"`+response.Data.ID+`"}`)); err == nil {
		t.Error("canceled context ignored")
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	if got := request("/api/example/detail", `{"id":"`+response.Data.ID+`"}`, true).Code; got != 500 {
		t.Errorf("database failure status: %d", got)
	}
}

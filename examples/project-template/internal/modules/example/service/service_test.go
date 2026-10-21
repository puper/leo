package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"server/internal/derrors"
)

func TestRegisteredMethodsRejectInvalidArguments(t *testing.T) {
	sc := New(nil)
	for _, tc := range []struct{ method, body string }{
		{"example.Create", `{}`}, {"example.Create", `{"name":"   "}`},
		{"example.Create", `{"name":123}`}, {"example.Create", `{`},
		{"example.Create", `{"name":"` + strings.Repeat("界", 101) + `"}`},
		{"example.Get", `{}`}, {"example.Get", `{"id":"-1"}`},
		{"example.Get", `{"id":"bad"}`},
	} {
		t.Run(tc.method+tc.body, func(t *testing.T) {
			_, err := sc.Call(context.Background(), tc.method, json.RawMessage(tc.body))
			var detailed *derrors.DetailedError
			if !errors.As(err, &detailed) || detailed.Code != 400000 {
				t.Fatalf("expected parameter error, got %v", err)
			}
		})
	}
}

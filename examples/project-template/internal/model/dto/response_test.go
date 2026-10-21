package dto

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/internal/derrors"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResp_Send_HTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupResp      func(*Resp)
		expectedStatus int
	}{
		{
			name: "Ok should return 200",
			setupResp: func(r *Resp) {
				r.Ok(map[string]string{"key": "value"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Error should return 500",
			setupResp: func(r *Resp) {
				r.Error(derrors.NewInternalError(nil))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "UnAuthError should return 401",
			setupResp: func(r *Resp) {
				r.Error(derrors.NewUnAuthError(nil))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "ParamError should return 400",
			setupResp: func(r *Resp) {
				r.Error(derrors.NewParamError(nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "NotFoundError should return 404",
			setupResp: func(r *Resp) {
				r.Error(derrors.NewNotFoundError(nil))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "ForbiddenError should return 403",
			setupResp: func(r *Resp) {
				r.Error(derrors.NewForbiddenError(nil))
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			resp := NewResp(c)
			tt.setupResp(resp)

			if w.Code != tt.expectedStatus {
				t.Errorf("Send() HTTP status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestResp_Error_JSONBodyContainsErrorCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	resp := NewResp(c)
	resp.Error(derrors.NewInternalError(nil))

	var result Resp
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != http.StatusInternalServerError*1000 {
		t.Errorf("JSON body code = %d, want %d", result.Code, http.StatusInternalServerError*1000)
	}
}

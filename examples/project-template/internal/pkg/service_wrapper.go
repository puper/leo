package pkg

import (
	"encoding/json"

	"server/internal/derrors"
	"server/internal/logging"
	"server/internal/model/dto"

	"github.com/gin-gonic/gin"
	"github.com/puper/servicecaller"
)

func ServiceWrapper(servManager *servicecaller.ServiceCaller, serviceMethod string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var args json.RawMessage
		resp := dto.NewResp(c)
		if err := c.ShouldBindJSON(&args); err != nil {
			resp.Error(derrors.NewParamError(err))
			return
		}
		reply, err := servManager.Call(c.Request.Context(), serviceMethod, args)
		if err != nil {
			if _, ok := err.(*derrors.DetailedError); !ok {
				logging.GetLogger().With(
					"serviceMethod", serviceMethod,
					"path", c.Request.URL.Path,
					"err", err,
				).Error("service call failed")
			}
			resp.Error(err)
			return
		}
		resp.Ok(reply)
	}
}

package route

import (
	"server/internal/middleware"
	"server/internal/pkg"

	"github.com/gin-gonic/gin"
	"github.com/puper/servicecaller"
)

func RouteFunc(sc *servicecaller.ServiceCaller) func(*gin.Engine) error {
	return func(r *gin.Engine) error {
		group := r.Group("/api/example", middleware.AuthMiddleware())
		group.POST("/detail", pkg.ServiceWrapper(sc, "example.Get"))
		group.POST("/create", pkg.ServiceWrapper(sc, "example.Create"))
		return nil
	}
}

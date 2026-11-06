package route

import (
	"github.com/gin-gonic/gin"
)

var routeFuncs = []func(*gin.Engine) error{}

func Register(fn func(*gin.Engine) error) {
	routeFuncs = append(routeFuncs, fn)
}

func GetRouteFuncs() []func(*gin.Engine) error {
	return routeFuncs
}

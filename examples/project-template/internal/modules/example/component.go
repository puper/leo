package example

import (
	localRoute "server/internal/modules/example/route"
	"server/internal/modules/example/service"
	"server/internal/route"
)

type Component struct{}

func New(container Container) *Component {
	route.Register(localRoute.RouteFunc(service.New(container)))
	return &Component{}
}

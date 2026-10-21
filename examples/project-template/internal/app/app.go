package app

import (
	"server/internal/components"

	"github.com/bwmarrin/snowflake"
	"github.com/puper/leo/components/db"
	"github.com/puper/leo/components/zaplog/log"
	"github.com/puper/leo/engine"
	"go.uber.org/zap"
)

var (
	app       *engine.Engine
	container = new(Container)
)

type Container struct{}

func Set(e *engine.Engine)          { app = e }
func GetContainer() *Container      { return container }
func (me *Container) GetDb() *db.Db { return app.Get(components.Db).(*db.Db) }
func (me *Container) GetLogger(names ...string) *zap.SugaredLogger {
	return app.Get(components.Log).(*log.Log).Get(names...)
}
func (me *Container) GenerateID() snowflake.ID {
	return app.Get(components.IdGen).(*snowflake.Node).Generate()
}

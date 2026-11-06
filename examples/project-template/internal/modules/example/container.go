package example

import (
	"server/internal/ifaces/db"
	"server/internal/ifaces/idgen"
)

type Container interface {
	db.GetDb
	idgen.GenerateID
}

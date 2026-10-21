package db

import "github.com/puper/leo/v2/components/db"

type GetDb interface {
	GetDb() *db.Db
}

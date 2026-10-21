package db

import "github.com/puper/leo/components/db"

type GetDb interface {
	GetDb() *db.Db
}

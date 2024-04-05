package db

import (
	"embed"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func LoadMigrates(migrateFs embed.FS) (map[string][]*Migration, error) {
	migrates := map[string][]*Migration{}
	fs, err := migrateFs.ReadDir("sqls")
	if err != nil {
		return migrates, nil
	}
	for _, f := range fs {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if f.IsDir() {
			sqlFs, err := migrateFs.ReadDir("sqls/" + f.Name())
			if err != nil {
				return nil, errors.WithMessagef(err, "read dir sqls/%s", f.Name())
			}
			dbName := f.Name()
			ms := map[string]*Migration{}
			for _, sqlf := range sqlFs {
				if sqlf.IsDir() {
					continue
				}
				if strings.HasPrefix(sqlf.Name(), ".") || !strings.HasSuffix(sqlf.Name(), ".sql") {
					continue
				}
				tmp := strings.Split(sqlf.Name(), ".")
				if len(tmp) != 3 {
					continue
				}
				id := tmp[0]

				isUp := true
				if tmp[1] == "up" {
					isUp = true
				} else if tmp[1] == "down" {
					isUp = false
				} else {
					continue
				}
				sqlBytes, err := migrateFs.ReadFile("sqls/" + dbName + "/" + sqlf.Name())
				if err != nil {
					return nil, errors.WithMessagef(err, "read file sqls/%s/%s", dbName, sqlf.Name())
				}
				if _, ok := ms[id]; !ok {
					ms[id] = &Migration{
						ID: id,
					}
				}
				if isUp {
					ms[id].Migrate = func(tx *gorm.DB) error {
						return tx.Session(&gorm.Session{}).Exec(string(sqlBytes)).Error
					}
				} else {
					ms[id].Rollback = func(tx *gorm.DB) error {
						return tx.Session(&gorm.Session{}).Exec(string(sqlBytes)).Error
					}
				}
			}
			if len(ms) > 0 {
				migrates[dbName] = make([]*Migration, 0, len(ms))
				for _, m := range ms {
					migrates[dbName] = append(migrates[dbName], m)
				}
				sort.SliceStable(
					migrates[dbName],
					func(i, j int) bool {
						idI := strings.Split(migrates[dbName][i].ID, "_")
						idJ := strings.Split(migrates[dbName][j].ID, "_")
						return cast.ToInt(idI[0]) < cast.ToInt(idJ[0])
					},
				)
			}
		}
	}
	return migrates, nil
}

package db

import (
	"io"
	"sync"
	"testing"

	"gorm.io/gorm"
)

func TestDatabaseImplementsCloser(t *testing.T) {
	var dbInterface interface{} = &Db{}
	if _, ok := dbInterface.(io.Closer); !ok {
		t.Error("Db should implement io.Closer")
	}
}

func TestDatabaseConcurrentReadRace(t *testing.T) {
	db := &Db{
		wrappers: map[string]*Wrapper{
			"test": {
				master: &gorm.DB{},
				slave:  []*gorm.DB{},
			},
		},
	}

	for i := 0; i < 3; i++ {
		db.wrappers["test"].slave = append(db.wrappers["test"].slave, &gorm.DB{})
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = db.Read("test")
		}()
	}
	wg.Wait()
}

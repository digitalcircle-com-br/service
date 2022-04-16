package service

import (
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbs = make(map[string]*gorm.DB)
var mx sync.RWMutex

func saveDb(n string, d *gorm.DB) {
	mx.Lock()
	defer mx.Unlock()
	dbs[n] = d
}
func loadDb(n string) (d *gorm.DB, ok bool) {
	mx.RLock()
	defer mx.RUnlock()
	d, ok = dbs[n]
	return
}

func delDb(n string) {
	mx.Lock()
	defer mx.Unlock()
	delete(dbs, n)
}

func DB() (ret *gorm.DB, err error) {
	return DBN("default")
}

func DBN(n string) (ret *gorm.DB, err error) {
	ret, ok := loadDb(n)
	if !ok {

		dsn, lerr := DataHGet("dsn", n)

		if lerr != nil {
			err = lerr
			return
		}

		ret, lerr = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true, // disables implicit prepared statement usage
		}), &gorm.Config{})

		if lerr != nil {
			err = lerr
			return
		}

		lerr = ret.Raw("select 1+1").Error
		if lerr != nil {
			err = lerr
			return
		}
		Debug("DB: New  Connection: %s", n)
		saveDb(n, ret)

	}
	return
}

func DBClose(n string) error {
	db, ok := loadDb(n)
	if ok {
		rdb, err := db.DB()
		if err != nil {
			return err
		}
		err = rdb.Close()
		if err != nil {
			return err
		}
		Debug("DB: Closed connection: %s", n)
		delDb(n)
	}
	return nil
}

func DBCloseAll() {
	ks := make([]string, len(dbs))
	for k := range dbs {
		ks = append(ks, k)
	}
	for _, k := range ks {
		DBClose(k)
	}
}

// +build test

package config

import (
	_ "database/sql"
	"fmt"
	"path"
	"runtime"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func InitTest() {
	Conf = BuildDefaultConf()
	Conf.Core.PublicURL = "https://dandelion.to/"

	_, file, _, _ := runtime.Caller(1)
	dbFile := path.Dir(file) + "/test.db" // db put in current package's directory

	var err error
	DB, err = sqlx.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_loc=auto", dbFile))
	if err != nil {
		panic(err)
	}
}

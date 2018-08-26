package main

import (
	_ "database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// InitDatabase init database connection
func InitDatabase() (*sqlx.DB, error) {
	dbConf := Conf.Database
	db, err := sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4,utf8", dbConf.User, dbConf.Pass, dbConf.Host, dbConf.Port, dbConf.Name))

	if err != nil {
		return db, err
	}

	err = db.Ping()
	if err != nil {
		return db, err
	}

	db.SetMaxIdleConns(dbConf.MaxIdleConns)
	// for db invalid connection after EOF
	db.SetConnMaxLifetime(time.Second)

	// connect success
	return db, nil
}

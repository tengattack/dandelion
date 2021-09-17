// +build test

package config

import (
	_ "database/sql"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tengattack/dandelion/log"
)

// ReadQueries from mysql sql file
func ReadQueries(filePath string) (string, error) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	queries := string(raw)
	queries = queryFixPK(queries)
	queries = queryFixINT(queries)
	queries = queryFixCOMMENT(queries)
	queries = queryFixCREATETABLE(queries)
	// TODO: remove comment, add index
	return queries, nil
}

func queryFixPK(queries string) string {
	re := regexp.MustCompile(`(?i)\sAUTO_INCREMENT\s+PRIMARY\s+KEY(\s|,)`)
	re2 := regexp.MustCompile(`(?i)(\s)(AUTO_INCREMENT\s+PRIMARY\s+KEY)(\s|,)`)
	queries = re.ReplaceAllStringFunc(queries, func(s string) string {
		parts := re2.FindStringSubmatch(s)
		parts[2] = "PRIMARY KEY AUTOINCREMENT"
		return strings.Join(parts[1:], "")
	})
	return queries
}

func queryFixINT(queries string) string {
	re := regexp.MustCompile(`(?i)\s([a-zA-Z]*INT)(\s+|\s*\(\d+\)\s+)(UNSIGNED(:?\s|,))?`)
	re2 := regexp.MustCompile(`(?i)(\s)([A-Z]*INT)(\s+|\s*\(\d+\)\s+)(UNSIGNED(:?\s|,))?`)
	queries = re.ReplaceAllStringFunc(queries, func(s string) string {
		parts := re2.FindStringSubmatch(s)
		parts[2] = "INTEGER"
		parts[3] = " "
		parts[4] = ""
		return strings.Join(parts[1:], "")
	})
	return queries
}

func queryFixCOMMENT(queries string) string {
	re := regexp.MustCompile(`(?i)COMMENT\s+'.*?[^\\]'`)
	s := re.ReplaceAllString(queries, "")
	re = regexp.MustCompile(`(?i)COMMENT\s+".*?[^\\]"`)
	s = re.ReplaceAllString(s, "")
	return s
}

func queryFixCREATETABLE(queries string) string {
	// remove 'KEY idx_appid_status (`app_id`, `status`)'
	re := regexp.MustCompile(`(?i)CREATE TABLE\s+.*?\([\s\S]*?,\s*(KEY|INDEX|UNIQUE).*?\)`)
	re2 := regexp.MustCompile(`(?i)(CREATE TABLE\s+.*?\([\s\S]*?)(,\s*(KEY|INDEX|UNIQUE).*?\))`)
	loop := true
	for loop {
		loop = false
		queries = re.ReplaceAllStringFunc(queries, func(s string) string {
			loop = true
			parts := re2.FindStringSubmatch(s)
			parts[2] = ""
			parts[3] = "" // sub match
			return strings.Join(parts[1:], "")
		})
	}

	// remove 'ENGINE=InnoDB CHARACTER SET=utf8 COLLATE=utf8_general_ci'
	re = regexp.MustCompile(`(?i)CREATE TABLE\s+.*?\([\s\S]*?\S\s*\)[^\)]*?;`)
	re2 = regexp.MustCompile(`(?i)(CREATE TABLE\s+.*?\([\s\S]*?\S\s*\))([^\)]*?)(;)`)
	queries = re.ReplaceAllStringFunc(queries, func(s string) string {
		parts := re2.FindStringSubmatch(s)
		parts[2] = ""
		return strings.Join(parts[1:], "")
	})
	return queries
}

func InitTest() {
	Conf = BuildDefaultConf()
	Conf.Core.PublicURL = "https://dandelion.to/"
	Conf.Kafka.Enabled = false

	err := log.InitLog(&Conf.Log)
	if err != nil {
		panic(err)
	}

	_, file, _, _ := runtime.Caller(1)
	dbFile := path.Dir(file) + "/test.db" // db put in current package's directory

	DB, err = sqlx.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_loc=auto", dbFile))
	if err != nil {
		panic(err)
	}

	_, currentFile, _, _ := runtime.Caller(0)
	queries, err := ReadQueries(path.Join(path.Dir(currentFile), "../../../data/schema.sql"))
	if err != nil {
		panic(queries)
	}
	_, err = DB.Exec(queries)
	if err != nil {
		panic(err)
	}
}

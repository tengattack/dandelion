package main

import (
	"flag"
	"fmt"
	"os"

	"../../log"
	"../../mq"
	"../../repository"
	"./config"
	"github.com/jmoiron/sqlx"
)

var (
	// Conf is the main config
	Conf config.Config
	// Repo is git repository
	Repo *repository.Repository
	// DB is Database
	DB *sqlx.DB
	// MQ is MessageQueue
	MQ *mq.MessageQueue
)

func main() {
	configPath := flag.String("config", "", "config file")
	showVerbose := flag.Bool("verbose", false, "show verbose debug log")
	showHelp := flag.Bool("help", false, "show help message")
	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}
	if *configPath == "" {
		fmt.Fprint(os.Stderr, "Please specify a config file\n\n")
		flag.Usage()
		os.Exit(1)
	}

	conf, err := config.LoadConfig(*configPath)
	if err != nil {
		panic(err)
	}
	if *showVerbose {
		conf.Log.AccessLevel = "debug"
		conf.Log.ErrorLevel = "debug"
	}
	Conf = conf

	err = log.InitLog(&Conf.Log)
	if err != nil {
		panic(err)
	}

	Repo, err = repository.InitRepository(conf.Core.RepositoryPath, conf.Core.RemoteURL)
	if err != nil {
		log.LogError.Errorf("init repository error: %v", err)
		panic(err)
	}

	db, err := InitDatabase()
	if err != nil {
		log.LogError.Errorf("database error: %v", err)
		panic(err)
	}
	defer db.Close()
	DB = db

	m, err := mq.NewProducer(Conf.Kafka.Servers, Conf.Kafka.Topic)
	if err != nil {
		log.LogError.Errorf("database error: %v", err)
		panic(err)
	}
	defer m.Close()
	MQ = m

	err = RunHTTPServer()
	if err != nil {
		log.LogError.Errorf("http server error: %v", err)
		panic(err)
	}
}
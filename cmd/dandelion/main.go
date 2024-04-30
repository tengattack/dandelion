package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	_ "go.uber.org/automaxprocs"

	"github.com/tengattack/dandelion/client"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/log"
	"github.com/tengattack/dandelion/mq"
	"github.com/tengattack/dandelion/repository"
	"github.com/tengattack/tgo/logger"
)

var (
	// Version control for dandelion
	Version = "0.0.1-dev"
)

func main() {
	client.SetVersion(Version)
	var defaultConfigPath string
	if runtime.GOOS == "windows" {
		defaultConfigPath = "config.yml"
	} else {
		defaultConfigPath = "/etc/dandelion/config.yml"
	}
	configPath := flag.String("config", defaultConfigPath, "config file")
	showVerbose := flag.Bool("verbose", false, "show verbose debug log")
	showHelp := flag.Bool("help", false, "show help message")
	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "Please specify a config file")
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
	config.Conf = conf

	err = log.InitLog(&config.Conf.Log)
	if err != nil {
		panic(err)
	}
	client.SetLogger(log.GetClientLogger())

	config.Repo, err = repository.InitRepository(&conf.Repository)
	if err != nil {
		logger.Errorf("init repository error: %v", err)
		panic(err)
	}

	db, err := InitDatabase(&config.Conf.Database)
	if err != nil {
		logger.Errorf("database error: %v", err)
		panic(err)
	}
	defer db.Close()
	config.DB = db

	if config.Conf.Kafka.Enabled {
		m, err := mq.NewProducer(config.Conf.Kafka.Servers, config.Conf.Kafka.Topic)
		if err != nil {
			logger.Errorf("database error: %v", err)
			panic(err)
		}
		defer m.Close()
		config.MQ = m
	}

	err = RunHTTPServer()
	if err != nil {
		logger.Errorf("http server error: %v", err)
		panic(err)
	}
}

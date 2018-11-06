package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/tengattack/dandelion/client"
	"github.com/tengattack/dandelion/cmd/dandelion-seed/config"
	"github.com/tengattack/dandelion/log"
	"github.com/tengattack/dandelion/mq"
)

var (
	// Conf is the client config
	Conf config.Config
	// Client is the dandelion client instance
	Client *client.DandelionClient
)

func main() {
	var defaultConfigPath string
	if runtime.GOOS == "windows" {
		defaultConfigPath = "config.yml"
	} else {
		defaultConfigPath = "/etc/dandelion-seed/config.yml"
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
	Conf = conf

	err = log.InitLog(&Conf.Log)
	if err != nil {
		panic(err)
	}

	Client, err = client.NewDandelionClient(Conf.Dandelion.URL)
	if err != nil {
		log.LogError.Errorf("dandelion init error: %v", err)
		panic(err)
	}
	defer Client.Close()

	err = CheckCurrentConfigs()
	if err != nil {
		log.LogError.Errorf("check current configs error: %v", err)
		panic(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	go RunHTTPServer()

	if Conf.Kafka.Enabled {
		m, err := mq.NewConsumer(Conf.Kafka.Servers, Conf.Kafka.Topic, Conf.Kafka.GroupID, sigchan)
		if err != nil {
			log.LogError.Errorf("check current configs error: %v", err)
			panic(err)
		}
		defer m.Close()

		for message := range m.Messages() {
			log.LogAccess.Infof("received message: %s", message)
			err = HandleMessage(message)
			if err != nil {
				log.LogError.Errorf("handle message error: %v", err)
			}
		}
	} else {
		<-sigchan
	}
}

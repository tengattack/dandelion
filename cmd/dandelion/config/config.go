package config

import (
	"io/ioutil"
	"runtime"

	"github.com/tengattack/tgo/log"

	"gopkg.in/yaml.v2"
)

// Config is config structure.
type Config struct {
	Core     SectionCore     `yaml:"core"`
	Log      log.Config      `yaml:"log"`
	Database SectionDatabase `yaml:"database"`
	Kafka    SectionKafka    `yaml:"kafka"`
}

// SectionCore is sub section of config.
type SectionCore struct {
	Enabled        bool   `yaml:"enabled"`
	Address        string `yaml:"address"`
	Port           int    `yaml:"port"`
	Mode           string `yaml:"mode"`
	PublicURL      string `yaml:"public_url"`
	RepositoryPath string `yaml:"repository_path"`
	RemoteURL      string `yaml:"remote_url"`
}

// SectionDatabase is sub section of config.
type SectionDatabase struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Name         string `yaml:"name"`
	User         string `yaml:"user"`
	Pass         string `yaml:"pass"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// SectionKafka is sub section of config.
type SectionKafka struct {
	Enabled bool     `yaml:"enabled"`
	Topic   string   `yaml:"topic"`
	Servers []string `yaml:"servers"`
}

// BuildDefaultConf is default config setting.
func BuildDefaultConf() Config {
	var conf Config

	// Core
	conf.Core.Enabled = true
	conf.Core.Address = ""
	conf.Core.Port = 9012
	conf.Core.Mode = "release"
	conf.Core.PublicURL = ""
	conf.Core.RepositoryPath = ""
	conf.Core.RemoteURL = ""

	// Log
	conf.Log.Format = "string"
	conf.Log.AccessLog = "stdout"
	conf.Log.AccessLevel = "debug"
	conf.Log.ErrorLog = "stderr"
	conf.Log.ErrorLevel = "error"
	conf.Log.Agent.Enabled = false

	// Database
	conf.Database.Host = "127.0.0.1"
	conf.Database.Port = 3306
	conf.Database.Name = ""
	conf.Database.User = ""
	conf.Database.Pass = ""
	conf.Database.MaxIdleConns = runtime.NumCPU()

	// Kafka
	conf.Kafka.Enabled = true
	conf.Kafka.Topic = ""

	return conf
}

// LoadConfig load config from file
func LoadConfig(confPath string) (Config, error) {
	conf := BuildDefaultConf()

	configFile, err := ioutil.ReadFile(confPath)

	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(configFile, &conf)

	if err != nil {
		return conf, err
	}

	return conf, nil
}

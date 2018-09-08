package config

import (
	"io/ioutil"
	"os"

	"github.com/tengattack/tgo/log"

	"gopkg.in/yaml.v2"
)

// Config is config structure.
type Config struct {
	API       SectionAPI       `yaml:"api"`
	Log       log.Config       `yaml:"log"`
	Dandelion SectionDandelion `yaml:"dandelion"`
	Kafka     SectionKafka     `yaml:"kafka"`
	Configs   []SectionConfig  `yaml:"configs"`
}

// SectionAPI is sub section of config.
type SectionAPI struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
	Mode    string `yaml:"mode"`
}

// SectionLog is sub section of config.
type SectionLog struct {
	Format      string          `yaml:"format"`
	AccessLog   string          `yaml:"access_log"`
	AccessLevel string          `yaml:"access_level"`
	ErrorLog    string          `yaml:"error_log"`
	ErrorLevel  string          `yaml:"error_level"`
	Agent       SectionLogAgent `yaml:"agent"`
}

// SectionLogAgent is sub section of SectionLog.
type SectionLogAgent struct {
	Enabled    bool   `yaml:"enabled"`
	DSN        string `yaml:"dsn"`
	AppID      string `yaml:"app_id"`
	Host       string `yaml:"host"`
	InstanceID string `yaml:"instance_id"`
}

// SectionDandelion is sub section of config.
type SectionDandelion struct {
	URL string `yaml:"url"`
}

// SectionKafka is sub section of config.
type SectionKafka struct {
	Enabled bool     `yaml:"enabled"`
	Topic   string   `yaml:"topic"`
	GroupID string   `yaml:"group_id"`
	Servers []string `yaml:"servers"`
}

// SectionConfig is sub section of config.
type SectionConfig struct {
	ID         int
	AppID      string   `yaml:"app_id"`
	Path       string   `yaml:"path"`
	Chown      string   `yaml:"chown"`
	Chmod      string   `yaml:"chmod"`
	MetaFiles  []string `yaml:"meta_files"`
	ExecReload string   `yaml:"exec_reload"`
}

// BuildDefaultConf is default config setting.
func BuildDefaultConf() Config {
	var conf Config

	// API
	conf.API.Enabled = true
	conf.API.Address = ""
	conf.API.Port = 9013
	conf.API.Mode = "release"

	// Log
	conf.Log.Format = "string"
	conf.Log.AccessLog = "stdout"
	conf.Log.AccessLevel = "debug"
	conf.Log.ErrorLog = "stderr"
	conf.Log.ErrorLevel = "error"
	conf.Log.Agent.Enabled = false

	// Dandelion
	conf.Dandelion.URL = "http://127.0.0.1:9012"

	// Kafka
	conf.Kafka.Enabled = true
	conf.Kafka.Topic = ""
	conf.Kafka.GroupID = ""

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

	if conf.Kafka.GroupID == "" {
		hostname := os.Getenv("HOST")
		if hostname == "" {
			hostname, _ = os.Hostname()
		}
		conf.Kafka.GroupID = hostname
	}

	// mark id
	for i := range conf.Configs {
		conf.Configs[i].ID = i
	}

	return conf, nil
}

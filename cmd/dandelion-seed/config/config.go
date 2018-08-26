package config

import (
	"io/ioutil"

	"../../../log"

	"gopkg.in/yaml.v2"
)

// Config is config structure.
type Config struct {
	Log       log.Config       `yaml:"log"`
	Dandelion SectionDandelion `yaml:"dandelion"`
	Kafka     SectionKafka     `yaml:"kafka"`
	Configs   []SectionConfig  `yaml:"configs"`
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
	Topic   string   `yaml:"topic"`
	GroupID string   `yaml:"group_id"`
	Servers []string `yaml:"servers"`
}

// SectionConfig is sub section of config.
type SectionConfig struct {
	AppID      string   `yaml:"app_id"`
	Path       string   `yaml:"path"`
	Chown      string   `yaml:"chown"`
	MetaFiles  []string `yaml:"meta_files"`
	ExecReload string   `yaml:"exec_reload"`
}

// BuildDefaultConf is default config setting.
func BuildDefaultConf() Config {
	var conf Config

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
	conf.Kafka.Topic = ""
	conf.Kafka.GroupID = ""

	return conf
}

// LoadConfig load config from file
func LoadConfig(confPath string) (Config, error) {
	var conf Config

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

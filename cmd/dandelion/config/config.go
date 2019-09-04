package config

import (
	"io/ioutil"
	"runtime"

	"github.com/tengattack/tgo/log"

	"gopkg.in/yaml.v2"
)

// Config is config structure.
type Config struct {
	Core       SectionCore       `yaml:"core"`
	Log        log.Config        `yaml:"log"`
	Database   SectionDatabase   `yaml:"database"`
	Kafka      SectionKafka      `yaml:"kafka"`
	Kubernetes SectionKubernetes `yaml:"kubernetes"`
	Registry   SectionRegistry   `yaml:"registry"`
	Webhook    SectionWebhook    `yaml:"webhook"`
}

// SectionCore is sub section of config.
type SectionCore struct {
	Enabled        bool   `yaml:"enabled"`
	Address        string `yaml:"address"`
	Port           int    `yaml:"port"`
	SSL            bool   `yaml:"ssl"`
	SSLPort        int    `yaml:"ssl_port"`
	CertPath       string `yaml:"cert_path"`
	CertKeyPath    string `yaml:"cert_key_path"`
	Mode           string `yaml:"mode"`
	PublicURL      string `yaml:"public_url"`
	RepositoryPath string `yaml:"repository_path"`
	ArchivePath    string `yaml:"archive_path"`
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

// SectionKubernetes is sub section of config.
type SectionKubernetes struct {
	InCluster bool   `yaml:"in_cluster"`
	Config    string `yaml:"config"`
	Namespace string `yaml:"namespace"`
}

// SectionRegistry is sub section of config.
type SectionRegistry struct {
	Endpoint string `yaml:"endpoint"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// SectionWebhook is sub section of config.
type SectionWebhook struct {
	URL string `yaml:"url"`
}

// BuildDefaultConf is default config setting.
func BuildDefaultConf() Config {
	var conf Config

	// Core
	conf.Core.Enabled = true
	conf.Core.Address = ""
	conf.Core.Port = 9012
	conf.Core.SSL = false
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

	// Kubernetes
	conf.Kubernetes.Namespace = "default"

	// Registry
	conf.Registry.Endpoint = ""

	// Webhook
	conf.Webhook.URL = ""

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

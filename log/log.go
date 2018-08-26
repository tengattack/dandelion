package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
	logrusagent "github.com/tengattack/logrus-agent-hook"
)

// colors
var (
	ColorGreen   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	ColorWhite   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	ColorYellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	ColorRed     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	ColorBlue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	ColorMagenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	ColorCyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	ColorReset   = string([]byte{27, 91, 48, 109})
)

// LogReq is http request log
type LogReq struct {
	URI         string `json:"uri"`
	Method      string `json:"method"`
	IP          string `json:"ip"`
	ContentType string `json:"content_type"`
	Agent       string `json:"agent"`
}

// Config is logging config.
type Config struct {
	Format      string      `yaml:"format"`
	AccessLog   string      `yaml:"access_log"`
	AccessLevel string      `yaml:"access_level"`
	ErrorLog    string      `yaml:"error_log"`
	ErrorLevel  string      `yaml:"error_level"`
	Agent       AgentConfig `yaml:"agent"`
}

// AgentConfig is sub section of LogConfig.
type AgentConfig struct {
	Enabled    bool   `yaml:"enabled"`
	DSN        string `yaml:"dsn"`
	AppID      string `yaml:"app_id"`
	Host       string `yaml:"host"`
	InstanceID string `yaml:"instance_id"`
}

var conf *Config
var (
	// IsTerm instructs current stdout whether is terminal
	IsTerm bool
	// LogAccess is log access log
	LogAccess *logrus.Logger
	// LogError is log error log
	LogError *logrus.Logger
)

func init() {
	IsTerm = isatty.IsTerminal(os.Stdout.Fd())
}

// InitLog use for initial log module
func InitLog(logConf *Config) error {
	var err error

	conf = logConf
	if conf.Agent.Enabled {
		// get default host and instance id from environment variables or hostname
		if conf.Agent.Host == "" || conf.Agent.InstanceID == "" {
			hostname, _ := os.Hostname()
			if conf.Agent.Host == "" {
				host := os.Getenv("HOST")
				if host == "" {
					host = hostname
				}
				conf.Agent.Host = host
			}
			if conf.Agent.InstanceID == "" {
				instanceID := os.Getenv("INSTANCE_ID")
				if instanceID == "" {
					instanceID = hostname
				}
				conf.Agent.InstanceID = instanceID
			}
		}
	}

	// init logger
	LogAccess = logrus.New()
	LogError = logrus.New()

	LogAccess.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006/01/02 - 15:04:05",
		FullTimestamp:   true,
	}

	LogError.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006/01/02 - 15:04:05",
		FullTimestamp:   true,
	}

	// set logger
	if err = SetLogLevel(LogAccess, conf.AccessLevel); err != nil {
		return errors.New("Set access log level error: " + err.Error())
	}

	if err = SetLogLevel(LogError, conf.ErrorLevel); err != nil {
		return errors.New("Set error log level error: " + err.Error())
	}

	if err = SetLogOut(LogAccess, conf.AccessLog); err != nil {
		return errors.New("Set access log path error: " + err.Error())
	}

	if err = SetLogOut(LogError, conf.ErrorLog); err != nil {
		return errors.New("Set error log path error: " + err.Error())
	}

	return nil
}

// SetLogOut provide log stdout and stderr output
func SetLogOut(log *logrus.Logger, outString string) error {
	switch outString {
	case "stdout":
		log.Out = os.Stdout
	case "stderr":
		log.Out = os.Stderr
	default:
		f, err := os.OpenFile(outString, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)

		if err != nil {
			return err
		}

		log.Out = f
	}

	if conf.Agent.Enabled {
		// configure log agent (logstash) hook
		u, err := url.Parse(conf.Agent.DSN)
		if err != nil {
			return err
		}
		conn, err := net.Dial(u.Scheme, u.Host)
		if err != nil {
			return err
		}
		hook := logrusagent.New(
			conn, logrusagent.DefaultFormatter(
				logrus.Fields{
					"app_id":      conf.Agent.AppID,
					"host":        conf.Agent.Host,
					"instance_id": conf.Agent.InstanceID,
				}))
		log.Hooks.Add(hook)
	}

	return nil
}

// SetLogLevel is define log level what you want
// log level: panic, fatal, error, warn, info and debug
func SetLogLevel(log *logrus.Logger, levelString string) error {
	level, err := logrus.ParseLevel(levelString)

	if err != nil {
		return err
	}

	log.Level = level

	return nil
}

// LogRequest record http request
func LogRequest(uri string, method string, ip string, contentType string, agent string) {
	var output string
	log := &LogReq{
		URI:         uri,
		Method:      method,
		IP:          ip,
		ContentType: contentType,
		Agent:       agent,
	}

	if conf.Format == "json" {
		logJSON, _ := json.Marshal(log)

		output = string(logJSON)
	} else {
		var headerColor, resetColor string

		if IsTerm {
			headerColor = ColorMagenta
			resetColor = ColorReset
		}

		// format is string
		output = fmt.Sprintf("|%s header %s| %s %s %s %s %s",
			headerColor, resetColor,
			log.Method,
			log.URI,
			log.IP,
			log.ContentType,
			log.Agent,
		)
	}

	LogAccess.Info(output)
}

// LogMiddleware provide gin router handler.
func LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		LogRequest(c.Request.URL.Path, c.Request.Method, c.ClientIP(), c.ContentType(), c.Request.Header.Get("User-Agent"))
		c.Next()
	}
}

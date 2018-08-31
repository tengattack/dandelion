package log

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tengattack/tgo/log"
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

var conf *log.Config

var (
	// LogAccess is log access log
	LogAccess *logrus.Logger
	// LogError is log error log
	LogError *logrus.Logger
)

// InitLog use for initial log module
func InitLog(logConf *log.Config) error {
	err := log.InitLog(logConf)
	if err != nil {
		return err
	}
	LogAccess = log.LogAccess
	LogError = log.LogError
	conf = log.GetLogConfig()
	return nil
}

// LogRequest record http request
func LogRequest(uri string, method string, ip string, contentType string, agent string) {
	var output string
	req := &LogReq{
		URI:         uri,
		Method:      method,
		IP:          ip,
		ContentType: contentType,
		Agent:       agent,
	}

	if conf.Format == "json" {
		logJSON, _ := json.Marshal(req)

		output = string(logJSON)
	} else {
		var headerColor, resetColor string

		if log.IsTerm {
			headerColor = ColorMagenta
			resetColor = ColorReset
		}

		// format is string
		output = fmt.Sprintf("|%s header %s| %s %s %s %s %s",
			headerColor, resetColor,
			req.Method,
			req.URI,
			req.IP,
			req.ContentType,
			req.Agent,
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

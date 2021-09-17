package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/log"
)

const (
	// ParamsError the http bad request for error params
	ParamsError = "Params error"
)

func abortWithError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, gin.H{
		"code": code,
		"info": message,
	})
}

func succeed(c *gin.Context, message interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"info": message,
	})
}

// HandleMessage handle dandelion messages
func HandleMessage(m *app.NotifyMessage) {
	switch m.Event {
	case "check":
		fallthrough
	case "publish":
		fallthrough
	case "rollback":
		for _, config := range Conf.Configs {
			// TODO: check matching of host and instance_id
			if config.AppID == m.AppID {
				err := CheckAppConfig(&config)
				if err != nil {
					log.LogError.WithField("app_id", m.AppID).Errorf("handle message error: %v", err)
					// PASS
				}
			}
		}
	}
}

func appHealthHandler(c *gin.Context) {
	succeed(c, "success")
}

func appCheckHandler(c *gin.Context) {
	appID := c.Param("app_id")

	if appID == "" {
		abortWithError(c, http.StatusBadRequest, ParamsError)
		return
	}

	var errs []error
	for _, config := range Conf.Configs {
		if config.AppID == appID {
			errs = append(errs, CheckAppConfig(&config))
		}
	}

	if len(errs) <= 0 {
		abortWithError(c, http.StatusNotFound, "not found specified app_id")
		return
	}

	succeed(c, gin.H{
		"app_id": appID,
		"errors": errs,
	})
}

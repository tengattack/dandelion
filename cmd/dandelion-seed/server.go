package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tengattack/dandelion/log"
)

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"info": "Welcome to dandelion.",
	})
}

func routerEngine() *gin.Engine {
	// set server mode
	gin.SetMode(Conf.API.Mode)

	r := gin.New()

	// Global middleware
	//r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(log.LogMiddleware())

	r.POST("/check/:app_id", appCheckHandler)

	return r
}

// RunHTTPServer provide run http or https protocol.
func RunHTTPServer() (err error) {
	if !Conf.API.Enabled {
		log.LogAccess.Debug("httpd server is disabled.")
		return nil
	}

	log.LogAccess.Debugf("HTTPD server is running on %s:%d.", Conf.API.Address, Conf.API.Port)

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", Conf.API.Address, Conf.API.Port), routerEngine())

	return err
}

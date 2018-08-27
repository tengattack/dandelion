package main

import (
	"fmt"
	"net/http"

	"github.com/tengattack/dandelion/log"

	"github.com/gin-gonic/gin"
)

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"info": "Welcome to dandelion.",
	})
}

func routerEngine() *gin.Engine {
	// set server mode
	gin.SetMode(Conf.Core.Mode)

	r := gin.New()

	// Global middleware
	//r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(log.LogMiddleware())

	// expvar
	//r.GET("/debug/vars", expvar.Handler())

	r.POST("/sync", appSyncHandler)
	r.POST("/sync/:app_id", appSyncHandler)
	r.GET("/list", appListHandler)
	r.GET("/list/:app_id/configs", appListConfigsHandler)
	r.GET("/list/:app_id/commits", appListCommitsHandler)
	r.GET("/list/:app_id/tree/:commit_id", appListFilesHandler)
	r.GET("/list/:app_id/tree/:commit_id/*path", appGetFileHandler)
	r.POST("/publish/:app_id", appPublishConfigHandler)
	r.POST("/rollback/:app_id", appRollbackConfigHandler)
	r.GET("/match/:app_id", appMatchConfigHandler)
	r.GET("/connect/push", wsPushHandler)
	r.GET("/", rootHandler)

	return r
}

// RunHTTPServer provide run http or https protocol.
func RunHTTPServer() (err error) {
	if !Conf.Core.Enabled {
		log.LogAccess.Debug("httpd server is disabled.")
		return nil
	}

	err = InitHandlers()
	if err != nil {
		return err
	}
	log.LogAccess.Debugf("HTTPD server is running on %s:%d.", Conf.Core.Address, Conf.Core.Port)
	/* if PushConf.Core.AutoTLS.Enabled {
		s := autoTLSServer()
		err = s.ListenAndServeTLS("", "")
	} else if PushConf.Core.SSL && PushConf.Core.CertPath != "" && PushConf.Core.KeyPath != "" {
		err = http.ListenAndServeTLS(PushConf.Core.Address+":"+PushConf.Core.Port, PushConf.Core.CertPath, PushConf.Core.KeyPath, routerEngine())
	} else { */
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", Conf.Core.Address, Conf.Core.Port), routerEngine())
	// }

	return err
}

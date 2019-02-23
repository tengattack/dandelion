package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/tengattack/dandelion/log"

	"github.com/gin-gonic/gin"
)

var (
	baseURLRegexp *regexp.Regexp
)

func init() {
	baseURLRegexp = regexp.MustCompile(`(<base href=|\.PUBLIC_URL = )".*"`)
}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"info": "Welcome to dandelion.",
	})
}

func indexHandler(c *gin.Context) {
	path := "index.html"
	res, err := Asset(path)
	if err != nil {
		c.Next()
		return
	}
	contentType := "text/html"
	if Conf.Core.PublicURL != "" {
		// public url
		res = baseURLRegexp.ReplaceAll(res, []byte(`$1`+strconv.Quote(Conf.Core.PublicURL)))
	} else if n := strings.Count(c.Request.URL.Path, "/"); n > 1 {
		// runtime update base path
		res = baseURLRegexp.ReplaceAll(res, []byte(`$1"`+strings.Repeat("../", n-1)+`"`))
	}
	c.Data(http.StatusOK, contentType, res)
}

func routerEngine() *gin.Engine {
	// set server mode
	gin.SetMode(Conf.Core.Mode)

	r := gin.New()

	// Global middleware
	//r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(log.LogMiddleware())

	// web public
	r.GET("/assets/*asset", servePublic)
	r.GET("/favicon.ico", servePublic)
	r.GET("/manifest.json", servePublic)
	r.GET("/index.html", servePublic)
	r.GET("/a/*app_id", indexHandler)
	r.GET("/", indexHandler)

	// expvar
	//r.GET("/debug/vars", expvar.Handler())

	// websocket
	r.GET("/connect/push", wsPushHandler)

	g := r.Group("/api/v1")
	g.POST("/sync", appSyncHandler)
	g.POST("/sync/:app_id", appSyncHandler)
	g.GET("/list", appListHandler)
	g.GET("/list/:app_id/configs", appListConfigsHandler)
	g.GET("/list/:app_id/commits", appListCommitsHandler)
	g.GET("/list/:app_id/instances", appListInstancesHandler)
	g.GET("/list/:app_id/tree/:commit_id", appListFilesHandler)
	g.GET("/list/:app_id/tree/:commit_id/*path", appGetFileHandler)
	g.GET("/archive/:app_id/:commit_id", appGetArchiveHandler) // ends with `.zip`
	g.POST("/publish/:app_id", appPublishConfigHandler)
	g.POST("/rollback/:app_id", appRollbackConfigHandler)
	g.GET("/match/:app_id", appMatchConfigHandler)
	g.POST("/check/:app_id", appCheckHandler)

	return r
}

//go:generate go-bindata -prefix "../../web/public" -pkg main -o bindata.go ../../web/public/...
func servePublic(c *gin.Context) {
	path := c.Request.URL.Path

	path = strings.Replace(path, "/", "", 1)
	split := strings.Split(path, ".")
	suffix := split[len(split)-1]

	res, err := Asset(path)
	if err != nil {
		c.Next()
		return
	}

	contentType := "text/plain"
	switch suffix {
	case "png":
		contentType = "image/png"
	case "jpg", "jpeg":
		contentType = "image/jpeg"
	case "gif":
		contentType = "image/gif"
	case "svg":
		contentType = "image/svg+xml"
	case "ico":
		contentType = "image/x-icon"
	case "js":
		contentType = "application/javascript"
	case "json":
		contentType = "application/json"
	case "css":
		contentType = "text/css"
	case "woff":
		contentType = "application/x-font-woff"
	case "ttf":
		contentType = "application/x-font-ttf"
	case "otf":
		contentType = "application/x-font-otf"
	case "html":
		contentType = "text/html"
	}

	c.Data(http.StatusOK, contentType, res)
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

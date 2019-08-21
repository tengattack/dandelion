package main

import (
	"fmt"
	"net/http"

	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/cmd/dandelion/controllers"
	"github.com/tengattack/dandelion/log"
)

// RunHTTPServer provide run http or https protocol.
func RunHTTPServer() error {
	if !config.Conf.Core.Enabled {
		log.LogAccess.Debug("httpd server is disabled.")
		return nil
	}

	router, err := controllers.InitHandlers()
	if err != nil {
		return err
	}
	log.LogAccess.Debugf("HTTPD server is running on %s:%d.", config.Conf.Core.Address, config.Conf.Core.Port)
	/* if PushConf.Core.AutoTLS.Enabled {
		s := autoTLSServer()
		err = s.ListenAndServeTLS("", "")
	} else if PushConf.Core.SSL && PushConf.Core.CertPath != "" && PushConf.Core.KeyPath != "" {
		err = http.ListenAndServeTLS(PushConf.Core.Address+":"+PushConf.Core.Port, PushConf.Core.CertPath, PushConf.Core.KeyPath, routerEngine())
	} else { */
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", config.Conf.Core.Address, config.Conf.Core.Port), router)
	// }

	return err
}

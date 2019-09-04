package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/cmd/dandelion/controllers"
	"github.com/tengattack/dandelion/log"
	"golang.org/x/sync/errgroup"
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

	eg, ctx := errgroup.WithContext(context.Background())

	if config.Conf.Core.SSL && config.Conf.Core.CertPath != "" && config.Conf.Core.CertKeyPath != "" {
		// SSL enabled
		eg.Go(func() error {
			err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", config.Conf.Core.Address, config.Conf.Core.SSLPort),
				config.Conf.Core.CertPath,
				config.Conf.Core.CertKeyPath,
				router)
			log.LogAccess.Errorf("HTTPD server (SSL) listen error: %v", err)
			return err
		})
		log.LogAccess.Debugf("HTTPD server (SSL) is running on %s:%d.", config.Conf.Core.Address, config.Conf.Core.SSLPort)
	}

	eg.Go(func() error {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", config.Conf.Core.Address, config.Conf.Core.Port), router)
		log.LogAccess.Errorf("HTTPD server listen error: %v", err)
		return err
	})
	log.LogAccess.Debugf("HTTPD server is running on %s:%d.", config.Conf.Core.Address, config.Conf.Core.Port)

	<-ctx.Done()
	return ctx.Err()
}

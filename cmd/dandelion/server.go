package main

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/cmd/dandelion/controllers"
	"github.com/tengattack/tgo/logger"
)

// RunHTTPServer provide run http or https protocol.
func RunHTTPServer() error {
	if !config.Conf.Core.Enabled {
		logger.Debug("httpd server is disabled.")
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
			logger.Errorf("HTTPD server (SSL) listen error: %v", err)
			return err
		})
		logger.Debugf("HTTPD server (SSL) is running on %s:%d.", config.Conf.Core.Address, config.Conf.Core.SSLPort)
	}

	eg.Go(func() error {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", config.Conf.Core.Address, config.Conf.Core.Port), router)
		logger.Errorf("HTTPD server listen error: %v", err)
		return err
	})
	logger.Debugf("HTTPD server is running on %s:%d.", config.Conf.Core.Address, config.Conf.Core.Port)

	<-ctx.Done()
	return ctx.Err()
}

package server

import (
	"context"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/api"
	"github.com/LambdaTest/neuron/pkg/lumber"

	"github.com/LambdaTest/neuron/config"
	"github.com/gin-gonic/gin"
)

// ListenAndServe initializes a server to respond to HTTP network requests.
func ListenAndServe(ctx context.Context, router *api.Router, cfg *config.Config, logger lumber.Logger) error {
	// set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	logger.Infof("Setting up http handler")

	errChan := make(chan error)

	// HTTP server instance
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router.Handler(),
	}

	// channel to signal server process exit
	done := make(chan struct{})
	go func() {
		logger.Infof("Starting server on port %s", cfg.Port)
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("listen: %#v", err)
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Infof("Caller has requested graceful shutdown. shutting down the server")
		if err := srv.Shutdown(context.Background()); err != nil && err != context.Canceled {
			logger.Errorf("Server Shutdown: error %v", err)
		}
		return nil
	case err := <-errChan:
		return err
	case <-done:
		return nil
	}
}

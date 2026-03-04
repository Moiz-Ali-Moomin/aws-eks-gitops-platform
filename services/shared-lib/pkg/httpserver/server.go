package httpserver

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
}

func New(handler http.Handler, serviceName string, port string) *Server {
	mux := http.NewServeMux()

	// Add observability endpoints
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap user handler with middleware
	instrumentedHandler := middleware.Chain(
		handler,
		middleware.Tracing(serviceName),
		middleware.Metrics,
	)

	// Mount everything
	// Note: In a real router like Chi/Gin, this mounting logic would be cleaner.
	// We assume 'handler' handles its own routing, so we wrap it.
	// But we need /metrics to bypass auth/etc usually.
	// For this simple mux setup:

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			promhttp.Handler().ServeHTTP(w, r)
			return
		}
		instrumentedHandler.ServeHTTP(w, r)
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + port,
			Handler: finalHandler,
		},
	}
}

func (s *Server) Run() error {
	// Server run context
	srvErr := make(chan error, 1)
	go func() {
		logger.Log.Info("Starting HTTP server", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srvErr <- err
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-srvErr:
		return err
	case <-quit:
		logger.Log.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			logger.Log.Error("Server forced to shutdown", zap.Error(err))
			return err
		}

		logger.Log.Info("Server exited properly")
		return nil
	}
}

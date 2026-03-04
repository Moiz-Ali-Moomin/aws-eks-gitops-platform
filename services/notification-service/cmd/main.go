package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

var consumer *event.Consumer

func main() {
	logger.Init("notification-service")

	tp, err := tracing.InitTracer("notification-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"kafka:9092"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create consumer instance
	consumer = event.NewConsumer(
		brokers,
		"notification-group",
		[]string{"conversion.completed"},
		HandleNotification,
	)

	// Start Kafka consumer in the background (non-blocking, self-healing)
	logger.Log.Info("Starting notification-service Kafka consumer (async)...")
	consumer.StartAsync(ctx)

	// HTTP server for health checks (runs regardless of Kafka state)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
		<-sigterm
		logger.Log.Info("Shutting down notification-service...")
		cancel()
		_ = server.Close()
	}()

	logger.Log.Info("notification-service HTTP server listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Log.Fatal("HTTP server failed", zap.Error(err))
	}
}

// healthHandler returns 200 always (liveness probe)
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// readyHandler returns 200 only if Kafka consumer is connected (readiness probe)
func readyHandler(w http.ResponseWriter, r *http.Request) {
	if consumer != nil && consumer.IsReady() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("kafka not ready"))
	}
}

func HandleNotification(ctx context.Context, key string, data []byte) error {
	var evt event.Event
	if err := json.Unmarshal(data, &evt); err != nil {
		return nil
	}
	// Mock Email Sending
	logger.Log.Info("Sending notification email",
		zap.String("event", "conversion"),
		zap.String("session_id", evt.SessionID),
	)
	return nil
}

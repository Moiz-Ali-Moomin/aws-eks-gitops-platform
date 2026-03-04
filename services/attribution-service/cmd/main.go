package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/attribution-service/internal/handler"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

var consumer *event.Consumer

func main() {
	logger.Init("attribution-service")

	tp, err := tracing.InitTracer("attribution-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(nil) }()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"kafka-headless.kafka.svc:9092"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Producer for "campaign.attributed"
	producer, err := event.NewProducer(brokers, "campaign.attributed")
	if err != nil {
		logger.Log.Error("Failed to create producer", zap.Error(err))
	}
	defer producer.Close()

	h := handler.New(producer)

	// Create consumer instance
	consumer = event.NewConsumer(
		brokers,
		"attribution-group",
		[]string{"conversion.completed"},
		h.HandleConversionEvent,
	)

	// Start Kafka consumer in the background (non-blocking, self-healing)
	logger.Log.Info("Starting attribution-service Kafka consumer (async)...")
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
		logger.Log.Info("Shutting down attribution-service...")
		cancel()
		_ = server.Close()
	}()

	logger.Log.Info("attribution-service HTTP server listening on :8080")
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

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/conversion-webhook/internal/handler"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	logger.Init("conversion-webhook")

	tp, err := tracing.InitTracer("conversion-webhook")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Kafka Producer Setup
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"kafka:9092"}
	}

	producer, err := event.NewProducer(brokers, "conversion.completed")
	if err != nil {
		logger.Log.Error("Failed to create producer", zap.Error(err))
	}
	defer producer.Close()

	secretKey := os.Getenv("CLICKBANK_SECRET_KEY")
	h := handler.New(producer, secretKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/webhook/clickbank", h.HandleClickBankParam)

	logger.Log.Info("Starting conversion-webhook on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

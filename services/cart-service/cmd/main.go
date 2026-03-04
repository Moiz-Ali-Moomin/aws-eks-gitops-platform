package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/cart-service/internal/handler"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/httpserver"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	logger.Init("cart-service")

	tp, err := tracing.InitTracer("cart-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Init Event Producer
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		// Default for local
		brokers = []string{"kafka:9092"}
	}

	producer, err := event.NewProducer(brokers, "cart-service")
	if err != nil {
		logger.Log.Fatal("Failed to create kafka producer", zap.Error(err))
	}
	defer producer.Close()

	h := handler.New(producer)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/cart", h.HandleCart)

	server := httpserver.New(mux, "cart-service", "8080")
	if err := server.Run(); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

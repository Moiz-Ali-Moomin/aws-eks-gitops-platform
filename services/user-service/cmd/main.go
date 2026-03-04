package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/httpserver"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/user-service/internal/handler"
	"go.uber.org/zap"
)

func main() {
	logger.Init("user-service")

	tp, err := tracing.InitTracer("user-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.TODO()) }()

	h := handler.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/users", h.HandleUsers)     // POST create
	mux.HandleFunc("/users/", h.HandleUserByID) // GET profile

	server := httpserver.New(mux, "user-service", "8080")
	if err := server.Run(); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

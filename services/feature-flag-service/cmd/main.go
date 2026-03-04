package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	logger.Init("feature-flag-service")

	tp, err := tracing.InitTracer("feature-flag-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(nil) }()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/flags", GetFlags)

	logger.Log.Info("Starting feature-flag-service on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func GetFlags(w http.ResponseWriter, r *http.Request) {
	// Mock Flags
	flags := map[string]bool{
		"new_checkout_flow": true,
		"dark_mode":         false,
		"beta_features":     true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flags)
}

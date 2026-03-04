package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/admin-backoffice-service/internal/metabase"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	logger.Init("admin-backoffice-service")

	tp, err := tracing.InitTracer("admin-backoffice-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	// In reality: CRUD endpoints for /campaigns, /offers, /users
	mux.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": "System Healthy", "revenue": "high"})
	})

	// Metabase Analytics JSON API
	mux.HandleFunc("/api/analytics/token", metabase.GetAnalyticsTokenHandler)

	logger.Log.Info("Starting admin-backoffice-service on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

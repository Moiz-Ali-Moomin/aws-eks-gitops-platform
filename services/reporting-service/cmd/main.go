package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

type CampaignStats struct {
	CampaignID  string  `json:"campaign_id"`
	Clicks      int     `json:"clicks"`
	Conversions int     `json:"conversions"`
	Revenue     float64 `json:"revenue"`
	ROI         float64 `json:"roi"`
}

func main() {
	logger.Init("reporting-service")

	tp, err := tracing.InitTracer("reporting-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(nil) }()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/reports/campaigns", GetCampaignStats)

	logger.Log.Info("Starting reporting-service on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func GetCampaignStats(w http.ResponseWriter, r *http.Request) {
	// In production: Query 'analytics.funnel_daily' in Postgres
	stats := []CampaignStats{
		{CampaignID: "camp_001", Clicks: 15302, Conversions: 420, Revenue: 14700.00, ROI: 2.5},
		{CampaignID: "camp_002", Clicks: 5040, Conversions: 89, Revenue: 2100.00, ROI: 1.1},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

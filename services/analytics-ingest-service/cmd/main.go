package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IngestRequest struct {
	EventName string                 `json:"event_name"`
	Payload   map[string]interface{} `json:"payload"`
}

func main() {
	logger.Init("analytics-ingest-service")

	tp, err := tracing.InitTracer("analytics-ingest-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"kafka:9092"}
	}

	// We create producers for allowed topics on demand or pre-init map
	// For simplicity, let's assume one producer for 'raw.analytics' or dynamic
	// Master Prompt says "page.viewed", etc. We'll map dynamic here.

	// Pre-create producers
	producers := make(map[string]*event.Producer)
	topics := []string{"page.viewed", "cta.clicked", "user.signup", "checkout.completed"}

	for _, t := range topics {
		p, err := event.NewProducer(brokers, t)
		if err != nil {
			logger.Log.Error("Failed to create producer", zap.String("topic", t), zap.Error(err))
		} else {
			producers[t] = p
			defer p.Close()
		}
	}

	// WaitGroup to track pending emissions
	var wg sync.WaitGroup

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		var req IngestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		p, ok := producers[req.EventName]
		if !ok {
			http.Error(w, "Unknown Event Type", http.StatusBadRequest)
			return
		}

		// Wrap in standard event
		evt := event.Event{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			Service:   "analytics-ingest",
			Metadata:  req.Payload,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Use background context with timeout for detached execution
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := p.Emit(ctx, req.EventName, uuid.New().String(), evt); err != nil {
				logger.Log.Error("Ingest failed", zap.Error(err))
			}
		}()

		w.WriteHeader(http.StatusAccepted)
	})

	// Graceful Shutdown Setup
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	go func() {
		logger.Log.Info("Starting analytics-ingest-service on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("Waiting for pending events...")
	wg.Wait()
	logger.Log.Info("Server exiting")
}

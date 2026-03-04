package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Items     []Item    `json:"items"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

var orders = make(map[string]Order)

func main() {
	logger.Init("order-service")
	logger.Log.Info("Starting order-service...")

	tp, err := tracing.InitTracer("order-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"kafka-cluster-kafka-bootstrap.kafka:9092"}
	}

	producer, err := event.NewProducer(brokers, "order-service")
	if err != nil {
		logger.Log.Error("Failed to create kafka producer", zap.Error(err))
	} else {
		defer producer.Close()
	}

	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create Order
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var order Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		order.ID = uuid.New().String()
		order.Status = "PENDING"
		order.CreatedAt = time.Now()
		orders[order.ID] = order

		// Emit event if producer is available
		if producer != nil {
			evt := event.Event{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
				Service:   "order-service",
				Metadata: map[string]interface{}{
					"order_id": order.ID,
					"user_id":  order.UserID,
					"total":    order.Total,
				},
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := producer.Emit(ctx, "checkout.completed", order.ID, evt); err != nil {
					logger.Log.Error("Failed to emit order event", zap.Error(err))
				}
			}()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(order)
	})

	// List Orders
	mux.HandleFunc("GET /orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		orderList := make([]Order, 0, len(orders))
		for _, o := range orders {
			orderList = append(orderList, o)
		}
		json.NewEncoder(w).Encode(orderList)
	})

	// Get Order
	mux.HandleFunc("GET /order/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/order/"):]
		if order, ok := orders[id]; ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(order)
			return
		}
		http.Error(w, "Order not found", http.StatusNotFound)
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Listen failure", zap.Error(err))
		}
	}()
	logger.Log.Info("Server listening on :8080")

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Log.Info("Server exiting")
}

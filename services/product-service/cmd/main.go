package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	ImageURL    string  `json:"image_url"`
}

var products = []Product{
	{ID: "1", Name: "Vintage Camera", Description: "Retro style 35mm film camera", Price: 199.99, Currency: "USD", ImageURL: "/images/camera.jpg"},
	{ID: "2", Name: "Stylish Sunglasses", Description: "UV protection aviator sunglasses", Price: 59.99, Currency: "USD", ImageURL: "/images/glasses.jpg"},
	{ID: "3", Name: "Leather Hiking Boots", Description: "Durable waterproof boots", Price: 129.50, Currency: "USD", ImageURL: "/images/boots.jpg"},
	{ID: "4", Name: "Mechanical Watch", Description: "Automatic movement luxury watch", Price: 450.00, Currency: "USD", ImageURL: "/images/watch.jpg"},
}

func main() {
	logger.Init("product-service")
	logger.Log.Info("Starting product-service...")

	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// List Products
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	})

	// Get Product Details
	mux.HandleFunc("/product/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/product/"):]
		for _, p := range products {
			if p.ID == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(p)
				return
			}
		}
		http.Error(w, "Product not found", http.StatusNotFound)
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

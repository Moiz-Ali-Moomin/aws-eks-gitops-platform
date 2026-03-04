package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/storefront-service/internal/handler"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/storefront-service/internal/service"
)

func main() {
	// Initialize Logger
	logger.Init("storefront-service")

	config := handler.Config{
		Port:       getEnv("PORT", "3000"),
		GatewayURL: getEnv("GATEWAY_URL", "http://api-gateway:8080"),
	}

	// Initialize Kafka Producer
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka-bootstrap.kafka.svc.cluster.local:9092"), ",")
	producer, err := event.NewProducer(brokers, "page.viewed")
	if err != nil {
		log.Printf("Warning: Failed to initialize Kafka producer: %v", err)
		// Proceed without Kafka (producer will be nil)
	} else {
		defer producer.Close()
		log.Printf("Connected to Kafka brokers at %s", brokers)
	}

	// Initialize Services & Handler
	analyticsService := service.NewAnalyticsService(producer)
	h := handler.NewHandler(analyticsService, config)

	// Routes
	http.HandleFunc("/", h.Home) // Use new handler
	http.HandleFunc("/login", handleLogin(config))
	http.HandleFunc("/signup", handleSignup(config))
	http.HandleFunc("/logout", handleLogout())

	// Protected Routes
	http.HandleFunc("/cart", handleCart(config))
	http.HandleFunc("/cart/add", handleAddToCart(config))
	http.HandleFunc("/cart/remove", handleRemoveFromCart(config))
	http.HandleFunc("/checkout", handleCheckout(config))
	http.HandleFunc("/payment/process", handlePaymentProcess(config))

	// Graceful Shutdown
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: nil, // DefaultServeMux
	}

	go func() {
		log.Printf("Storefront-Service running on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Context for shutdown period
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Trigger Graceful Shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Flush Analytics Events
	analyticsService.Close()

	log.Println("Server exiting")
}

func handleLogin(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != "POST" {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Read JSON body instead of FormValue
		var reqBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		username := reqBody["username"]
		password := reqBody["password"]

		authBody, _ := json.Marshal(map[string]string{
			"username": username,
			"password": password,
		})

		resp, err := http.Post(cfg.GatewayURL+"/api/v1/login", "application/json", bytes.NewBuffer(authBody))
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, `{"error": "Invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		defer resp.Body.Close()

		var authResp handler.AuthResponse
		json.NewDecoder(resp.Body).Decode(&authResp)

		// Keep Cookies
		http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: authResp.Token, Path: "/", HttpOnly: true})
		http.SetCookie(w, &http.Cookie{Name: "auth_user", Value: authResp.Username, Path: "/"})

		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "user": authResp.Username})
	}
}

func handleSignup(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != "POST" {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var reqBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		username := reqBody["username"]
		password := reqBody["password"]

		authBody, _ := json.Marshal(map[string]string{
			"username": username,
			"password": password,
		})

		resp, err := http.Post(cfg.GatewayURL+"/api/v1/signup", "application/json", bytes.NewBuffer(authBody))
		if err != nil || resp.StatusCode != http.StatusCreated {
			http.Error(w, `{"error": "Failed to create account (User may exist)"}`, http.StatusConflict)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Account created successfully. Please login."})
	}
}

func handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: "auth_user", Value: "", Path: "/", MaxAge: -1})

		json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
	}
}

func handleCart(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		userID := handler.GetAuthenticatedUser(r)
		if userID == "" {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// 1. Fetch Cart
		req, _ := http.NewRequest("GET", cfg.GatewayURL+"/api/v1/cart?user_id="+userID, nil)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		var cart handler.Cart
		if err == nil && resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(&cart)
			resp.Body.Close()
		}

		// 2. Fetch Offers
		respOffers, err := http.Get(cfg.GatewayURL + "/api/v1/offers")
		offersMap := make(map[string]handler.Offer)
		if err == nil && respOffers.StatusCode == http.StatusOK {
			var offers []handler.Offer
			json.NewDecoder(respOffers.Body).Decode(&offers)
			respOffers.Body.Close()
			for _, o := range offers {
				offersMap[o.ID] = o
			}
		}

		// 3. Enrich Cart
		total := 0.0
		for i, item := range cart.Items {
			if offer, ok := offersMap[item.OfferID]; ok {
				cart.Items[i].Title = offer.Title
				cart.Items[i].Price = offer.Payout
				cart.Items[i].Quantity = item.Quantity
				total += offer.Payout * float64(item.Quantity)
			} else {
				cart.Items[i].Title = "Unknown Product"
			}
		}

		data := map[string]interface{}{
			"cart":  cart,
			"total": total,
			"user":  userID,
		}

		json.NewEncoder(w).Encode(data)
	}
}

func handleCheckout(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		userID := handler.GetAuthenticatedUser(r)
		if userID == "" {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		req, _ := http.NewRequest("GET", cfg.GatewayURL+"/api/v1/cart?user_id="+userID, nil)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, _ := client.Do(req)

		total := 0.0
		if resp != nil && resp.StatusCode == http.StatusOK {
			var cart handler.Cart
			json.NewDecoder(resp.Body).Decode(&cart)
			resp.Body.Close()

			respOffers, _ := http.Get(cfg.GatewayURL + "/api/v1/offers")
			if respOffers != nil && respOffers.StatusCode == http.StatusOK {
				var offers []handler.Offer
				json.NewDecoder(respOffers.Body).Decode(&offers)
				respOffers.Body.Close()
				offersMap := make(map[string]handler.Offer)
				for _, o := range offers {
					offersMap[o.ID] = o
				}
				for _, item := range cart.Items {
					if off, ok := offersMap[item.OfferID]; ok {
						total += off.Payout * float64(item.Quantity)
					}
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"total": total,
			"user":  userID,
		})
	}
}

func handlePaymentProcess(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		userID := handler.GetAuthenticatedUser(r)
		if userID == "" {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if r.Method != "POST" {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Call backend to clear cart
		req, _ := http.NewRequest("DELETE", cfg.GatewayURL+"/api/v1/cart?user_id="+userID, nil)
		client := &http.Client{Timeout: 5 * time.Second}
		client.Do(req)

		rand.Seed(time.Now().UnixNano())
		days := rand.Intn(5) + 3

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"deliveryDays": days,
			"user":         userID,
		})
	}
}

func handleAddToCart(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		userID := handler.GetAuthenticatedUser(r)
		if userID == "" {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if r.Method != "POST" {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var reqBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		offerID := reqBody["offer_id"]

		apiURL := fmt.Sprintf("%s/api/v1/cart?user_id=%s", cfg.GatewayURL, userID)

		item := map[string]interface{}{
			"offer_id": offerID,
			"quantity": 1,
		}
		jsonBody, _ := json.Marshal(item)

		resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil || resp.StatusCode != http.StatusOK {
			log.Printf("Failed to add to cart: %v", err)
			http.Error(w, `{"error": "Failed to add to cart"}`, http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		json.NewEncoder(w).Encode(map[string]string{"message": "Item added to cart"})
	}
}

func handleRemoveFromCart(cfg handler.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Implementation for removing would hit Gateway DELETE similar to clear
		json.NewEncoder(w).Encode(map[string]string{"message": "Not implemented in detailed BFF yet"})
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

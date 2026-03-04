package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	logger.Init("api-gateway")

	tp, err := tracing.InitTracer("api-gateway")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.TODO()) }()

	// Simple Proxy Logic (In real world, use Istio VirtualServices heavily)
	// This gateway might serve as an aggregation layer for mobile apps

	mux := http.NewServeMux()
	// Health endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Root path handler - API Gateway status
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// Let other handlers handle non-root paths
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"api-gateway","status":"healthy","version":"1.0.0"}`))
	})

	// Offer Service
	offerURLStr := os.Getenv("OFFER_SERVICE_URL")
	if offerURLStr == "" {
		offerURLStr = "http://offer-service.ecommerce.svc.cluster.local:8080"
	}
	offerURL, _ := url.Parse(offerURLStr)
	offerProxy := httputil.NewSingleHostReverseProxy(offerURL)
	mux.Handle("/api/v1/offers", http.StripPrefix("/api/v1", offerProxy))

	// Auth Service
	authURLStr := os.Getenv("AUTH_SERVICE_URL")
	if authURLStr == "" {
		authURLStr = "http://auth-service.ecommerce.svc.cluster.local:8080"
	}
	authURL, _ := url.Parse(authURLStr)
	authProxy := httputil.NewSingleHostReverseProxy(authURL)
	mux.Handle("/api/v1/auth", http.StripPrefix("/api/v1", authProxy)) // check if auth service expects /auth prefix or not. usually it's just /login. if gateway strips /api/v1, request becomes /auth/login? Let's assume auth service has /login at root. if so, we should handle /api/v1/auth/login -> /login. wait, main.go of auth-service has mux.HandleFunc("/login", Login). So if I request /api/v1/auth/login, I want it to hit /login. So StripPrefix("/api/v1/auth") is better. But standard pattern is often /api/v1/resource.
	// Let's stick to stripping /api/v1 for validation consistency.
	// Update: Auth service has /login. If I strip /api/v1, request to /api/v1/login -> /login. Correct.
	mux.Handle("/api/v1/login", http.StripPrefix("/api/v1", authProxy))
	mux.Handle("/api/v1/signup", http.StripPrefix("/api/v1", authProxy))
	mux.Handle("/api/v1/validate", http.StripPrefix("/api/v1", authProxy))

	// Cart Service
	cartURLStr := os.Getenv("CART_SERVICE_URL")
	if cartURLStr == "" {
		cartURLStr = "http://cart-service.ecommerce.svc.cluster.local:8080"
	}
	cartURL, _ := url.Parse(cartURLStr)
	cartProxy := httputil.NewSingleHostReverseProxy(cartURL)
	// Cart service: /cart. Request: /api/v1/cart -> /cart.
	mux.Handle("/api/v1/cart", http.StripPrefix("/api/v1", cartProxy))

	// User Service
	userURLStr := os.Getenv("USER_SERVICE_URL")
	if userURLStr == "" {
		userURLStr = "http://user-service.ecommerce.svc.cluster.local:8080"
	}
	userURL, _ := url.Parse(userURLStr)
	userProxy := httputil.NewSingleHostReverseProxy(userURL)
	// User service: /users. Request: /api/v1/users -> /users.
	mux.Handle("/api/v1/users", http.StripPrefix("/api/v1", userProxy))

	// Storefront Service (Primary UI)
	storefrontURLStr := os.Getenv("STOREFRONT_SERVICE_URL")
	if storefrontURLStr == "" {
		storefrontURLStr = "http://storefront-service.ecommerce.svc.cluster.local:3000"
	}
	storefrontURL, _ := url.Parse(storefrontURLStr)
	storefrontProxy := httputil.NewSingleHostReverseProxy(storefrontURL)

	// Storefront handles the e-commerce UI flows
	mux.Handle("/login", storefrontProxy)
	mux.Handle("/signup", storefrontProxy)
	mux.Handle("/logout", storefrontProxy)
	mux.Handle("/cart", storefrontProxy)
	mux.Handle("/cart/", storefrontProxy) // For /cart/add etc.
	mux.Handle("/checkout", storefrontProxy)
	mux.Handle("/payment/", storefrontProxy)
	mux.Handle("/static/", storefrontProxy) // Static assets

	// Landing Service (Campaign Landing Pages)
	landingURLStr := os.Getenv("LANDING_SERVICE_URL")
	if landingURLStr == "" {
		landingURLStr = "http://landing-service.ecommerce.svc.cluster.local:8080"
	}
	landingURL, _ := url.Parse(landingURLStr)
	landingProxy := httputil.NewSingleHostReverseProxy(landingURL)
	mux.Handle("/landing/", landingProxy) // Landing pages at /landing/*

	// Redirect Service (Click Tracking)
	redirectURLStr := os.Getenv("REDIRECT_SERVICE_URL")
	if redirectURLStr == "" {
		redirectURLStr = "http://redirect-service.ecommerce.svc.cluster.local:8080"
	}
	redirectURL, _ := url.Parse(redirectURLStr)
	redirectProxy := httputil.NewSingleHostReverseProxy(redirectURL)
	mux.Handle("/click/", redirectProxy)

	// Storefront as UI fallback for unmatched paths starting with /ui/
	mux.Handle("/ui/", http.StripPrefix("/ui", storefrontProxy))

	logger.Log.Info("Starting api-gateway on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

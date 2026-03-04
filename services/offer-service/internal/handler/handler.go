package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

type Offer struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Payout      float64 `json:"payout"`
	VendorID    string  `json:"vendor_id"`
	Category    string  `json:"category"`
}

type Handler struct {
	// In real world, inject Service/Repository here
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) GetOffers(w http.ResponseWriter, r *http.Request) {
	// Mock database content - realistic ecommerce products with categories
	offers := []Offer{
		// Electronics
		{ID: "103", Title: "Wireless Earbuds Pro", Description: "Active noise cancellation, 24h battery", Payout: 79.99, VendorID: "tech_store", Category: "Electronics"},
		{ID: "104", Title: "Smart Watch Series 5", Description: "Fitness tracking, heart rate monitor", Payout: 199.99, VendorID: "tech_store", Category: "Electronics"},
		{ID: "108", Title: "Bluetooth Speaker Mini", Description: "360 sound, waterproof IPX7", Payout: 59.99, VendorID: "tech_store", Category: "Electronics"},
		{ID: "115", Title: "Wireless Charging Pad", Description: "Fast charge 15W, Qi compatible", Payout: 29.99, VendorID: "tech_store", Category: "Electronics"},
		{ID: "117", Title: "Mechanical Keyboard RGB", Description: "Cherry MX switches, programmable", Payout: 119.00, VendorID: "tech_store", Category: "Electronics"},

		// Fitness
		{ID: "105", Title: "Running Shoes Air Max", Description: "Lightweight, breathable mesh", Payout: 129.00, VendorID: "sports_world", Category: "Fitness"},
		{ID: "106", Title: "Yoga Mat Premium", Description: "Non-slip, eco-friendly material", Payout: 45.00, VendorID: "fitness_hub", Category: "Fitness"},
		{ID: "116", Title: "Protein Powder Whey", Description: "25g protein per serving, 2kg", Payout: 54.99, VendorID: "fitness_hub", Category: "Fitness"},

		// Fashion
		{ID: "109", Title: "Leather Wallet Classic", Description: "Genuine leather, RFID blocking", Payout: 49.00, VendorID: "fashion_boutique", Category: "Fashion"},
		{ID: "110", Title: "Sunglasses Polarized", Description: "UV400 protection, unisex", Payout: 65.00, VendorID: "fashion_boutique", Category: "Fashion"},
		{ID: "111", Title: "Backpack Travel Pro", Description: "40L capacity, laptop compartment", Payout: 89.00, VendorID: "travel_gear", Category: "Fashion"},

		// Home & Living
		{ID: "107", Title: "Stainless Steel Water Bottle", Description: "24hr cold, 12hr hot, 750ml", Payout: 25.99, VendorID: "home_essentials", Category: "Home"},
		{ID: "113", Title: "Desk Lamp LED", Description: "Adjustable brightness, USB charging", Payout: 35.00, VendorID: "home_essentials", Category: "Home"},
		{ID: "114", Title: "Coffee Mug Ceramic", Description: "Handcrafted, 400ml capacity", Payout: 18.00, VendorID: "home_essentials", Category: "Home"},

		// Health & Wellness
		{ID: "101", Title: "Keto Diet Plan", Description: "High converting keto offer", Payout: 35.00, VendorID: "health_plus", Category: "Health"},
		{ID: "112", Title: "Electric Toothbrush", Description: "Sonic technology, 5 modes", Payout: 55.00, VendorID: "health_plus", Category: "Health"},

		// Learning
		{ID: "102", Title: "Crypto Masterclass", Description: "Learn crypto trading", Payout: 150.00, VendorID: "crypto_guru", Category: "Learning"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(offers)
}

func (h *Handler) GetOfferByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/offers/")
	logger.Log.Info("Fetching offer", zap.String("offer_id", id))

	// Mock single offer
	offer := Offer{
		ID:          id,
		Title:       "Dynamic Offer",
		Description: "Dynamically fetched offer",
		Payout:      50.00,
		VendorID:    "vendor_x",
		Category:    "General",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(offer)
}

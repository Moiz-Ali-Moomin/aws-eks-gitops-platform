package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CartItem struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
}

type Cart struct {
	UserID string     `json:"user_id"`
	Items  []CartItem `json:"items"`
}

type Handler struct {
	carts    map[string]*Cart
	producer *event.Producer
}

func New(p *event.Producer) *Handler {
	return &Handler{
		carts:    make(map[string]*Cart),
		producer: p,
	}
}

func (h *Handler) HandleCart(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	if h.carts[userID] == nil {
		h.carts[userID] = &Cart{
			UserID: userID,
			Items:  []CartItem{},
		}
	}

	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(h.carts[userID])
		return
	}

	if r.Method == http.MethodPost {
		var item CartItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if item exists, update qty
		found := false
		for i, existing := range h.carts[userID].Items {
			if existing.OfferID == item.OfferID {
				h.carts[userID].Items[i].Quantity += item.Quantity
				found = true
				break
			}
		}
		if !found {
			h.carts[userID].Items = append(h.carts[userID].Items, item)
		}

		logger.Log.Info("Added to cart", zap.String("user_id", userID), zap.String("offer_id", item.OfferID))

		// Emit Event
		evt := event.Event{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			Service:   "cart-service",
			SessionID: userID, // Assuming UserID acts as session for cart
			Metadata: map[string]interface{}{
				"action":   "add_item",
				"offer_id": item.OfferID,
				"quantity": item.Quantity,
			},
		}
		// Fire and forget (or log error)
		if h.producer != nil {
			if err := h.producer.Emit(r.Context(), "cart.updated", userID, evt); err != nil {
				logger.Log.Error("Failed to emit cart event", zap.Error(err))
			}
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle Delete/Clear for Checkout
	if r.Method == http.MethodDelete {
		h.carts[userID].Items = []CartItem{}
		w.WriteHeader(http.StatusOK)
		return
	}
}

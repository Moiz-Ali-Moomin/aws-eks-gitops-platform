package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	producer *event.Producer
}

func New(p *event.Producer) *Handler {
	return &Handler{producer: p}
}

func (h *Handler) ServeLandingPage(w http.ResponseWriter, r *http.Request) {
	// Simple Logic: Detect offer ID from path or query
	offerID := r.URL.Query().Get("offer")
	if offerID == "" {
		offerID = "default"
	}

	// Emit Page View Event
	sessionID := uuid.New().String() // In reality, check cookie/header first

	evt := event.Event{
		EventID:   uuid.New().String(),
		Timestamp: time.Now(),
		Service:   "landing-service",
		OfferID:   offerID,
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"path":       r.URL.Path,
			"user_agent": r.UserAgent(),
		},
	}

	if h.producer != nil {
		go func() {
			if err := h.producer.Emit(context.Background(), event.TypePageViewed, sessionID, evt); err != nil {
				logger.Log.Error("Failed to emit view", zap.Error(err))
			}
		}()
	}

	// Render HTML (Mock)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<html>
			<head><title>Super Offer %s</title></head>
			<body>
				<h1>Exclusive Deal!</h1>
				<p>Get the best product now.</p>
				<a href="/click/%s?session_id=%s">BUY NOW</a>
			</body>
		</html>
	`, offerID, offerID, sessionID)
}

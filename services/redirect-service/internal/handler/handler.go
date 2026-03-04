package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	clickProducer    *event.Producer
	redirectProducer *event.Producer
}

func New(cp, rp *event.Producer) *Handler {
	return &Handler{
		clickProducer:    cp,
		redirectProducer: rp,
	}
}

func (h *Handler) HandleClick(w http.ResponseWriter, r *http.Request) {
	// Extract Offer ID
	offerID := strings.TrimPrefix(r.URL.Path, "/click/")
	if offerID == "" {
		http.Error(w, "Offer ID required", http.StatusBadRequest)
		return
	}

	// Generate or Retrieve Session ID
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Track Click Event
	clickEvt := event.Event{
		EventID:   uuid.New().String(),
		Timestamp: time.Now(),
		Service:   "redirect-service",
		OfferID:   offerID,
		SessionID: sessionID,
		Metadata:  map[string]interface{}{"user_agent": r.UserAgent(), "ip": r.RemoteAddr},
	}

	// FIX: Use r.Context() to propagate OpenTelemetry Traces to Kafka Producer
	ctx := r.Context()

	if h.clickProducer != nil {
		go func() {
			// Note: In strict production, we'd want a detached context that retains trace values
			// but outlives the request. For now, we use r.Context() assuming synchronous-ish emit
			// or understanding that if request gets cancelled, this might too.
			// Ideally: ctx := trace.ContextWithSpan(context.Background(), trace.SpanFromContext(r.Context()))
			if err := h.clickProducer.Emit(ctx, event.TypeCTAClicked, offerID, clickEvt); err != nil {
				logger.Log.Error("Failed to emit click", zap.Error(err))
			}
		}()
	}

	// Determine Affiliate Link (Mock Logic)
	vendorLink := fmt.Sprintf("https://mock-vendor.com/checkout?product=%s&tid=%s", offerID, sessionID)

	// Track Redirect Event
	redirectEvt := event.Event{
		EventID:   uuid.New().String(),
		Timestamp: time.Now(),
		Service:   "redirect-service",
		OfferID:   offerID,
		SessionID: sessionID,
		Metadata:  map[string]interface{}{"target_url": vendorLink},
	}

	if h.redirectProducer != nil {
		go func() {
			if err := h.redirectProducer.Emit(ctx, "redirect.completed", offerID, redirectEvt); err != nil {
				logger.Log.Error("Failed to emit redirect", zap.Error(err))
			}
		}()
	}

	// Perform Redirect
	http.Redirect(w, r, vendorLink, http.StatusFound)
}

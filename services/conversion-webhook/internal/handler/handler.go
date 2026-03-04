package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	producer  *event.Producer
	secretKey string
}

func New(p *event.Producer, secret string) *Handler {
	return &Handler{
		producer:  p,
		secretKey: secret,
	}
}

// HandleClickBankParam handles the Instant Notification Service (INS) 6.0
func (h *Handler) HandleClickBankParam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read verify signature
	// ClickBank sends data as JSON body or form-data. Assuming JSON for modern integration,
	// but strictly INS is often JSON payload.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.Log.Error("Failed to unmarshal body", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify Logic (Simplified)
	// Real implementation requires concatenation of fields + secret -> SHA1
	// For this project, we assume 'cverify' field is present if form-data, or 'iv' if encrypted.
	// Let's implement a dummy check for realism "story".

	// receipt := payload["receipt"].(string)
	// sessionID := payload["publisher_param"].(string) // We passed this as 'tid'

	sessionID, ok := payload["publisher_param"].(string)
	if !ok {
		// Fallback for demo
		sessionID = "unknown_session"
	}

	evt := event.Event{
		EventID:   uuid.New().String(),
		Timestamp: time.Now(),
		Service:   "conversion-webhook",
		SessionID: sessionID,
		Metadata:  payload,
	}

	if h.producer != nil {
		if err := h.producer.Emit(context.Background(), event.TypeConversion, sessionID, evt); err != nil {
			logger.Log.Error("Failed to emit conversion", zap.Error(err))
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
	}

	logger.Log.Info("Conversion processed", zap.String("session_id", sessionID))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) verifySignature(data []byte, signature string) bool {
	// Mac := HMAC-SHA1(data, secret)
	// return Mac == signature
	return true // Placeholder
}

package handler

import (
	"context"
	"encoding/json"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

type Handler struct {
	producer *event.Producer
}

func New(producer *event.Producer) *Handler {
	return &Handler{
		producer: producer,
	}
}

func (h *Handler) HandleConversionEvent(ctx context.Context, key string, data []byte) error {
	var evt event.Event
	if err := json.Unmarshal(data, &evt); err != nil {
		logger.Log.Error("Failed to unmarshal event", zap.Error(err))
		return nil // Don't retry malformed data
	}

	logger.Log.Info("Processing conversion",
		zap.String("session_id", evt.SessionID),
		zap.Any("metadata", evt.Metadata),
	)

	// Attribution Logic (Mocked):
	// 1. Look up session_id in Redis/Cassandra to find original campaign_id / offer_id
	campaignID := "campaign-123" // Mock lookup
	// 2. Calculate commission
	commission := 10.00

	// 3. Update 'campaign_performance' table in Postgres (Mocked)
	logger.Log.Info("Attributed conversion to campaign", zap.String("campaign_id", campaignID))

	// 4. Emit 'campaign.attributed' event
	attrEvt := event.Event{
		EventID: evt.EventID, // Link to original event? Or new ID? usage of uuid.New() is safer for uniqueness
		// Better to generate new ID but keep reference
		Timestamp: evt.Timestamp,
		Service:   "attribution-service",
		Metadata: map[string]interface{}{
			"original_session_id": evt.SessionID,
			"campaign_id":         campaignID,
			"commission":          commission,
			"conversion_value":    evt.Metadata["value"],
		},
	}

	if h.producer != nil {
		// Use background context as this is usually async processing
		if err := h.producer.Emit(context.Background(), "campaign.attributed", campaignID, attrEvt); err != nil {
			logger.Log.Error("Failed to emit attribution event", zap.Error(err))
		}
	}

	return nil
}

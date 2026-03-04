package service

import (
	"context"

	"sync"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

type AnalyticsService struct {
	producer *event.Producer
	wg       sync.WaitGroup
}

func NewAnalyticsService(producer *event.Producer) *AnalyticsService {
	return &AnalyticsService{producer: producer}
}

func (s *AnalyticsService) TrackPageView(path, userAgent string) {
	if s.producer == nil {
		return
	}

	payload := map[string]interface{}{
		"event":      "page.viewed",
		"path":       path,
		"user_agent": userAgent,
		"source":     "storefront",
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use path as partitioning key for now
		if err := s.producer.Emit(ctx, "page.viewed", path, payload); err != nil {
			logger.Log.Error("Failed to publish page.viewed event",
				zap.Error(err),
				zap.String("path", path),
			)
		}
	}()
}

func (s *AnalyticsService) Close() {
	// Wait for all pending events to finish
	logger.Log.Info("Flushing analytics events...")
	s.wg.Wait()
	logger.Log.Info("Analytics events flushed.")
}

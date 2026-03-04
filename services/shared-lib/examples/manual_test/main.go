package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/google/uuid"
)

func main() {
	// Configuration
	brokers := []string{"localhost:9092"}
	topic := "checkout.events"

	// 1. Startup Validation
	checker := event.NewStartupCheck(brokers)

	// Check connectivity
	if err := checker.EnsureKafkaConnectivity(); err != nil {
		log.Printf("WARNING: Kafka not reachable (expected if running locally without Kafka): %v\n", err)
		// For verification purposes, we might continue or exit.
		// In production, we would exit.
	}

	// 2. Initialize Producer
	producer, err := event.NewProducer(brokers, "checkout-service")
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// 3. Create Event: Checkout Completed
	orderID := uuid.New().String()
	evt := event.CheckoutCompleted{
		Metadata: event.Metadata{
			EventID:       uuid.New().String(),
			EventType:     event.TypeCheckoutCompleted,
			ServiceName:   "checkout-service",
			UserID:        "user-123",
			CorrelationID: uuid.New().String(),
			Timestamp:     time.Now(),
			Version:       "1.0.0",
		},
		OrderID:     orderID,
		CartID:      "cart-555",
		TotalAmount: 99.99,
		Currency:    "USD",
	}

	// 4. Emit Event
	ctx := context.Background()
	// Use OrderID as partition key for ordering
	if err := producer.Emit(ctx, topic, orderID, evt); err != nil {
		log.Printf("Failed to emit event: %v", err)
	} else {
		fmt.Printf("Successfully emitted event: %s %s\n", evt.EventType, evt.EventID)
	}

	// 5. Create Event: CTA Clicked
	ctaEvt := event.CTAClicked{
		Metadata: event.Metadata{
			EventID:       uuid.New().String(),
			EventType:     event.TypeCTAClicked,
			ServiceName:   "storefront-service",
			UserID:        "user-123",
			CorrelationID: uuid.New().String(),
			Timestamp:     time.Now(),
			Version:       "1.0.0",
		},
		ButtonID:    "checkout-btn",
		LinkURL:     "/checkout",
		PageSection: "cart-summary",
	}

	if err := producer.Emit(ctx, "analytics.events", "user-123", ctaEvt); err != nil {
		log.Printf("Failed to emit CTA event: %v", err)
	} else {
		fmt.Printf("Successfully emitted event: %s %s\n", ctaEvt.EventType, ctaEvt.EventID)
	}
}

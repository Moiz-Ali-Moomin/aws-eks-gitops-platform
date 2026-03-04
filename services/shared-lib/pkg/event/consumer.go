package event

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

type Handler func(ctx context.Context, key string, event []byte) error

type Consumer struct {
	brokers []string
	topics  []string
	groupID string
	handler Handler

	// State
	mu      sync.RWMutex
	running bool
	ready   bool
}

func NewConsumer(brokers []string, groupID string, topics []string, handler Handler) *Consumer {
	return &Consumer{
		brokers: brokers,
		topics:  topics,
		groupID: groupID,
		handler: handler,
	}
}

// IsReady returns true if the consumer is connected to Kafka and consuming.
// Useful for Kubernetes readiness probes.
func (c *Consumer) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// StartAsync starts the consumer in a background goroutine.
// It will retry indefinitely until Kafka becomes available.
// This function returns immediately and never blocks.
// Use IsReady() to check if the consumer is connected.
func (c *Consumer) StartAsync(ctx context.Context) {
	go c.runWithRetry(ctx)
}

// Start is a blocking call that runs the consumer.
// DEPRECATED: Use StartAsync for non-blocking startup.
// This method is kept for backwards compatibility but now also retries forever.
func (c *Consumer) Start(ctx context.Context) error {
	c.runWithRetry(ctx)
	return nil // Never returns error, retries forever
}

func (c *Consumer) runWithRetry(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	// Retry connecting to Kafka indefinitely with exponential backoff
	var consumer sarama.ConsumerGroup
	var err error
	backoff := 2 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Consumer context cancelled, stopping retry loop")
			return
		default:
		}

		consumer, err = sarama.NewConsumerGroup(c.brokers, c.groupID, config)
		if err == nil {
			logger.Log.Info("Successfully connected to Kafka", zap.Strings("brokers", c.brokers))
			break
		}

		logger.Log.Warn("Failed to connect to Kafka, retrying...",
			zap.Error(err),
			zap.Duration("backoff", backoff),
		)
		time.Sleep(backoff)

		// Exponential backoff with cap
		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.ready = false
		c.running = false
		c.mu.Unlock()
		if consumer != nil {
			_ = consumer.Close()
		}
	}()

	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle graceful shutdown
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigterm:
			logger.Log.Info("Terminating consumer via signal")
			cancel()
		case <-ctx.Done():
		}
	}()

	handler := &groupHandler{
		handler: c.handler,
	}

	// Consume loop with auto-reconnect
	for {
		if err := consumer.Consume(consumerCtx, c.topics, handler); err != nil {
			logger.Log.Error("Error from consumer, will reconnect", zap.Error(err))
			// If Consume returns error (e.g., rebalance), we just loop and retry
		}
		if consumerCtx.Err() != nil {
			logger.Log.Info("Consumer loop exiting due to context cancellation")
			return
		}
		// Brief pause before next consume call
		time.Sleep(500 * time.Millisecond)
	}
}

type groupHandler struct {
	handler Handler
}

func (h *groupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h *groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		logger.Log.Debug("Message claimed", zap.String("topic", msg.Topic), zap.Int64("offset", msg.Offset))
		if err := h.handler(sess.Context(), string(msg.Key), msg.Value); err != nil {
			logger.Log.Error("Handler failed", zap.Error(err))
			// Continue processing, mark offset (at-least-once)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

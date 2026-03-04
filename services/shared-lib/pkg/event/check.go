package event

import (
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

// StartupCheck performs mandatory Kafka validations
type StartupCheck struct {
	brokers []string
}

func NewStartupCheck(brokers []string) *StartupCheck {
	return &StartupCheck{brokers: brokers}
}

// EnsureKafkaConnectivity verifies we can talk to the brokers
func (s *StartupCheck) EnsureKafkaConnectivity() error {
	logger.Log.Info("Verifying Kafka connectivity...", zap.Strings("brokers", s.brokers))

	config := sarama.NewConfig()
	config.Net.DialTimeout = 5 * time.Second

	client, err := sarama.NewClient(s.brokers, config)
	if err != nil {
		return fmt.Errorf("kafka connectivity check failed: %w", err)
	}
	_ = client.Close()

	logger.Log.Info("Kafka connectivity verified")
	return nil
}

// EnsureTopicsExist checks if mandatory topics exist
func (s *StartupCheck) EnsureTopicsExist(topics []string) error {
	logger.Log.Info("Verifying mandatory topics exist...", zap.Strings("topics", topics))

	config := sarama.NewConfig()
	client, err := sarama.NewClient(s.brokers, config)
	if err != nil {
		return fmt.Errorf("failed to create kafka client for topic check: %w", err)
	}
	defer client.Close()

	// Refresh metadata to ensure we see all topics
	if err := client.RefreshMetadata(); err != nil {
		return fmt.Errorf("failed to refresh metadata: %w", err)
	}

	existingTopics, err := client.Topics()
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	missing := []string{}
	existingMap := make(map[string]bool)
	for _, t := range existingTopics {
		existingMap[t] = true
	}

	for _, required := range topics {
		if !existingMap[required] {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("startup failed: missing mandatory topics: %v. AUTO-CREATION IS DISABLED", missing)
	}

	logger.Log.Info("All mandatory topics verified")
	return nil
}

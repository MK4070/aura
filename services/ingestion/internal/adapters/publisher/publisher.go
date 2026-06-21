package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Publisher struct {
	client *kgo.Client
	// topic  string
}

func NewPublisher(brokers []string, topic string) (*Publisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),

		kgo.DefaultProduceTopic(topic),
	)
	if err != nil {
		return nil, fmt.Errorf("failure to init kafka client: %w", err)
	}

	if err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping kafka brokers: %w", err)
	}

	return &Publisher{
		client: client,
		// topic:  topic,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, event domain.DocumentUploadedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	record := &kgo.Record{
		// Topic: p.topic,
		Value: payload,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("failed to produce message to kafka: %w", err)
	}

	return nil
}

func (p *Publisher) Close(ctx context.Context) error {
	var err error

	if flushErr := p.client.Flush(ctx); flushErr != nil {
		err = fmt.Errorf("failed to flush events: %w", flushErr)
	}

	p.client.Close()
	return err
}

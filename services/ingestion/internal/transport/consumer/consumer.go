package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/process"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	Brokers    []string
	Topic      string
	GroupID    string
	NumWorkers int
	MaxRetries int
}

type Consumer struct {
	Client *kgo.Client
	Store  process.DocumentStore
	Logger *slog.Logger
	Cfg    Config
}

func NewConsumer(cfg Config, store process.DocumentStore, logger *slog.Logger) (*Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.ConsumerGroup(cfg.GroupID),

		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to init kafka client: %w", err)
	}

	return &Consumer{Client: client, Store: store, Logger: logger, Cfg: cfg}, nil
}

type job struct {
	record *kgo.Record
	cmd    process.ProcessCommand
}

func (c *Consumer) Start(ctx context.Context, processor process.Processor) error {
	jobs := make(chan job, c.Cfg.NumWorkers)
	var wg sync.WaitGroup

	for i := range c.Cfg.NumWorkers {
		wg.Go(func() {
			c.runWorker(ctx, jobs, processor, i)
		})
	}

	c.dispatch(ctx, jobs)

	close(jobs)
	wg.Wait()

	c.Client.Close()
	return nil
}

func (c *Consumer) dispatch(ctx context.Context, jobs chan<- job) {
	for {
		if ctx.Err() != nil {
			return
		}

		fetches := c.Client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			if errors.Is(err, context.Canceled) {
				return
			}
			c.Logger.Error("fetch error", "topic", topic, "partition", partition, "err", err)
		})

		fetches.EachRecord(func(r *kgo.Record) {
			cmd, err := deserialize(r)
			if err != nil {
				c.Logger.Error("failed to deserialize",
					"offset", r.Offset,
					"err", err,
				)

				// dropping so we don't fetch it again
				// TODO: use DLQ first before marking

				c.Client.MarkCommitRecords(r)
				return
			}

			select {
			case jobs <- job{record: r, cmd: cmd}:
			case <-ctx.Done():
				// do not mark record, so it will be redelivered on restart
			}
		})
	}
}

type processResult int

const (
	resultSuccess processResult = iota
	resultPermanentFailure
	resultAborted
)

func (c *Consumer) runWorker(ctx context.Context, jobs <-chan job, processor process.Processor, workerID int) {
	for j := range jobs {
		result, err := c.processWithRetry(ctx, j, processor)

		switch result {
		case resultSuccess:
			// Mark this document as successfully processed
			if storeErr := c.Store.MarkAsProcessed(context.Background(), j.cmd.DocumentID); storeErr != nil {
				c.Logger.Error(
					"failed to mark document as processed",
					"worker_id", workerID,
					"document_id", j.cmd.DocumentID,
					"error", storeErr,
				)
			}
			c.Client.MarkCommitRecords(j.record)

		case resultPermanentFailure:
			// Retries exhausted. Mark as failed using the returned error.
			if storeErr := c.Store.MarkAsFailed(context.Background(), j.cmd.DocumentID, err.Error()); storeErr != nil {
				c.Logger.Error(
					"failed to mark document as failed",
					"worker_id", workerID,
					"document_id", j.cmd.DocumentID,
					"error", storeErr,
				)
			}

			// We still commit the offset because we reached a terminal state (recorded in DB)
			c.Client.MarkCommitRecords(j.record)

		case resultAborted:
			// Context canceled (shutdown). Do not mark DB, do not commit offset.
			c.Logger.Debug("worker aborted; skipping DB update and kafka commit",
				"document_id", j.cmd.DocumentID,
			)
		}
	}
}

func (c *Consumer) processWithRetry(ctx context.Context, j job, processor process.Processor) (processResult, error) {
	var err error
	for attempt := range c.Cfg.MaxRetries + 1 {
		if attempt > 0 {
			wait := exponentialBackoff(attempt)
			c.Logger.Warn("retrying", "doc", j.cmd.DocumentID, "attempt", attempt)

			select {
			case <-time.After(wait):

			case <-ctx.Done():
				// aborted by shutdown. do not mark record
				return resultAborted, ctx.Err()
			}
		}

		c.Logger.Debug("Processing:",
			"id", j.cmd.DocumentID,
			"record", j.record.Value,
		)
		err = processor.ProcessDocument(ctx, j.cmd)
		if err == nil {
			return resultSuccess, nil
		}
	}

	c.Logger.Error("processing failed permanently", "doc", j.cmd.DocumentID, "err", err)
	return resultPermanentFailure, err
}

func exponentialBackoff(attempt int) time.Duration {
	base := 100 * time.Millisecond
	cap := 10 * time.Second
	exp := min(time.Duration(math.Pow(2, float64(attempt)))*base, cap)
	return time.Duration(rand.Int64N(int64(exp)))
}

func deserialize(r *kgo.Record) (process.ProcessCommand, error) {
	var event domain.DocumentUploadedEvent
	if err := json.Unmarshal(r.Value, &event); err != nil {
		return process.ProcessCommand{}, fmt.Errorf("unmarshal record: %w", err)
	}
	return process.ProcessCommand(event.Data), nil
}

package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) PublishMetrics(ctx context.Context, batch MetricsBatch) error {
	b, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal metrics batch: %w", err)
	}

	subject := fmt.Sprintf("infrasense.metrics.v1.default.%s", batch.DeviceID)
	msgID := metricsMsgID(batch.DeviceID, batch.CollectedAt)

	_, err = p.js.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    b,
		Header:  nats.Header{"Nats-Msg-Id": []string{msgID}},
	}, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish metrics: %w", err)
	}
	return nil
}

func (p *Publisher) PublishEvent(ctx context.Context, ev HardwareEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	subject := fmt.Sprintf("infrasense.events.v1.default.%s", ev.DeviceID)
	msgID := ev.DedupeKey
	if msgID == "" {
		msgID = metricsMsgID(ev.DeviceID, ev.ObservedAt)
	}

	_, err = p.js.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    b,
		Header:  nats.Header{"Nats-Msg-Id": []string{msgID}},
	}, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil
}

func metricsMsgID(deviceID string, t time.Time) string {
	h := sha256.Sum256([]byte(deviceID + "|" + t.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h[:])
}


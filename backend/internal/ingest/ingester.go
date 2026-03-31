package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/metrics"
	"github.com/infrasense/backend/internal/models"
	"github.com/nats-io/nats.go"
)

type Config struct {
	NATSURL            string
	VictoriaWriteURL   string
	VMBatchSize        int
	VMBatchTimeout     time.Duration
	DurableNameMetrics string
	DurableNameEvents  string
}

type Ingester struct {
	dbRepo   *db.HardwareEventRepository
	vmWriter *metrics.VictoriaMetricsWriter
	js       nats.JetStreamContext
}

func New(js nats.JetStreamContext, dbRepo *db.HardwareEventRepository, vmWriter *metrics.VictoriaMetricsWriter) *Ingester {
	return &Ingester{js: js, dbRepo: dbRepo, vmWriter: vmWriter}
}

func EnsureStreams(js nats.JetStreamContext) error {
	streams := []struct {
		name     string
		subjects []string
		maxAge   time.Duration
	}{
		{name: "INFRA_METRICS", subjects: []string{"infrasense.metrics.v1.>"}, maxAge: 6 * time.Hour},
		{name: "INFRA_EVENTS", subjects: []string{"infrasense.events.v1.>"}, maxAge: 72 * time.Hour},
	}

	for _, s := range streams {
		if _, err := js.StreamInfo(s.name); err == nil {
			continue
		}
		_, err := js.AddStream(&nats.StreamConfig{
			Name:      s.name,
			Subjects:  s.subjects,
			Retention: nats.LimitsPolicy,
			MaxAge:    s.maxAge,
			Storage:   nats.FileStorage,
			Discard:   nats.DiscardOld,
		})
		if err != nil {
			return fmt.Errorf("add stream %s: %w", s.name, err)
		}
	}
	return nil
}

func (i *Ingester) Run(ctx context.Context, cfg Config) error {
	i.vmWriter.Start()
	defer i.vmWriter.Stop()

	if err := i.consumeMetrics(ctx, cfg.DurableNameMetrics); err != nil {
		return err
	}
	if err := i.consumeEvents(ctx, cfg.DurableNameEvents); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (i *Ingester) consumeMetrics(ctx context.Context, durable string) error {
	sub, err := i.js.PullSubscribe("infrasense.metrics.v1.>", durable, nats.BindStream("INFRA_METRICS"))
	if err != nil {
		return fmt.Errorf("subscribe metrics: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgs, err := sub.Fetch(100, nats.MaxWait(2*time.Second))
			if err != nil && err != nats.ErrTimeout {
				log.Printf("metrics fetch error: %v", err)
				continue
			}
			for _, m := range msgs {
				if err := i.handleMetricsMsg(ctx, m); err != nil {
					log.Printf("metrics handle error: %v", err)
					_ = m.Nak()
				} else {
					_ = m.Ack()
				}
			}
		}
	}()

	return nil
}

func (i *Ingester) handleMetricsMsg(ctx context.Context, msg *nats.Msg) error {
	var batch MetricsBatch
	if err := json.Unmarshal(msg.Data, &batch); err != nil {
		return fmt.Errorf("decode metrics batch: %w", err)
	}
	for _, s := range batch.Samples {
		if s.Labels == nil {
			s.Labels = map[string]string{}
		}
		// enforce core labels
		if _, ok := s.Labels["device_id"]; !ok {
			s.Labels["device_id"] = batch.DeviceID
		}
		if _, ok := s.Labels["source_protocol"]; !ok {
			s.Labels["source_protocol"] = batch.Source
		}
		if err := i.vmWriter.WriteMetric(s.Name, s.Value, s.Labels, s.Timestamp); err != nil {
			return err
		}
	}
	return nil
}

func (i *Ingester) consumeEvents(ctx context.Context, durable string) error {
	sub, err := i.js.PullSubscribe("infrasense.events.v1.>", durable, nats.BindStream("INFRA_EVENTS"))
	if err != nil {
		return fmt.Errorf("subscribe events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgs, err := sub.Fetch(100, nats.MaxWait(2*time.Second))
			if err != nil && err != nats.ErrTimeout {
				log.Printf("events fetch error: %v", err)
				continue
			}
			for _, m := range msgs {
				if err := i.handleEventMsg(ctx, m); err != nil {
					log.Printf("event handle error: %v", err)
					_ = m.Nak()
				} else {
					_ = m.Ack()
				}
			}
		}
	}()

	return nil
}

func (i *Ingester) handleEventMsg(ctx context.Context, msg *nats.Msg) error {
	var ev HardwareEvent
	if err := json.Unmarshal(msg.Data, &ev); err != nil {
		return fmt.Errorf("decode hardware event: %w", err)
	}

	deviceID, err := uuid.Parse(ev.DeviceID)
	if err != nil {
		return fmt.Errorf("invalid device_id: %w", err)
	}

	dedupe := ev.DedupeKey
	if dedupe == "" {
		dedupe = stableDedupe(ev)
	}

	modelEv := models.HardwareEvent{
		DeviceID:       deviceID,
		OccurredAt:     ev.OccurredAt,
		ObservedAt:     ev.ObservedAt,
		SourceProtocol: ev.SourceProtocol,
		Vendor:         ev.Vendor,
		Model:          ev.Model,
		Firmware:       ev.Firmware,
		Component:      ev.Component,
		EventType:      ev.EventType,
		Severity:       ev.Severity,
		Message:        ev.Message,
		Classification: ev.Classification,
		Raw:            ev.Raw,
		DedupeKey:      dedupe,
	}

	return i.dbRepo.InsertIfNotExists(ctx, modelEv)
}

func stableDedupe(ev HardwareEvent) string {
	h := sha256.New()
	_, _ = h.Write([]byte(ev.DeviceID))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(ev.SourceProtocol))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(ev.Component))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(ev.EventType))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(ev.Severity))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(ev.Message))
	if ev.OccurredAt != nil {
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(ev.OccurredAt.UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func EnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}


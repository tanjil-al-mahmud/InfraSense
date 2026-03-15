package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

type VictoriaMetricsWriter struct {
	url          string
	batchSize    int
	batchTimeout time.Duration
	buffer       []prompb.TimeSeries
	bufferMutex  sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	httpClient   *http.Client
}

func NewVictoriaMetricsWriter(url string, batchSize int, batchTimeout time.Duration) *VictoriaMetricsWriter {
	ctx, cancel := context.WithCancel(context.Background())
	return &VictoriaMetricsWriter{
		url:          url,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		buffer:       make([]prompb.TimeSeries, 0, batchSize),
		ctx:          ctx,
		cancel:       cancel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (w *VictoriaMetricsWriter) Start() {
	w.wg.Add(1)
	go w.flushLoop()
}

func (w *VictoriaMetricsWriter) Stop() {
	w.cancel()
	w.wg.Wait()
	w.flush() // Final flush
}

func (w *VictoriaMetricsWriter) WriteMetric(name string, value float64, labels map[string]string, timestamp time.Time) error {
	// Convert to Prometheus TimeSeries
	promLabels := []prompb.Label{
		{Name: "__name__", Value: name},
	}
	for k, v := range labels {
		promLabels = append(promLabels, prompb.Label{Name: k, Value: v})
	}

	ts := prompb.TimeSeries{
		Labels: promLabels,
		Samples: []prompb.Sample{
			{
				Value:     value,
				Timestamp: timestamp.UnixMilli(),
			},
		},
	}

	w.bufferMutex.Lock()
	w.buffer = append(w.buffer, ts)
	shouldFlush := len(w.buffer) >= w.batchSize
	w.bufferMutex.Unlock()

	if shouldFlush {
		return w.flush()
	}

	return nil
}

func (w *VictoriaMetricsWriter) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.flush(); err != nil {
				log.Printf("Error flushing metrics: %v", err)
			}
		}
	}
}

func (w *VictoriaMetricsWriter) flush() error {
	w.bufferMutex.Lock()
	if len(w.buffer) == 0 {
		w.bufferMutex.Unlock()
		return nil
	}

	// Copy buffer and reset
	batch := make([]prompb.TimeSeries, len(w.buffer))
	copy(batch, w.buffer)
	w.buffer = w.buffer[:0]
	w.bufferMutex.Unlock()

	// Send with retry logic
	return w.sendWithRetry(batch, 3)
}

func (w *VictoriaMetricsWriter) sendWithRetry(batch []prompb.TimeSeries, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("Retrying metrics push after %v (attempt %d/%d)", backoff, attempt+1, maxRetries)
			time.Sleep(backoff)
		}

		err := w.send(batch)
		if err == nil {
			if attempt > 0 {
				log.Printf("Metrics push succeeded after %d retries", attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("Metrics push failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
	}

	return fmt.Errorf("failed to push metrics after %d attempts: %w", maxRetries, lastErr)
}

func (w *VictoriaMetricsWriter) send(batch []prompb.TimeSeries) error {
	// Create WriteRequest
	writeRequest := &prompb.WriteRequest{
		Timeseries: batch,
	}

	// Marshal to protobuf
	data, err := writeRequest.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal write request: %w", err)
	}

	// Compress with snappy
	compressed := snappy.Encode(nil, data)

	// Send HTTP request
	req, err := http.NewRequest("POST", w.url, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully pushed %d metrics to VictoriaMetrics", len(batch))
	return nil
}

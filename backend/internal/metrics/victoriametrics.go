package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

// VictoriaMetricsWriter batches Prometheus remote_write samples and sends them to VictoriaMetrics.
type VictoriaMetricsWriter struct {
	url          string
	batchSize    int
	batchTimeout time.Duration

	buffer      []prompb.TimeSeries
	bufferMutex sync.Mutex

	client *http.Client
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewVictoriaMetricsWriter(url string, batchSize int, batchTimeout time.Duration) *VictoriaMetricsWriter {
	ctx, cancel := context.WithCancel(context.Background())
	return &VictoriaMetricsWriter{
		url:          url,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		buffer:       make([]prompb.TimeSeries, 0, batchSize),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (w *VictoriaMetricsWriter) Start() {
	w.wg.Add(1)
	go w.flushLoop()
}

func (w *VictoriaMetricsWriter) Stop() {
	w.cancel()
	w.wg.Wait()
	_ = w.flush()
}

func (w *VictoriaMetricsWriter) WriteMetric(name string, value float64, labels map[string]string, timestamp time.Time) error {
	ts := prompb.TimeSeries{
		Labels: []prompb.Label{
			{Name: "__name__", Value: name},
		},
		Samples: []prompb.Sample{
			{
				Value:     value,
				Timestamp: timestamp.UnixMilli(),
			},
		},
	}

	for k, v := range labels {
		ts.Labels = append(ts.Labels, prompb.Label{Name: k, Value: v})
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

	toSend := make([]prompb.TimeSeries, len(w.buffer))
	copy(toSend, w.buffer)
	w.buffer = w.buffer[:0]
	w.bufferMutex.Unlock()

	return w.sendWithRetry(toSend, 3)
}

func (w *VictoriaMetricsWriter) sendWithRetry(timeSeries []prompb.TimeSeries, maxRetries int) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			time.Sleep(backoff)
		}
		if err := w.send(timeSeries); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("failed to push metrics after %d attempts: %w", maxRetries, lastErr)
}

func (w *VictoriaMetricsWriter) send(timeSeries []prompb.TimeSeries) error {
	writeReq := &prompb.WriteRequest{Timeseries: timeSeries}
	data, err := writeReq.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal write request: %w", err)
	}

	compressed := snappy.Encode(nil, data)
	req, err := http.NewRequestWithContext(w.ctx, http.MethodPost, w.url, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	return nil
}


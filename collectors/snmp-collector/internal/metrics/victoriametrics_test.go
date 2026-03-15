package metrics

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestVictoriaMetricsWriter_RetryLogic tests the exponential backoff retry logic
func TestVictoriaMetricsWriter_RetryLogic(t *testing.T) {
	var attemptCount int32

	// Create a test server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create writer
	writer := NewVictoriaMetricsWriter(server.URL, 10, 1*time.Second)
	writer.Start()
	defer writer.Stop()

	// Write a metric
	labels := map[string]string{"device_id": "123"}
	err := writer.WriteMetric("test_metric", 42.0, labels, time.Now())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Force flush
	writer.flush()

	// Verify retry happened
	if atomic.LoadInt32(&attemptCount) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestVictoriaMetricsWriter_MaxRetriesExceeded tests that retry stops after max attempts
func TestVictoriaMetricsWriter_MaxRetriesExceeded(t *testing.T) {
	var attemptCount int32

	// Create a test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create writer
	writer := NewVictoriaMetricsWriter(server.URL, 10, 1*time.Second)
	writer.Start()
	defer writer.Stop()

	// Write a metric
	labels := map[string]string{"device_id": "123"}
	err := writer.WriteMetric("test_metric", 42.0, labels, time.Now())

	if err != nil {
		t.Fatalf("WriteMetric should not return error immediately: %v", err)
	}

	// Force flush
	err = writer.flush()

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	// Verify max retries (3 attempts)
	if atomic.LoadInt32(&attemptCount) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestVictoriaMetricsWriter_Batching tests that metrics are batched correctly
func TestVictoriaMetricsWriter_Batching(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create writer with batch size of 5
	writer := NewVictoriaMetricsWriter(server.URL, 5, 10*time.Second)
	writer.Start()
	defer writer.Stop()

	// Write 12 metrics
	labels := map[string]string{"device_id": "123"}
	for i := 0; i < 12; i++ {
		writer.WriteMetric("test_metric", float64(i), labels, time.Now())
	}

	// Should have triggered 2 flushes (5 + 5 = 10 metrics)
	time.Sleep(100 * time.Millisecond)

	count := atomic.LoadInt32(&requestCount)
	if count != 2 {
		t.Errorf("Expected 2 batch requests, got %d", count)
	}
}

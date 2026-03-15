package collector

import (
	"log"
	"math"
	"time"
)

const (
	maxBackoffDuration = 10 * time.Minute
	initialBackoff     = 1 * time.Second
)

type RetryState struct {
	deviceID     int64
	hostname     string
	failureCount int
	lastAttempt  time.Time
	nextAttempt  time.Time
}

type RetryManager struct {
	states map[int64]*RetryState
}

func NewRetryManager() *RetryManager {
	return &RetryManager{
		states: make(map[int64]*RetryState),
	}
}

func (rm *RetryManager) ShouldRetry(deviceID int64, hostname string) bool {
	state, exists := rm.states[deviceID]
	if !exists {
		// First attempt
		return true
	}

	// Check if enough time has passed for next retry
	return time.Now().After(state.nextAttempt)
}

func (rm *RetryManager) RecordFailure(deviceID int64, hostname string) time.Duration {
	state, exists := rm.states[deviceID]
	if !exists {
		state = &RetryState{
			deviceID:     deviceID,
			hostname:     hostname,
			failureCount: 0,
		}
		rm.states[deviceID] = state
	}

	state.failureCount++
	state.lastAttempt = time.Now()

	// Calculate exponential backoff: 1s, 2s, 4s, 8s, ..., max 10 minutes
	backoff := time.Duration(math.Pow(2, float64(state.failureCount-1))) * initialBackoff
	if backoff > maxBackoffDuration {
		backoff = maxBackoffDuration
	}

	state.nextAttempt = time.Now().Add(backoff)

	log.Printf("Device %s failed %d times, next retry in %v", hostname, state.failureCount, backoff)

	return backoff
}

func (rm *RetryManager) RecordSuccess(deviceID int64) {
	// Reset failure count on success
	delete(rm.states, deviceID)
}

func (rm *RetryManager) GetFailureCount(deviceID int64) int {
	state, exists := rm.states[deviceID]
	if !exists {
		return 0
	}
	return state.failureCount
}

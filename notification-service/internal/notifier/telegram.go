package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/infrasense/notification-service/internal/webhook"
)

type TelegramNotifier struct {
	botToken    string
	chatID      string
	client      *http.Client
	rateLimiter *RateLimiter
}

type RateLimiter struct {
	tokens    int
	maxTokens int
	refillRate time.Duration
	lastRefill time.Time
	mutex     sync.Mutex
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		// Rate limit: 30 messages per minute = 1 message per 2 seconds
		rateLimiter: NewRateLimiter(30, 2*time.Second),
	}
}

func (t *TelegramNotifier) Name() string {
	return "Telegram"
}

func (t *TelegramNotifier) Send(alert webhook.NotificationAlert) error {
	// Wait for rate limiter
	for !t.rateLimiter.Allow() {
		time.Sleep(100 * time.Millisecond)
	}

	return t.sendWithRetry(alert, 3)
}

func (t *TelegramNotifier) sendWithRetry(alert webhook.NotificationAlert, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			time.Sleep(backoff)
		}

		err := t.sendMessage(alert)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (t *TelegramNotifier) sendMessage(alert webhook.NotificationAlert) error {
	message := webhook.FormatAlertMessage(alert)

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

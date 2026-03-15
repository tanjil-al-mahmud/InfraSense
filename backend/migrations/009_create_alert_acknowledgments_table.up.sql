CREATE TABLE IF NOT EXISTS alert_acknowledgments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_fingerprint VARCHAR(255) NOT NULL,
    acknowledged_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(alert_fingerprint)
);

CREATE INDEX idx_alert_acknowledgments_user_id ON alert_acknowledgments(user_id);
CREATE INDEX idx_alert_acknowledgments_fingerprint ON alert_acknowledgments(alert_fingerprint);

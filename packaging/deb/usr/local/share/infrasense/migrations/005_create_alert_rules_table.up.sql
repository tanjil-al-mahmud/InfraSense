CREATE TABLE alert_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  metric_name VARCHAR(255) NOT NULL,
  comparison_operator VARCHAR(10) NOT NULL,
  threshold_value NUMERIC NOT NULL,
  severity VARCHAR(20) NOT NULL,
  device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
  device_group_id UUID REFERENCES device_groups(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_device_id ON alert_rules(device_id);
CREATE INDEX idx_alert_rules_enabled ON alert_rules(enabled);

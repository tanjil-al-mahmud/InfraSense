CREATE TABLE collector_status (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  collector_name VARCHAR(255) NOT NULL UNIQUE,
  collector_type VARCHAR(50) NOT NULL,
  status VARCHAR(20) NOT NULL,
  last_poll_time TIMESTAMP,
  last_success_time TIMESTAMP,
  last_error TEXT,
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

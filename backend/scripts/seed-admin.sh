#!/bin/bash
# Seed admin user into InfraSense database
# Password hash below = "admin123" (bcrypt cost 12)
# Run from the repo root: bash backend/scripts/seed-admin.sh

docker exec infrasense-postgres psql -U infrasense -d infrasense -c \
  "INSERT INTO users (id, username, email, password_hash, role, enabled, created_at, updated_at)
   VALUES (
     gen_random_uuid(),
     'admin',
     'admin@infrasense.local',
     '\$2a\$12\$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpSVeu1Ky',
     'admin',
     true,
     NOW(),
     NOW()
   )
   ON CONFLICT (username) DO UPDATE
     SET password_hash = '\$2a\$12\$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpSVeu1Ky',
         enabled = true;"

echo "Admin user seeded. Login: admin / admin123"
echo "Change the password immediately after first login."

-- Create default admin user
-- Password: Admin@123456 (bcrypt hash cost 12)

INSERT INTO users (id, username, password_hash, email, role, enabled, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'admin',
    '$2b$12$VReytW0YQwzSdQANoMvWwux4lMTFxRHx.eO2ZVpBzG8mar25XY55K',
    'admin@example.com',
    'admin',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (username) DO UPDATE
    SET password_hash = '$2b$12$VReytW0YQwzSdQANoMvWwux4lMTFxRHx.eO2ZVpBzG8mar25XY55K',
        enabled = true,
        updated_at = NOW();

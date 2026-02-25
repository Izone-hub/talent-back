-- Seed script to create first admin user
-- Password: admin123 (hashed with bcrypt)
-- You can generate a new hash using: https://bcrypt-generator.com/ or Go's bcrypt package

INSERT INTO users (
    id,
    first_name,
    last_name,
    email,
    password,
    role,
    auth_provider,
    created_at,
    updated_at
)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Admin',
    'User',
    'admin@talentplatform.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- 'admin123'
    'admin',
    'email',
    now(),
    now()
)
ON CONFLICT (email) DO NOTHING;

-- Note: The password hash above is for 'admin123'
-- To generate a new hash, use Go:
-- hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost)
-- Then replace the hash in the INSERT statement above

-- Seed data for Arc LMS
-- This script creates initial test data for development

BEGIN;

-- Create a test super admin (password: admin123)
INSERT INTO users (id, tenant_id, role, email, password_hash, first_name, last_name, status, permissions, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    NULL,  -- SUPER_ADMIN has no tenant
    'SUPER_ADMIN',
    'superadmin@arclms.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYILSWowUBm',  -- bcrypt hash of "admin123"
    'Super',
    'Admin',
    'ACTIVE',
    '["tenant:create", "tenant:read", "tenant:update", "tenant:delete", "tenant:suspend", "user:create", "user:read", "user:update", "user:delete", "billing:read", "audit:read"]'::jsonb,
    NOW(),
    NOW()
);

-- Create a test tenant
INSERT INTO tenants (id, name, school_type, contact_email, address, logo, status, configuration, billing_contact, principal_admin_id, created_at, updated_at)
VALUES (
    '10000000-0000-0000-0000-000000000001',
    'Test Primary School',
    'PRIMARY',
    'info@testschool.com',
    '123 Education Street, Lagos, Nigeria',
    '',
    'ACTIVE',
    '{
        "timezone": "Africa/Lagos",
        "default_grade_weighting": {"continuous_assessment": 40, "examination": 60},
        "attendance_threshold": 75,
        "invoice_grace_period_days": 14,
        "suspension_threshold_days": 30
    }'::jsonb,
    '{
        "name": "Finance Manager",
        "email": "finance@testschool.com",
        "phone": "+234-123-456-7890"
    }'::jsonb,
    '00000000-0000-0000-0000-000000000001',
    NOW(),
    NOW()
);

-- Create a test admin for the tenant (password: admin123)
INSERT INTO users (id, tenant_id, role, email, password_hash, first_name, last_name, status, permissions, created_at, updated_at)
VALUES (
    '10000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000001',
    'ADMIN',
    'admin@testschool.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYILSWowUBm',
    'School',
    'Administrator',
    'ACTIVE',
    '["session:create", "session:read", "session:update", "session:delete", "term:create", "term:read", "term:update", "class:create", "class:read", "class:update", "course:create", "course:read", "course:update", "user:create", "user:read", "user:update", "user:invite", "billing:read"]'::jsonb,
    NOW(),
    NOW()
);

-- Create a test tutor (password: tutor123)
INSERT INTO users (id, tenant_id, role, email, password_hash, first_name, last_name, status, created_at, updated_at)
VALUES (
    '10000000-0000-0000-0000-000000000003',
    '10000000-0000-0000-0000-000000000001',
    'TUTOR',
    'tutor@testschool.com',
    '$2a$12$3fK8YQjK8x5h3X9R6Xo1xOz.MQJ3h1x7L3Y5G3Y5G3Y5G3Y5G3Y5G',
    'Jane',
    'Teacher',
    'ACTIVE',
    NOW(),
    NOW()
);

-- Create a test student (password: student123)
INSERT INTO users (id, tenant_id, role, email, password_hash, first_name, last_name, status, created_at, updated_at)
VALUES (
    '10000000-0000-0000-0000-000000000004',
    '10000000-0000-0000-0000-000000000001',
    'STUDENT',
    'student@testschool.com',
    '$2a$12$5fK8YQjK8x5h3X9R6Xo1xOz.MQJ3h1x7L3Y5G3Y5G3Y5G3Y5G3Y5G',
    'John',
    'Student',
    'ACTIVE',
    NOW(),
    NOW()
);

-- Create a test session (2025/2026)
INSERT INTO sessions (id, tenant_id, label, start_year, end_year, status, created_at, updated_at)
VALUES (
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '2025/2026',
    2025,
    2026,
    'ACTIVE',
    NOW(),
    NOW()
);

-- Create test terms
INSERT INTO terms (id, tenant_id, session_id, ordinal, start_date, end_date, status, holidays, created_at, updated_at)
VALUES
    (
        '30000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'FIRST',
        '2025-09-15',
        '2025-12-20',
        'ACTIVE',
        '[
            {"date": "2025-10-01", "name": "Independence Day", "type": "PUBLIC_HOLIDAY"},
            {"date": "2025-12-25", "name": "Christmas Day", "type": "PUBLIC_HOLIDAY"}
        ]'::jsonb,
        NOW(),
        NOW()
    ),
    (
        '30000000-0000-0000-0000-000000000002',
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'SECOND',
        '2026-01-05',
        '2026-04-15',
        'DRAFT',
        '[]'::jsonb,
        NOW(),
        NOW()
    ),
    (
        '30000000-0000-0000-0000-000000000003',
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'THIRD',
        '2026-04-20',
        '2026-07-25',
        'DRAFT',
        '[]'::jsonb,
        NOW(),
        NOW()
    );

-- Create test classes
INSERT INTO classes (id, tenant_id, session_id, name, arm, level, capacity, status, created_at, updated_at)
VALUES
    (
        '40000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'Primary 1',
        'A',
        'PRIMARY',
        30,
        'ACTIVE',
        NOW(),
        NOW()
    ),
    (
        '40000000-0000-0000-0000-000000000002',
        '10000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'Primary 2',
        'A',
        'PRIMARY',
        30,
        'ACTIVE',
        NOW(),
        NOW()
    );

-- Enroll student in class
INSERT INTO enrollments (id, tenant_id, student_id, class_id, session_id, status, enrolled_at, created_at, updated_at)
VALUES (
    '50000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000004',
    '40000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    'ACTIVE',
    NOW(),
    NOW(),
    NOW()
);

-- Create test courses
INSERT INTO courses (id, tenant_id, class_id, name, code, subject, tutor_id, status, created_at, updated_at)
VALUES
    (
        '60000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000001',
        'Mathematics',
        'MATH101',
        'Mathematics',
        '10000000-0000-0000-0000-000000000003',
        'ACTIVE',
        NOW(),
        NOW()
    ),
    (
        '60000000-0000-0000-0000-000000000002',
        '10000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000001',
        'English Language',
        'ENG101',
        'English',
        '10000000-0000-0000-0000-000000000003',
        'ACTIVE',
        NOW(),
        NOW()
    );

COMMIT;

-- Print success message
SELECT 'Seed data inserted successfully!' AS message;
SELECT 'Test accounts:' AS message;
SELECT '  Super Admin: superadmin@arclms.com / admin123' AS message;
SELECT '  School Admin: admin@testschool.com / admin123' AS message;
SELECT '  Tutor: tutor@testschool.com / tutor123' AS message;
SELECT '  Student: student@testschool.com / student123' AS message;

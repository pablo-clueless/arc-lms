# Database Seeding

## Overview

The Arc LMS API automatically seeds essential data when the application starts. This ensures that the system always has the necessary initial data to function properly.

## SuperAdmin Seed

### Default Configuration

On first startup, the application will automatically create a **SuperAdmin** user with the following credentials:

- **Email**: `smsnmicheal@gmail.com`
- **Password**: `Asdfgh123@`
- **Role**: `SUPER_ADMIN`
- **Status**: `ACTIVE`
- **Permissions**: `*:*` (Full system access)

### Customizing SuperAdmin Credentials

You can customize the superadmin credentials by setting environment variables in your `.env` file:

```env
# SuperAdmin Seed Configuration
SUPERADMIN_EMAIL=your-admin@example.com
SUPERADMIN_PASSWORD=YourSecurePassword123!
SUPERADMIN_FIRST_NAME=Your
SUPERADMIN_LAST_NAME=Name
```

### How It Works

1. **On Startup**: When the application starts, it runs the seed script after establishing the database connection.

2. **Idempotent**: The seed script checks if a user with the specified email already exists:
   - ✅ **If user exists**: Skips seeding (logs: "👨‍💼 SuperAdmin already exists")
   - ✅ **If user doesn't exist**: Creates the superadmin user

3. **Non-Blocking**: If seeding fails, the application will log a warning but continue to start normally. This prevents deployment issues.

### Startup Log Example

When the superadmin is created:
```
✅ Database connection established
🌱 Running database seeds...
✅ SuperAdmin created successfully: smsnmicheal@gmail.com
  - Role: SUPER_ADMIN
  - Status: ACTIVE
  - Permissions: Full system access
✅ Database seeding completed
```

When the superadmin already exists:
```
🌐 Database connection established
🌱 Running database seeds...
👨‍💼 SuperAdmin already exists: smsnmicheal@gmail.com (skipping seed)
✅ Database seeding completed
```

## Security Considerations

### Production Deployment

⚠️ **IMPORTANT**: For production deployments:

1. **Change Default Credentials**: Always customize the superadmin credentials using environment variables
2. **Use Strong Passwords**: Ensure the password meets security requirements (minimum 8 characters, mix of upper/lower case, numbers, special characters)
3. **Secure Environment Variables**: Store credentials securely (e.g., AWS Secrets Manager, Azure Key Vault, Kubernetes Secrets)

### Password Security

- Passwords are hashed using **bcrypt** before storage
- The password hash uses a cost factor of **10** (default bcrypt cost)
- Plain text passwords are never stored in the database

## SuperAdmin Capabilities

A SuperAdmin user has unrestricted access to:

- ✅ Create, update, and delete **tenants**
- ✅ Suspend and reactivate tenants
- ✅ Manage users across all tenants
- ✅ Access all audit logs
- ✅ View all sessions, terms, classes, courses, and enrollments
- ✅ System-wide configuration changes

### Key Differences from Admin

| Feature | SuperAdmin | Admin |
|---------|-----------|-------|
| Tenant Scope | Cross-tenant (all tenants) | Single tenant only |
| Create Tenants | ✅ Yes | ❌ No |
| Suspend Tenants | ✅ Yes | ❌ No |
| Access All Audit Logs | ✅ Yes | ❌ Only tenant logs |
| Tenant Isolation | ❌ Not applied | ✅ Applied |

## Adding More Seeds

To add additional seed data, edit the `internal/seed/seed.go` file and add new seed functions to the `SeedAll` function:

```go
func SeedAll(db *sql.DB) error {
	log.Println("🌱 Running database seeds...")

	// Seed SuperAdmin
	if err := SeedSuperAdmin(db, superAdminConfig); err != nil {
		return fmt.Errorf("failed to seed superadmin: %w", err)
	}

	// Add more seed functions here
	// if err := SeedDefaultTenant(db); err != nil {
	//     return fmt.Errorf("failed to seed default tenant: %w", err)
	// }

	log.Println("✅ Database seeding completed")
	return nil
}
```

## Troubleshooting

### Seed Fails on Startup

If seeding fails, check:

1. **Database Connection**: Ensure the database is accessible
2. **Migration Status**: Run migrations before seeding: `make migrate-up`
3. **User Table**: Verify the `users` table exists
4. **Duplicate Email**: If a user with the email already exists, the seed will skip

### Reset SuperAdmin

To reset the superadmin user:

1. Delete the existing user from the database:
   ```sql
   DELETE FROM users WHERE email = 'smsnmicheal@gmail.com';
   ```

2. Restart the application - it will recreate the superadmin user

### Change SuperAdmin Password

After the initial seed, you can change the superadmin password via the API:

```bash
# Login as superadmin
curl -X POST http://localhost:8080/api/v1/public/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "smsnmicheal@gmail.com",
    "password": "Asdfgh123@"
  }'

# Use the access token to change password
curl -X PUT http://localhost:8080/api/v1/users/me/password \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "Asdfgh123@",
    "new_password": "NewSecurePassword123!"
  }'
```

## Related Files

- **Seed Implementation**: `internal/seed/seed.go`
- **Main Application**: `cmd/api/main.go` (calls `seed.SeedAll()`)
- **User Repository**: `internal/repository/postgres/user.go`
- **Password Hashing**: `internal/pkg/crypto/password.go`

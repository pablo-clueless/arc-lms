# Arc LMS - Learning Management System API

A comprehensive, multi-tenant Learning Management System API designed for Nigerian primary and secondary schools.

## Features

- **Multi-Tenancy**: Isolated data per school with tenant-scoped access
- **RBAC**: Fine-grained role-based access control (SUPER_ADMIN, ADMIN, TUTOR, STUDENT)
- **Idempotency**: Redis-backed idempotency for all mutating operations
- **Race Condition Prevention**: Database-level constraints and row locking
- **Auto-Billing**: Automatic invoice generation at term start
- **Timetable Generation**: Automated conflict-free timetable scheduling
- **Comprehensive Audit Logging**: Immutable audit trail for all operations
- **Rate Limiting**: Redis-based rate limiting with burst support
- **OpenAPI Documentation**: Swagger UI and ReDoc support

## Tech Stack

- **Language**: Go 1.23
- **Web Framework**: Gin
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **Migrations**: golang-migrate
- **Authentication**: JWT (15min access, 30day refresh)
- **Documentation**: OpenAPI 3.0 with Swagger UI and ReDoc

## Project Structure

```
arc-lms/
├── cmd/api/                  # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── domain/               # Domain models (16 entities)
│   ├── repository/postgres/  # Database layer
│   ├── service/              # Business logic
│   ├── handler/              # HTTP handlers
│   ├── middleware/           # Auth, RBAC, Idempotency, Rate Limiting
│   ├── pkg/                  # Utility packages (JWT, crypto, errors)
│   └── router/               # Route definitions
├── migrations/               # Database migrations (34 files)
├── docs/openapi/             # OpenAPI specification
└── scripts/                  # Utility scripts
```

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 16+
- Redis 7+
- Make (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd arc-lms
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env and set your secrets
   ```

4. **Start database and Redis (using Docker)**
   ```bash
   docker-compose up -d postgresql redis
   ```

5. **Run migrations**
   ```bash
   make migrate-up
   ```

6. **Build and run the application**
   ```bash
   make build
   make run
   ```

   Or run directly:
   ```bash
   go run cmd/api/main.go
   ```

The API will be available at `http://localhost:8080`

### Docker Setup

Run the entire stack with Docker Compose:

```bash
docker-compose up --build
```

## API Documentation

Once the server is running, access the documentation at:

- **Swagger UI**: http://localhost:8080/docs/index.html
- **ReDoc**: http://localhost:8080/redoc
- **OpenAPI Spec**: http://localhost:8080/api/v1/openapi.json
- **Health Check**: http://localhost:8080/health

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `GO_ENV` | Environment (development/production) | `development` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `JWT_ACCESS_SECRET` | JWT access token secret | Required |
| `JWT_REFRESH_SECRET` | JWT refresh token secret | Required |
| `JWT_ACCESS_TTL` | Access token TTL | `15m` |
| `JWT_REFRESH_TTL` | Refresh token TTL | `720h` (30 days) |

## Database Migrations

### Create a new migration
```bash
make migrate-create
# Enter migration name when prompted
```

### Run all pending migrations
```bash
make migrate-up
```

### Rollback last migration
```bash
make migrate-down
```

### Check migration version
```bash
make migrate-version
```

## Development

### Run with hot reload
```bash
make dev  # Requires Air
```

### Run tests
```bash
make test
```

### Run tests with coverage
```bash
make test-coverage
```

### Run linter
```bash
make lint  # Requires golangci-lint
```

### Format code
```bash
make fmt
```

## Architecture Highlights

### Idempotency

All mutating endpoints (POST/PUT/PATCH/DELETE) support the `Idempotency-Key` header:

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Idempotency-Key: unique-key-123" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Test School"}'
```

Duplicate requests with the same key return the cached response.

### Race Condition Prevention

Critical operations use:
- SELECT FOR UPDATE locks
- Database unique constraints
- Atomic transactions

Examples:
- Student enrollment (BR-003: one class per session)
- Session activation (BR-007: one active session per tenant)
- Quiz attempts (BR-018: one attempt per student)

### RBAC

Permissions follow the pattern `resource:action`:

```go
"tenant:create"     // Create tenants (SUPER_ADMIN only)
"user:read"         // Read users
"course:update"     // Update courses
"billing:delete"    // Delete billing records
```

SUPER_ADMIN and ADMIN roles have permissions arrays. TUTOR and STUDENT use role-based checks only.

### Auto-Billing

When a term is activated:
1. Count active students (with SELECT FOR UPDATE)
2. Create subscription record with snapshot count
3. Generate invoice: `count × NGN 500` (stored in kobo)
4. Send email notification

Billing is idempotent - duplicate activation attempts return existing invoice.

### Timetable Generation

Algorithm:
1. Build availability matrix (days × time slots)
2. Remove holidays and non-instructional days
3. For each course, assign periods using backtracking
4. Validate no tutor/class conflicts
5. Save in DRAFT status

Constraint enforcement:
- No tutor double-booking (exclusion constraint)
- No class double-booking (application logic)
- Balanced distribution (max periods per week)

## Business Rules

| ID | Rule |
|----|------|
| BR-001 | SUPER_ADMIN must have NULL tenant_id |
| BR-002 | Terms within a session must not overlap |
| BR-003 | Student can only be in one class per session |
| BR-007 | Only one active session per tenant |
| BR-009 | Student count is snapshotted at term start |
| BR-018 | One quiz attempt per student |

All business rules are enforced via database constraints or application logic with proper transaction handling.

## API Endpoints

### Authentication
- `POST /api/v1/public/auth/login` - Login
- `POST /api/v1/public/auth/register` - Register
- `POST /api/v1/public/auth/password-reset` - Password reset

### Tenants (SUPER_ADMIN only)
- `GET /api/v1/tenants` - List tenants
- `POST /api/v1/tenants` - Create tenant
- `GET /api/v1/tenants/:id` - Get tenant
- `PUT /api/v1/tenants/:id` - Update tenant
- `POST /api/v1/tenants/:id/suspend` - Suspend tenant

### Sessions & Terms
- `GET /api/v1/sessions` - List sessions
- `POST /api/v1/sessions` - Create session
- `POST /api/v1/sessions/:id/activate` - Activate session
- `POST /api/v1/sessions/:session_id/terms` - Create term

### Classes & Courses
- `GET /api/v1/classes` - List classes
- `POST /api/v1/classes` - Create class
- `GET /api/v1/courses` - List courses
- `POST /api/v1/courses` - Create course

### Assessments
- `GET /api/v1/assessments/quizzes` - List quizzes
- `POST /api/v1/assessments/quizzes` - Create quiz
- `POST /api/v1/assessments/quizzes/:id/submit` - Submit quiz

### Billing
- `GET /api/v1/billing/invoices` - List invoices
- `GET /api/v1/billing/invoices/:id` - Get invoice

See full API documentation at `/docs` when server is running.

## Performance Targets

| Metric | Target |
|--------|--------|
| API response time (p95) | < 300ms (reads), < 600ms (writes) |
| Timetable generation | < 10 seconds per class |
| Report card generation | < 5 seconds per student |
| Notification delivery | < 3 seconds (in-app) |
| System availability | 99.9% uptime |

## Security

- All passwords hashed with bcrypt (cost 12)
- JWT access tokens expire after 15 minutes
- JWT refresh tokens expire after 30 days
- All data encrypted at rest (PostgreSQL)
- All data encrypted in transit (TLS 1.2+)
- Rate limiting: 1000 req/min per tenant
- CORS enabled with configurable origins
- SQL injection prevention via prepared statements
- XSS prevention via input validation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT

## Support

For issues, questions, or contributions, please open an issue on GitHub.

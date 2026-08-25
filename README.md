# Izone-hub Talent Backend

**Izone-hub Talent** is a comprehensive talent management platform built with Go, designed to connect job seekers with employers through intelligent matching and skill-based assessments.

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- PostgreSQL 13+
- Docker & Docker Compose (for easy sandbox setup)

### 1. Sandbox Setup (Recommended)

Build custom sandbox Docker images (Go, TypeScript, Flutter):
```bash
docker build -t sandbox-go:1.25 -f templates/Dockerfile.sandbox-go .
docker build -t sandbox-node:22 -f templates/Dockerfile.sandbox-node .
docker build -t sandbox-flutter:3.27 -f templates/Dockerfile.sandbox-flutter .
```

Vendor node_modules into the Node-based framework templates (required — sandbox
containers run with `--network none`, so `npm install` inside them cannot reach
the registry):
```bash
./templates/vendor-node-modules.sh
```

### Supported Sandbox Languages & Frameworks

**Languages (`type: standard`)** — Python, JavaScript, TypeScript, Go, Java,
C++, C, Rust, Ruby, **Dart**, **SQL (SQLite)**.

- **Dart**: runs with the official `dart:stable` image; function-mode tests use
  a generated harness + `dart:mirrors`.
- **SQL (SQLite)**: executes `.sql` against an in-memory SQLite DB via a bundled
  Python runner. If `schema.sql` / `seed.sql` are present in the template they
  are applied first. Function mode compares query result rows against
  `expected_rows` (or `expected`) per test case:
  `[{"query": "SELECT ...", "expected": [[1], [2]]}]`.

**Frameworks (`type: framework`)** — React, Vue, Svelte, Node.js/Express,
Next.js, Flutter. Framework execution copies `templates/<framework>/<template_id>`
(default: `default`) and runs the framework test suite:

| Framework | Test command | Notes |
|---|---|---|
| React | `npm test` (Vitest + Testing Library) | deps vendored by script above |
| Vue | `npm test` (Vitest + @vue/test-utils) | |
| Svelte | `npm test` (Vitest + Testing Library Svelte) | |
| Express | `npm test` (Vitest + Supertest) | |
| Next.js | `npm test` (Vitest + Testing Library) | component tests only; no SSR/dev server offline |
| Flutter | `flutter pub get --offline; flutter test` | uses `sandbox-flutter:3.27`; template ships only SDK deps so it works fully offline |

### Start a local PostgreSQL and load sample data:

```bash
# Start PostgreSQL
docker compose up -d

# The seed.sql is auto-loaded on first start.
# It creates sample users, jobs, tags, and 15 quiz questions.

# Run the server
go run .
```

### 2. Manual Setup

```bash
# Configure your .env file with database credentials
cp .env.example .env
# Edit .env with your PostgreSQL connection details

# Run migrations
go run github.com/pressly/goose/v3/cmd/goose -dir sql/schema postgres "$DATABASE_URL" up

# Load sample data (optional)
psql "$DATABASE_URL" -f scripts/seed.sql

# Run the application
go run .
```

### 3. API Documentation

- **Health Check**: http://localhost:5000

## 📂 Project Structure

```
talent-backend/
├── config/         # Configuration management
├── controller/     # HTTP request handlers
├── database/       # Database connection and SQLC queries
├── middleware/     # Authentication and request middleware
├── models/         # Data structures and validation
├── router/         # HTTP route definitions
├── service/        # Business logic
├── sql/            # SQL migration files
├── sqlc.yml        # SQLC configuration
├── go.mod          # Go module dependencies
└── main.go         # Application entry point
```

## 🛠️ Tech Stack

- **Language**: Go 1.22+
- **Database**: PostgreSQL
- **ORM/Query Builder**: sqlc (type-safe SQL)
- **Authentication**: JWT, OAuth2 (GitHub)
- **Validation**: go-playground/validator
- **Logging**: zap
- **Testing**: testify

## 🧪 Testing

Run all tests with:

```bash
go test ./...
```

## 🔐 Authentication

### Authentication Flow

1. User redirects to GitHub for OAuth
2. GitHub authenticates and redirects back to `/api/v1/auth/github/callback`
3. Server exchanges code for access token
4. JWT token is issued and returned to client

### JWT Structure

```json
{
  "user_id": "uuid",
  "github_id": 12345,
  "github_username": "username",
  "role": "user|admin",
  "exp": 1234567890
}
```

## 📁 Important Files

- **`main.go`**: Application entry point and server setup
- **`config/config.go`**: Environment variable configuration
- **`database/db.go`**: Database connection and SQLC client
- **`sql/schema.sql`**: Database schema
- **`sql/queries.sql`**: SQL queries for sqlc
- **`router/router.go`**: Main router configuration
- **`router/v1_routes.go`**: V1 API routes
- **`controller/auth_controller.go`**: Authentication handlers
- **`controller/job_controller.go`**: Job management handlers
- **`controller/cv_controller.go`**: CV management handlers
- **`controller/tag_controller.go`**: Tag management handlers
- **`controller/question_controller.go`**: Question management handlers

## 📝 API Endpoints

### Auth
- `GET /api/v1/auth/github/login` - Start GitHub OAuth flow
- `GET /api/v1/auth/github/callback` - Handle GitHub callback
- `GET /api/v1/auth/me` - Get current user profile

### Jobs
- `GET /api/v1/jobs` - List all published jobs
- `GET /api/v1/jobs/{id}` - Get job details
- `POST /api/v1/jobs` - Create a new job (admin)
- `PUT /api/v1/jobs/{id}` - Update job (admin)
- `PATCH /api/v1/jobs/{id}/publish` - Publish job (admin)
- `PATCH /api/v1/jobs/{id}/close` - Close job (admin)
- `PATCH /api/v1/jobs/{id}/archive` - Archive job (admin)

### CV
- `POST /api/v1/cv/upload` - Upload CV
- `GET /api/v1/cv/current` - Get current CV
- `GET /api/v1/cv/versions` - List CV versions
- `GET /api/v1/cv/{id}/download` - Download CV
- `DELETE /api/v1/cv/{id}` - Delete CV

### Tags
- `GET /api/v1/tags` - List all tags
- `POST /api/v1/tags` - Create tag (admin)
- `GET /api/v1/tags/{id}` - Get tag details
- `PUT /api/v1/tags/{id}` - Update tag (admin)
- `DELETE /api/v1/tags/{id}` - Delete tag (admin)
- `POST /api/v1/tags/assign` - Assign tag to job (admin)
- `POST /api/v1/tags/remove` - Remove tag from job (admin)
- `GET /api/v1/tags/{id}/jobs` - Get jobs by tag
- `GET /api/v1/jobs/{id}/tags` - Get tags by job

### Questions
- `GET /api/v1/questions` - List all questions
- `GET /api/v1/questions/{id}` - Get question details
- `POST /api/v1/questions` - Create question (admin)
- `PUT /api/v1/questions/{id}` - Update question (admin)
- `DELETE /api/v1/questions/{id}` - Delete question (admin)

### Quiz (Skill Assessment)
- `GET /api/v1/quizzes` - List my quiz attempts
- `GET /api/v1/quizzes/{id}` - Get quiz attempt details
- `POST /api/v1/quizzes/{id}/start` - Start a new quiz (10 questions)
- `GET /api/v1/quizzes/{id}/question` - Get next random question
- `POST /api/v1/quizzes/{id}/answer` - Save answer for current question
- `POST /api/v1/quizzes/{id}/submit` - Submit and score the quiz

Each quiz picks **10 random questions** from the database. Questions are served one at a time in random order. After all 10 are answered, the quiz is scored and the application status is updated.

## 🔐 Role-Based Access Control

### User Role
- Can create and manage their own CV
- Can view published jobs
- Can apply for jobs
- Can view questions

### Admin Role
- All user permissions
- Can create, update, and delete jobs
- Can publish, close, and archive jobs
- Can create and manage tags
- Can create, update, and delete questions
- Can manage users (future)

## 🧪 Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific test file
go test ./controller/auth_controller_test.go

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests use a real PostgreSQL database connection.

```bash
# Run integration tests
go test ./integration_test.go
```

## ⚙️ Configuration

Create a `.env` file in the root directory:

```env
# Database configuration
DATABASE_URL=postgres://user:password@localhost:5432/talentdb?sslmode=disable

# GitHub OAuth
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret
GITHUB_CALLBACK_URL=http://localhost:5000/api/v1/auth/github/callback

# JWT configuration
JWT_SECRET=your_secret_key
JWT_TTL=24h

# Server configuration
PORT=5000
```

## 📂 File Uploads

CV files are uploaded to the `uploads/` directory.

## 🤝 Contributing

1. Create a feature branch
2. Make your changes
3. Run `sqlc generate` if you modified SQL files
4. Run `go test ./...` to ensure tests pass
5. Submit a pull request

## 📝 License

MIT License

## 📞 Support

For issues or questions, please open an issue on the GitHub repository.

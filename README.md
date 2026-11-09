# Go User Management REST API

A beginner-friendly portfolio project using Go's `net/http` with the Chi router, PostgreSQL, `pgx`, bcrypt password hashing, and JWT authentication.

## Prerequisites
- Go 1.22+
- Docker Desktop (or Docker Engine + Compose)
- curl or Postman

## 1. Start PostgreSQL
```bash
docker compose up -d
docker compose ps
```

## 2. Create the table
Linux/macOS/Git Bash:
```bash
docker exec -i user-management-postgres psql -U appuser -d userdb < migrations/001_create_users.sql
```
PowerShell:
```powershell
Get-Content .\migrations\001_create_users.sql | docker exec -i user-management-postgres psql -U appuser -d userdb
```

## 3. Configure environment variables
Linux/macOS/Git Bash:
```bash
export DATABASE_URL='postgres://appuser:apppassword@localhost:5432/userdb?sslmode=disable'
export JWT_SECRET='replace-with-a-random-secret-at-least-32-characters'
export PORT=8080
```
PowerShell:
```powershell
$env:DATABASE_URL='postgres://appuser:apppassword@localhost:5432/userdb?sslmode=disable'
$env:JWT_SECRET='replace-with-a-random-secret-at-least-32-characters'
$env:PORT='8080'
```
For local learning only, the database password is intentionally simple. Never use these credentials in production.

## 4. Run the API
```bash
go mod tidy
go run .
```
API: http://localhost:8080

## 5. Try the endpoints
Health:
```bash
curl http://localhost:8080/health
```
Create user:
```bash
curl -i -X POST http://localhost:8080/users -H 'Content-Type: application/json' -d '{"name":"Ravi","email":"ravi@example.com","password":"StrongPassword123"}'
```
List users:
```bash
curl 'http://localhost:8080/users?limit=20&offset=0'
```
Get user:
```bash
curl http://localhost:8080/users/1
```
Update user:
```bash
curl -i -X PUT http://localhost:8080/users/1 -H 'Content-Type: application/json' -d '{"name":"Ravi Kumar"}'
```
Login:
```bash
curl -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"ravi@example.com","password":"StrongPassword123"}'
```
Copy the token from the login response:
```bash
curl http://localhost:8080/me -H 'Authorization: Bearer YOUR_TOKEN_HERE'
```
Delete:
```bash
curl -i -X DELETE http://localhost:8080/users/1
```

## 6. Test and format
```bash
gofmt -w .
go test ./...
go vet ./...
```

## How the code is organized
This starter keeps the implementation in `main.go` so a beginner can follow the flow in one place. As a portfolio improvement, split it into:
- `internal/handler`: HTTP request/response logic
- `internal/service`: business rules
- `internal/repository`: SQL access
- `internal/model`: domain types
- `cmd/api`: application startup

## Important learning notes
- SQL values use `$1`, `$2` placeholders; never concatenate user input into SQL.
- Password hashes are never returned by API responses.
- JWT expires after one hour. Keep `JWT_SECRET` private and use a cryptographically random secret.
- This starter protects `/me`, but public CRUD endpoints are intentionally open for learning. Before production, add authorization so users can only read/update/delete permitted accounts, plus rate limiting, stronger validation, migrations tooling, TLS, and secret management.
- Email validation here is intentionally basic. Use a dedicated validation library or robust validation logic for production.
- `GET /users` uses limit/offset pagination; for very large tables, consider cursor pagination.
- Don't commit `.env` files or real secrets to GitHub.

## Portfolio improvement checklist
- [ ] Add repository/service/handler package separation
- [ ] Add integration tests using a test database
- [ ] Add structured logging and graceful shutdown
- [ ] Add role-based authorization
- [ ] Add CI with `go test ./...` and `go vet ./...`
- [ ] Add API documentation (OpenAPI/Swagger)
- [ ] Deploy the API and document the URL in this README

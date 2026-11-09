# Go User Management REST API

A beginner-friendly portfolio project built using Go, `net/http`, Chi router, PostgreSQL, pgx, bcrypt password hashing, JWT authentication, Docker, and Kubernetes.

This project demonstrates REST API development, database integration, authentication, containerization, and Kubernetes deployment.

## Tech Stack

- **Language:** Go 1.22+
- **HTTP Router:** Chi
- **Database:** PostgreSQL
- **Database Driver:** pgx
- **Authentication:** JWT
- **Password Hashing:** bcrypt
- **Containerization:** Docker, Docker Compose
- **Orchestration:** Kubernetes
- **Testing:** Go testing package

## Prerequisites

- Go 1.22+
- Docker Desktop (or Docker Engine + Compose)
- Kubernetes enabled in Docker Desktop
- kubectl
- curl or Postman

## Project Structure

```text
go-user-management-api-chi/
│
├── main.go
├── main_test.go
├── go.mod
├── go.sum
├── Dockerfile
├── compose.yaml
├── .dockerignore
├── .gitignore
├── .env.example
├── README.md
│
├── migrations/
│   └── 001_create_users.sql
│
└── k8s/
    ├── api-deployment.yaml
    ├── api-service.yaml
    └── postgres-deployment.yaml
```

## 1. Start PostgreSQL Using Docker Compose

Start the PostgreSQL container:

```bash
docker compose up -d
```

Check running containers:

```bash
docker compose ps
```

## 2. Create the Database Table

Execute the SQL migration to create the users table.

**Linux / macOS / Git Bash:**

```bash
docker exec -i user-management-postgres psql -U appuser -d userdb < migrations/001_create_users.sql
```

**Windows PowerShell:**

```powershell
Get-Content .\migrations\001_create_users.sql | docker exec -i user-management-postgres psql -U appuser -d userdb
```

## 3. Configure Environment Variables

Create a `.env` file in the project root.

Configure the following variables:

```env
PORT=8080
DATABASE_URL=postgres://appuser:your_password@localhost:5432/userdb?sslmode=disable
JWT_SECRET=your_random_secret_at_least_32_characters
POSTGRES_PASSWORD=your_password
```

Replace the example values with your own local credentials.

**Linux / macOS / Git Bash:**

```bash
export DATABASE_URL='postgres://appuser:your_password@localhost:5432/userdb?sslmode=disable'
export JWT_SECRET='your_random_secret_at_least_32_characters'
export PORT=8080
```

**Windows PowerShell:**

```powershell
$env:DATABASE_URL='postgres://appuser:your_password@localhost:5432/userdb?sslmode=disable'
$env:JWT_SECRET='your_random_secret_at_least_32_characters'
$env:PORT='8080'
```

Note:
- Use the PostgreSQL host port configured in `compose.yaml`.
- The example above assumes the host port is `5432`.
- Never commit `.env` or real credentials to GitHub.

## 4. Run the Go API Locally

Download dependencies:

```bash
go mod tidy
```

Run the application:

```bash
go run .
```

The API will be available at:

```text
http://localhost:8080
```

## 5. API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/users` | Create a user |
| GET | `/users` | List users |
| GET | `/users/{id}` | Get user by ID |
| PUT | `/users/{id}` | Update user |
| DELETE | `/users/{id}` | Delete user |
| POST | `/auth/login` | User login |
| GET | `/me` | Get authenticated user |

### Health Check

```bash
curl http://localhost:8080/health
```

### Create User

```bash
curl -i -X POST http://localhost:8080/users \
-H 'Content-Type: application/json' \
-d '{"name":"Ravi","email":"ravi@example.com","password":"StrongPassword123"}'
```

### List Users

```bash
curl 'http://localhost:8080/users?limit=20&offset=0'
```

### Get User

```bash
curl http://localhost:8080/users/1
```

### Update User

```bash
curl -i -X PUT http://localhost:8080/users/1 \
-H 'Content-Type: application/json' \
-d '{"name":"Ravi Kumar"}'
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
-H 'Content-Type: application/json' \
-d '{"email":"ravi@example.com","password":"StrongPassword123"}'
```

Copy the JWT token from the login response.

### Access Protected Endpoint

```bash
curl http://localhost:8080/me \
-H 'Authorization: Bearer YOUR_TOKEN_HERE'
```

### Delete User

```bash
curl -i -X DELETE http://localhost:8080/users/1
```

## 6. Docker Deployment

This project uses a multi-stage Docker build to create a lightweight production image.

### Build Docker Image

```bash
docker build -t user-management-api:v1 .
```

### Run Using Docker Compose

```bash
docker compose up -d --build
```

### Check Running Containers

```bash
docker compose ps
```

### View Container Logs

```bash
docker compose logs -f
```

The Dockerfile uses:
- Go Alpine image for building the application.
- Multi-stage build to reduce the final image size.
- Alpine runtime image.
- Non-root user for better container security.
- Compiled Go binary for application execution.

## 7. Kubernetes Deployment

This project includes Kubernetes configurations to deploy the Go API and PostgreSQL.

Kubernetes manages application deployment, replicas, services, and persistent storage.

### Kubernetes Architecture

```text
                 Client
                   |
                   |
             NodePort :30080
                   |
                   |
          Kubernetes Service
                   |
             Port :8080
                   |
          -------------------
          |                 |
       API Pod 1         API Pod 2
       Go + Chi          Go + Chi
          |                 |
          ---------+---------
                   |
             postgres:5432
                   |
             PostgreSQL Pod
                   |
             PersistentVolume
                   |
              Database Data
```

### Kubernetes Configuration Files

```text
k8s/
├── api-deployment.yaml
├── api-service.yaml
└── postgres-deployment.yaml
```

### Step 1: Enable Kubernetes

Enable Kubernetes in Docker Desktop.

Verify the cluster:

```powershell
kubectl get nodes
```

### Step 2: Build the Docker Image

Build the Go API image:

```powershell
docker build -t user-management-api:v1 .
```

The image is used by the local Docker Desktop Kubernetes cluster.

### Step 3: Create Kubernetes Secret

Create a local `.env` file containing the required environment variables.

The file must contain at least:

```env
POSTGRES_PASSWORD=your_password
JWT_SECRET=your_random_secret
```

Create the Kubernetes Secret:

```powershell
kubectl create secret generic user-management-secret --from-env-file=.env
```

The Secret provides sensitive configuration to the API and PostgreSQL containers.

Do not commit `.env` or real Secret manifests to GitHub.

### Step 4: Deploy PostgreSQL

Apply the PostgreSQL configuration:

```powershell
kubectl apply -f k8s/postgres-deployment.yaml
```

This creates:

- PostgreSQL Deployment
- PostgreSQL Service
- PersistentVolumeClaim

### Step 5: Deploy Go API

Apply the API Deployment:

```powershell
kubectl apply -f k8s/api-deployment.yaml
```

The API Deployment runs two replicas.

### Step 6: Expose the Go API

Apply the API Service:

```powershell
kubectl apply -f k8s/api-service.yaml
```

The NodePort Service exposes the API on port `30080`.

### Step 7: Verify Kubernetes Resources

Check Pods:

```powershell
kubectl get pods
```

Check Deployments:

```powershell
kubectl get deployments
```

Check Services:

```powershell
kubectl get services
```

Check PersistentVolumeClaims:

```powershell
kubectl get pvc
```

### Step 8: Access the API

The API is accessible through:

```text
http://localhost:30080
```

Test the health endpoint:

```text
http://localhost:30080/health
```

Expected response:

```json
{
  "status": "ok"
}
```

### Useful Kubernetes Commands

View API logs:

```powershell
kubectl logs deployment/user-management-api
```

View PostgreSQL logs:

```powershell
kubectl logs deployment/postgres
```

Describe API Deployment:

```powershell
kubectl describe deployment user-management-api
```

Delete API Deployment:

```powershell
kubectl delete -f k8s/api-deployment.yaml
```

Delete API Service:

```powershell
kubectl delete -f k8s/api-service.yaml
```

## 8. Testing and Code Formatting

Format Go code:

```bash
gofmt -w .
```

Run tests:

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

Run Go static analysis:

```bash
go vet ./...
```

## 9. How the Code Is Organized

The starter keeps the implementation in `main.go` so beginners can follow the application flow in one place.

As a future architectural improvement, split the application into:

```text
internal/
├── handler/
├── service/
├── repository/
└── model/

cmd/
└── api/
```

Responsibilities:

- `handler`: HTTP request and response handling.
- `service`: Business logic.
- `repository`: Database operations.
- `model`: Application data structures.
- `cmd/api`: Application startup.

## 10. Important Learning Notes

- SQL queries use `$1`, `$2` placeholders to prevent SQL injection.
- Passwords are hashed using bcrypt and password hashes are never returned in API responses.
- JWT tokens expire after one hour.
- Keep `JWT_SECRET` private and use a cryptographically random secret.
- The `/me` endpoint is protected using JWT authentication.
- Public CRUD endpoints are intentionally open for learning.
- Before production, add authorization so users can only access permitted accounts.
- Email validation is basic and should be improved for production.
- `GET /users` supports limit/offset pagination.
- For very large datasets, consider cursor-based pagination.
- Kubernetes Secrets provide configuration to Pods, but base64 encoding alone is not encryption.
- The Kubernetes setup in this project is intended for local learning using Docker Desktop.

## 11. Portfolio Improvement Checklist

- [x] Build Go REST API using Chi router
- [x] Integrate PostgreSQL using pgx
- [x] Implement CRUD operations
- [x] Implement JWT authentication
- [x] Hash passwords using bcrypt
- [x] Build Docker multi-stage image
- [x] Configure Docker Compose
- [x] Deploy Go API using Kubernetes
- [x] Deploy PostgreSQL in Kubernetes
- [x] Configure Kubernetes Secrets
- [x] Configure Kubernetes Services
- [x] Configure PersistentVolumeClaim
- [ ] Add repository/service/handler package separation
- [ ] Add integration tests using a test database
- [ ] Add structured logging and graceful shutdown
- [ ] Add role-based authorization
- [ ] Add GitHub Actions CI/CD pipeline
- [ ] Add API documentation using OpenAPI/Swagger
- [ ] Deploy to a cloud Kubernetes environment

## Author

Go Backend Developer | Go | REST APIs | PostgreSQL | Docker | Kubernetes
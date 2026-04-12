# Local & Docker Setup Guide

## ✅ Option 1: Run Locally (Go app on your machine + Docker services)

### Prerequisites

- Go 1.20+
- Docker & Docker Compose (for services only)

### Setup

**1. Install Go dependencies:**

```bash
go mod tidy
```

**2. Start only the infrastructure services in Docker:**

```bash
# Recommended: Use the local-only compose file (no goapp service)
docker-compose -f docker-compose.local.yml up

# OR: Start specific services from main compose file
docker-compose up postgres redis kafka zookeeper
```

**Why?** Services run in Docker but expose ports to `localhost`, so your local Go app connects via `localhost:5432`, `localhost:6379`, etc.

**3. Run the application locally (new terminal):**

```bash
# This loads .env.local which has localhost connections
go run ./cmd/server
```

**4. Access endpoints:**

- HTTP API: `http://localhost:8080`
- gRPC: `localhost:50051`
- Redis Insight: `http://localhost:5540`

### ✅ How env files work in this setup:

- `docker-compose up postgres redis...` skips the `goapp` service, so `env_file: ./.env` doesn't apply
- Your local Go app independently loads `.env.local` with `localhost` connections
- **Result:** Local app → localhost:5432 (dockerized PostgreSQL) ✅

---

## 🐳 Option 2: Run Everything in Docker (Full Stack)

### Prerequisites

- Docker & Docker Compose

### Setup

**1. Build and start everything:**

```bash
# Starts postgres, redis, kafka, AND the goapp service in Docker
docker-compose up --build
```

**Why?**

- The `goapp` service is started (skipped in Option 1)
- Container loads `.env` via `env_file: ./.env`
- Services use Docker network: `postgres:5432`, `redis:6379`, `kafka:9092`

**2. Access endpoints:**

- HTTP API: `http://localhost:8080`
- gRPC: `localhost:50051`
- Redis Insight: `http://localhost:5540`

---

## 🔄 Environment Variables Explained

### `.env.local` — Local Development

```
ENV=development
DB_HOST=localhost
REDIS_HOST=localhost
KAFKA_HOST=localhost
BACKEND_HTTP_PORT=8080
BACKEND_GRPC_PORT=50051
```

**Used by:** Go app running on your machine  
**Loaded when:** Running `go run ./cmd/server` locally

### `.env` — Docker/Production

```
ENV=production
DB_HOST=postgres
REDIS_HOST=redis
KAFKA_HOST=kafka
BACKEND_HTTP_PORT=8080
BACKEND_GRPC_PORT=50051
```

**Used by:** Go app running inside Docker container  
**Loaded by:** docker-compose.yaml's `env_file:` section for `goapp` service

---

## 📊 Quick Reference

| Scenario     | Services  | Go App    | Config       | Connection       |
| ------------ | --------- | --------- | ------------ | ---------------- |
| **Option 1** | 🐳 Docker | 🏃 Local  | `.env.local` | `localhost:5432` |
| **Option 2** | 🐳 Docker | 🐳 Docker | `.env`       | `postgres:5432`  |

---

## ✨ What Changed

1. **Created `docker-compose.local.yml`** — Separate compose file for local dev (no goapp service)
2. **Fixed server startup** — Both gRPC and HTTP servers run concurrently
3. **Graceful shutdown** — Proper SIGINT/SIGTERM handling
4. **Environment isolation** — `.env.local` excluded from Docker builds via `.dockerignore`
5. **Conditional loading** — Config auto-detects environment

---

## 🚀 Troubleshooting

### Local Development: "connection refused"

```bash
# Ensure services are running
docker-compose -f docker-compose.local.yml logs

# Check .env.local has localhost (not service names)
cat .env.local
```

### Docker Full Stack: "connection refused"

```bash
# Check if goapp is running
docker-compose ps

# View logs
docker-compose logs goapp

# Verify .env has service names (not localhost)
cat .env
```

### "port already in use"

```bash
# Stop all containers
docker-compose down
docker-compose -f docker-compose.local.yml down

# Free specific port (macOS)
lsof -ti:5432 | xargs kill -9
```

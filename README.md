# TaxiHub

A Go microservices-based taxi management system.

## Project Structure

```
taxihub/
├── driver-service/
│ ├── cmd/
│ │ └── main.go
│ ├── internal/
│ │ ├── handlers/
│ │ ├── models/
│ │ ├── repository/
│ │ ├── service/
│ │ └── config/
│ ├── go.mod
│ ├── go.sum
│ └── Dockerfile  
│
├── docker-compose.yml
└── README.md
```

## Services

### Driver Service (Port 8081)

- Manages driver profiles
- Handles driver location updates
- Manages driver availability status

## Technology Stack

- **Framework**: Fiber (Go web framework)
- **Containerization**: Docker & Docker Compose
- **Architecture**: Microservices

## Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose

### Running Locally

1. **Install dependencies for each service:**

   ```bash
   cd api-gateway
   go mod tidy

   cd ../driver-service
   go mod tidy
   ```

2. **Run services individually:**

   ```bash
   # API Gateway
   cd api-gateway
   go run cmd/main.go

   # Driver Service
   cd driver-service
   go run cmd/main.go
   ```

3. **Or run with Docker Compose:**d
   ```bash
   docker-compose up --build
   ```

### Health Check

- Driver Service: http://localhost:8081/health

## API Endpoints

### Driver Service

- `GET /health` - Health check endpoint

# Insider Backend - Financial Transaction System

A comprehensive Go-based financial transaction system with event sourcing, concurrent processing, and high availability features.

## Features

### Core Features
- **User Management**: Registration, authentication, role-based authorization
- **Transaction Processing**: Credit, debit, and transfer operations with atomic processing
- **Balance Management**: Real-time balance tracking with history
- **Audit Logging**: Comprehensive audit trail for all operations
- **Concurrent Processing**: Worker pool-based transaction processing
- **Event Sourcing**: Complete event history with replay capabilities
- **Caching**: Redis-based caching for improved performance

### Technical Features
- **HTTP API**: RESTful API with comprehensive endpoints
- **Database**: PostgreSQL with optimized schemas and indices
- **Authentication**: JWT-based authentication with refresh tokens
- **Middleware**: Rate limiting, CORS, security headers, logging
- **Monitoring**: Prometheus metrics and health checks
- **Graceful Shutdown**: Proper resource cleanup on shutdown
- **Docker Support**: Containerized deployment with docker-compose

## Architecture

The system follows Domain-Driven Design (DDD) principles with clean architecture:

```
├── cmd/                    # Application entry points
├── internal/               # Private application code
│   ├── domain/            # Domain models and business logic
│   ├── repository/        # Data access layer
│   ├── service/           # Business services
│   ├── handler/           # HTTP handlers
│   ├── middleware/        # HTTP middleware
│   ├── worker/            # Background job processing
│   ├── event/             # Event sourcing components
│   ├── metrics/           # Monitoring and metrics
│   ├── config/            # Configuration management
│   └── server/            # HTTP server setup
├── pkg/                   # Public reusable packages
├── migrations/            # Database migrations
├── deployments/           # Deployment configurations
└── scripts/               # Utility scripts
```

## Quick Start

### Prerequisites
- Go 1.21+
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone <repository-url>
cd insiderBackendPath
```

2. Start the services:
```bash
docker-compose up -d
```

3. Check service health:
```bash
curl http://localhost:8080/api/v1/health
```

### Manual Setup

1. Install dependencies:
```bash
go mod download
```

2. Set up environment variables:
```bash
cp env.example .env
# Edit .env with your configuration
```

3. Run database migrations:
```bash
# Install migrate tool first
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgres://postgres:password@localhost:5432/insider_backend?sslmode=disable" up
```

4. Start the application:
```bash
go run cmd/server/main.go
```

## API Documentation

### Authentication Endpoints

#### Register User
```http
POST /api/v1/auth/register
Content-Type: application/json

{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "secure_password",
    "role": "user"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
    "username": "john_doe",
    "password": "secure_password"
}
```

### User Management Endpoints

#### Get Current User
```http
GET /api/v1/users/me
Authorization: Bearer <access_token>
```

#### Update User
```http
PUT /api/v1/users/{id}
Authorization: Bearer <access_token>
Content-Type: application/json

{
    "username": "new_username",
    "email": "new_email@example.com"
}
```

#### List Users (Admin Only)
```http
GET /api/v1/users?limit=20&offset=0
Authorization: Bearer <access_token>
```

### Transaction Endpoints

#### Create Credit Transaction
```http
POST /api/v1/transactions/credit
Authorization: Bearer <access_token>
Content-Type: application/json

{
    "to_user_id": "123e4567-e89b-12d3-a456-426614174000",
    "amount": 100.50,
    "description": "Account credit",
    "reference_id": "REF001"
}
```

#### Create Debit Transaction
```http
POST /api/v1/transactions/debit
Authorization: Bearer <access_token>
Content-Type: application/json

{
    "from_user_id": "123e4567-e89b-12d3-a456-426614174000",
    "amount": 50.25,
    "description": "Account debit",
    "reference_id": "REF002"
}
```

#### Create Transfer Transaction
```http
POST /api/v1/transactions/transfer
Authorization: Bearer <access_token>
Content-Type: application/json

{
    "from_user_id": "123e4567-e89b-12d3-a456-426614174000",
    "to_user_id": "987fcdeb-51d2-43a1-b456-426614174000",
    "amount": 25.00,
    "description": "Transfer to friend",
    "reference_id": "REF003"
}
```

#### Get Transaction History
```http
GET /api/v1/transactions/history?limit=20&offset=0&type=transfer&status=completed
Authorization: Bearer <access_token>
```

### Balance Endpoints

#### Get Current Balance
```http
GET /api/v1/balances/current
Authorization: Bearer <access_token>
```

#### Get Balance History
```http
GET /api/v1/balances/historical?limit=20&offset=0
Authorization: Bearer <access_token>
```

#### Get Balance at Specific Time
```http
GET /api/v1/balances/at-time?timestamp=2023-12-01T12:00:00Z
Authorization: Bearer <access_token>
```

## Configuration

The application can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server host | `0.0.0.0` |
| `SERVER_PORT` | Server port | `8080` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `password` |
| `DB_NAME` | Database name | `insider_backend` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `JWT_SECRET` | JWT secret key | `your-secret-key` |
| `LOG_LEVEL` | Log level | `info` |

## Development

### Running Tests
```bash
go test ./...
```

### Database Migrations
```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Apply migrations
migrate -path migrations -database "postgres://..." up

# Rollback migrations
migrate -path migrations -database "postgres://..." down 1
```

### Building
```bash
# Build for current platform
go build -o bin/server cmd/server/main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o bin/server-linux cmd/server/main.go
```

## Monitoring

### Health Check
```http
GET /api/v1/health
```

### Metrics (Prometheus)
```http
GET /metrics
```

### Grafana Dashboard
Access Grafana at `http://localhost:3000` (admin/admin) when using docker-compose.

## Deployment

### Docker
```bash
# Build image
docker build -t insider-backend .

# Run container
docker run -p 8080:8080 insider-backend
```

### Docker Compose
```bash
# Development
docker-compose up -d

# Production
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## Security Considerations

- Change default JWT secret in production
- Use strong passwords for database and Redis
- Enable SSL/TLS for production deployments
- Configure proper firewall rules
- Regular security updates

## Performance

The system is designed for high performance with:
- Connection pooling for database connections
- Redis caching for frequently accessed data
- Concurrent transaction processing with worker pools
- Optimized database queries with proper indices
- Rate limiting to prevent abuse

## Troubleshooting

### Common Issues

1. **Database connection failed**
   - Check database credentials and connectivity
   - Ensure PostgreSQL is running and accessible

2. **Redis connection failed**
   - Verify Redis is running and accessible
   - Check Redis configuration

3. **JWT token invalid**
   - Ensure JWT secret is correctly configured
   - Check token expiration settings

### Logs
Application logs are structured JSON format. Key log fields:
- `level`: Log level (debug, info, warn, error)
- `message`: Log message
- `timestamp`: UTC timestamp
- `request_id`: Unique request identifier

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

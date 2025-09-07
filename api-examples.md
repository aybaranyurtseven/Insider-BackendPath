# API Examples

This document provides comprehensive examples of using the Insider Backend API.

## Authentication

### Register a New User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "securepassword123"
  }'
```

Response:
```json
{
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "user",
    "created_at": "2023-12-01T10:00:00Z",
    "updated_at": "2023-12-01T10:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

## Transaction Operations

### Credit Account
```bash
curl -X POST http://localhost:8080/api/v1/transactions/credit \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "to_user_id": "123e4567-e89b-12d3-a456-426614174000",
    "amount": 100.50,
    "description": "Initial deposit",
    "reference_id": "DEP001"
  }'
```

### Debit Account
```bash
curl -X POST http://localhost:8080/api/v1/transactions/debit \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "from_user_id": "123e4567-e89b-12d3-a456-426614174000",
    "amount": 25.00,
    "description": "Service fee",
    "reference_id": "FEE001"
  }'
```

### Transfer Between Accounts
```bash
curl -X POST http://localhost:8080/api/v1/transactions/transfer \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "from_user_id": "123e4567-e89b-12d3-a456-426614174000",
    "to_user_id": "987fcdeb-51d2-43a1-b456-426614174000",
    "amount": 50.00,
    "description": "Payment to friend",
    "reference_id": "TXN001"
  }'
```

### Get Transaction History
```bash
# Get all transactions for current user
curl -X GET "http://localhost:8080/api/v1/transactions/history?limit=10&offset=0" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Filter by transaction type
curl -X GET "http://localhost:8080/api/v1/transactions/history?type=transfer&status=completed" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Filter by date range
curl -X GET "http://localhost:8080/api/v1/transactions/history?from_date=2023-12-01T00:00:00Z&to_date=2023-12-31T23:59:59Z" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Balance Operations

### Get Current Balance
```bash
curl -X GET http://localhost:8080/api/v1/balances/current \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Response:
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "amount": 125.50,
  "last_updated_at": "2023-12-01T12:30:00Z",
  "version": 5
}
```

### Get Balance History
```bash
curl -X GET "http://localhost:8080/api/v1/balances/historical?limit=20&offset=0" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Get Balance at Specific Time
```bash
curl -X GET "http://localhost:8080/api/v1/balances/at-time?timestamp=2023-12-01T12:00:00Z" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## User Management

### Get Current User Info
```bash
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Update User Profile
```bash
curl -X PUT http://localhost:8080/api/v1/users/123e4567-e89b-12d3-a456-426614174000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "username": "john_doe_updated",
    "email": "john.updated@example.com"
  }'
```

### List All Users (Admin Only)
```bash
curl -X GET "http://localhost:8080/api/v1/users?limit=50&offset=0" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

## Administrative Operations

### Get System Health
```bash
curl -X GET http://localhost:8080/api/v1/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2023-12-01T12:00:00Z",
  "version": "1.0.0",
  "worker_pool": {
    "jobs_processed": 1250,
    "jobs_successful": 1248,
    "jobs_failed": 2,
    "jobs_in_progress": 3
  }
}
```

### Get Metrics (Prometheus Format)
```bash
curl -X GET http://localhost:8080/metrics
```

## Error Handling

### Common Error Responses

#### 400 Bad Request
```json
{
  "error": "Invalid request body"
}
```

#### 401 Unauthorized
```json
{
  "error": "Invalid token"
}
```

#### 403 Forbidden
```json
{
  "error": "Insufficient permissions"
}
```

#### 404 Not Found
```json
{
  "error": "User not found"
}
```

#### 429 Too Many Requests
```json
{
  "error": "Rate limit exceeded"
}
```

#### 500 Internal Server Error
```json
{
  "error": "Internal Server Error"
}
```

## Environment Setup for Testing

### Using cURL with Authentication
```bash
# 1. Register a user
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "testpassword123"
  }')

# 2. Extract access token
ACCESS_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.access_token')

# 3. Use token for subsequent requests
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Using Postman

1. **Environment Variables:**
   - `base_url`: `http://localhost:8080`
   - `access_token`: (set after login)

2. **Collection Variables:**
   - Create a collection for all API endpoints
   - Set up pre-request scripts to handle authentication

3. **Example Pre-request Script:**
```javascript
// Auto-login if token is expired
if (!pm.globals.get("access_token")) {
    pm.sendRequest({
        url: pm.globals.get("base_url") + "/api/v1/auth/login",
        method: "POST",
        header: {
            "Content-Type": "application/json"
        },
        body: {
            mode: "raw",
            raw: JSON.stringify({
                username: "testuser",
                password: "testpassword123"
            })
        }
    }, function(err, response) {
        if (!err && response.code === 200) {
            const token = response.json().access_token;
            pm.globals.set("access_token", token);
        }
    });
}
```

## WebSocket Examples (if implemented)

### Real-time Balance Updates
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/balance');

ws.onopen = function() {
    // Send authentication
    ws.send(JSON.stringify({
        type: 'auth',
        token: 'YOUR_ACCESS_TOKEN'
    }));
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Balance update:', data);
};
```

## Rate Limiting

The API implements rate limiting:
- **General API**: 100 requests per minute per IP
- **Auth endpoints**: 5 requests per minute per IP

Headers returned:
- `X-RateLimit-Limit`: Request limit
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Reset time (Unix timestamp)

## Security Best Practices

1. **Always use HTTPS in production**
2. **Store tokens securely** (never in localStorage for web apps)
3. **Implement token refresh logic**
4. **Validate all input data**
5. **Use proper CORS settings**
6. **Monitor for suspicious activity**

## Integration Examples

### Node.js Integration
```javascript
const axios = require('axios');

class InsiderAPI {
    constructor(baseURL, accessToken) {
        this.client = axios.create({
            baseURL,
            headers: {
                'Authorization': `Bearer ${accessToken}`,
                'Content-Type': 'application/json'
            }
        });
    }

    async createTransfer(fromUserId, toUserId, amount, description) {
        const response = await this.client.post('/api/v1/transactions/transfer', {
            from_user_id: fromUserId,
            to_user_id: toUserId,
            amount,
            description
        });
        return response.data;
    }

    async getBalance() {
        const response = await this.client.get('/api/v1/balances/current');
        return response.data;
    }
}
```

### Python Integration
```python
import requests

class InsiderAPI:
    def __init__(self, base_url, access_token):
        self.base_url = base_url
        self.headers = {
            'Authorization': f'Bearer {access_token}',
            'Content-Type': 'application/json'
        }
    
    def create_credit(self, to_user_id, amount, description):
        url = f"{self.base_url}/api/v1/transactions/credit"
        data = {
            "to_user_id": to_user_id,
            "amount": amount,
            "description": description
        }
        response = requests.post(url, json=data, headers=self.headers)
        return response.json()
    
    def get_transaction_history(self, limit=20, offset=0):
        url = f"{self.base_url}/api/v1/transactions/history"
        params = {"limit": limit, "offset": offset}
        response = requests.get(url, params=params, headers=self.headers)
        return response.json()
```

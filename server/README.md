# MulaMail 2 Server - Setup & Running Guide

## Quick Start

```bash
# 1. Install dependencies
cd server
go mod download

# 2. Start MongoDB (Docker)
docker run -d -p 27017:27017 --name mulamail-mongo mongo:latest

# 3. Set required environment variables
export ENCRYPTION_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
export MONGO_URI="mongodb://localhost:27017"
export MONGO_DB="mulamail"

# 4. Run the server
go run main.go
```

The server will start on `http://localhost:8080`

## Prerequisites

### Required Services

1. **MongoDB** - For storing identities and mail accounts
2. **Solana RPC** - For blockchain transactions (defaults to mainnet)
3. **AWS S3** (optional for Phase 1) - For encrypted mail vault

### Required Tools

- Go 1.22 or later
- Docker (for MongoDB)
- AWS credentials (if using S3)

## Installation

### 1. Clone and Install Dependencies

```bash
cd server
go mod download
```

### 2. Set Up MongoDB

**Option A: Docker (Recommended)**

```bash
# Start MongoDB
docker run -d \
  --name mulamail-mongo \
  -p 27017:27017 \
  -v mulamail-data:/data/db \
  mongo:latest

# Verify it's running
docker ps | grep mulamail-mongo
```

**Option B: Local MongoDB**

```bash
# Install MongoDB (Ubuntu/Debian)
sudo apt-get install mongodb

# Start MongoDB
sudo systemctl start mongodb
```

**Option C: MongoDB Atlas (Cloud)**

```bash
# Get connection string from MongoDB Atlas
# Example: mongodb+srv://user:pass@cluster.mongodb.net/
```

### 3. Configure Environment Variables

Create a `.env` file or export variables:

```bash
# Server Configuration
export PORT="8080"                    # HTTP port (default: 8080)

# Database
export MONGO_URI="mongodb://localhost:27017"  # MongoDB connection string
export MONGO_DB="mulamail"                     # Database name

# Blockchain
export SOLANA_RPC="https://api.mainnet-beta.solana.com"  # Solana RPC endpoint
# For development, use devnet:
# export SOLANA_RPC="https://api.devnet.solana.com"

# Encryption (REQUIRED - generate a new key!)
export ENCRYPTION_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
# ⚠️ IMPORTANT: Generate a new 64-character hex key (32 bytes) for production!

# AWS S3 (optional for Phase 1)
export AWS_REGION="us-east-1"
export S3_BUCKET="mulamail-vault"
# AWS credentials from ~/.aws/credentials or environment
```

### 4. Generate Encryption Key

**IMPORTANT**: Generate a secure encryption key for production:

```bash
# Generate a secure 32-byte key (64 hex characters)
openssl rand -hex 32

# Set it as environment variable
export ENCRYPTION_KEY="$(openssl rand -hex 32)"
```

## Running the Server

### Development Mode

```bash
cd server

# Run with default settings
go run main.go

# Run with custom port
PORT=3000 go run main.go

# Run with devnet
SOLANA_RPC="https://api.devnet.solana.com" go run main.go
```

### Production Build

```bash
# Build binary
go build -o mulamail-server main.go

# Run binary
./mulamail-server
```

### Using Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o mulamail-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mulamail-server .
EXPOSE 8080
CMD ["./mulamail-server"]
```

Build and run:

```bash
# Build image
docker build -t mulamail-server .

# Run container
docker run -d \
  --name mulamail \
  -p 8080:8080 \
  -e MONGO_URI="mongodb://host.docker.internal:27017" \
  -e ENCRYPTION_KEY="your-key-here" \
  mulamail-server
```

### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:latest
    ports:
      - "27017:27017"
    volumes:
      - mulamail-data:/data/db

  server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - MONGO_URI=mongodb://mongodb:27017
      - MONGO_DB=mulamail
      - SOLANA_RPC=https://api.devnet.solana.com
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - AWS_REGION=us-east-1
      - S3_BUCKET=mulamail-vault
    depends_on:
      - mongodb

volumes:
  mulamail-data:
```

Run:

```bash
# Set encryption key
export ENCRYPTION_KEY=$(openssl rand -hex 32)

# Start services
docker-compose up -d

# View logs
docker-compose logs -f server
```

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `MONGO_URI` | Yes | `mongodb://localhost:27017` | MongoDB connection string |
| `MONGO_DB` | No | `mulamail` | MongoDB database name |
| `SOLANA_RPC` | No | `https://api.mainnet-beta.solana.com` | Solana RPC endpoint |
| `AWS_REGION` | No | `us-east-1` | AWS region for S3 |
| `S3_BUCKET` | No | `mulamail-vault` | S3 bucket name |
| `ENCRYPTION_KEY` | **Yes** | *(insecure default)* | 64-char hex key for AES-256-GCM |

### Solana RPC Endpoints

```bash
# Mainnet (production)
export SOLANA_RPC="https://api.mainnet-beta.solana.com"

# Devnet (development)
export SOLANA_RPC="https://api.devnet.solana.com"

# Testnet
export SOLANA_RPC="https://api.testnet.solana.com"

# Local validator
export SOLANA_RPC="http://localhost:8899"
```

## Verifying the Server

### Health Check

```bash
curl http://localhost:8080/api/health
# Expected: {"status":"ok"}
```

### Test API Endpoints

```bash
# Create identity transaction
curl -X POST http://localhost:8080/api/v1/identity/create-tx \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@mulamail.com",
    "pubkey": "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin"
  }'

# Resolve identity
curl "http://localhost:8080/api/v1/identity/resolve?email=test@mulamail.com"
```

## API Endpoints

### Health

- **GET** `/api/health` - Health check

### Identity Management

- **POST** `/api/v1/identity/create-tx` - Create unsigned identity transaction
- **POST** `/api/v1/identity/register` - Register identity on blockchain
- **GET** `/api/v1/identity/resolve` - Resolve identity by email or pubkey

### Mail Account Management

- **POST** `/api/v1/accounts` - Add mail account
- **GET** `/api/v1/accounts?owner=<pubkey>` - List accounts

### Mail Operations

- **GET** `/api/v1/mail/inbox?owner=<pubkey>&account=<email>` - Fetch inbox
- **GET** `/api/v1/mail/message?owner=<pubkey>&account=<email>&id=<msg-id>` - Get message
- **POST** `/api/v1/mail/send` - Send mail

See the [API documentation](../whitepaper.md) for detailed endpoint specifications.

## Troubleshooting

### MongoDB Connection Failed

**Error**: `MongoDB connect: connection refused`

**Solution**:
```bash
# Check if MongoDB is running
docker ps | grep mongo
# or
systemctl status mongodb

# Restart MongoDB
docker restart mulamail-mongo
# or
sudo systemctl restart mongodb
```

### Encryption Key Error

**Error**: `decode encryption key: encoding/hex: invalid byte`

**Solution**: Ensure `ENCRYPTION_KEY` is exactly 64 hexadecimal characters:
```bash
# Generate new key
export ENCRYPTION_KEY=$(openssl rand -hex 32)
```

### Solana RPC Timeout

**Error**: `get blockhash: timeout`

**Solutions**:
```bash
# Use devnet (faster for development)
export SOLANA_RPC="https://api.devnet.solana.com"

# Use a local validator
solana-test-validator
export SOLANA_RPC="http://localhost:8899"

# Use a premium RPC provider
export SOLANA_RPC="https://your-rpc-provider.com"
```

### Port Already in Use

**Error**: `bind: address already in use`

**Solution**:
```bash
# Use different port
PORT=3000 go run main.go

# Or kill the process using port 8080
lsof -ti:8080 | xargs kill -9
```

## Development

### Hot Reload

Install air for hot reloading:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### Debug Mode

```bash
# Run with verbose logging
go run main.go -v

# Enable race detector
go run -race main.go
```

## Testing

See [TEST_README.md](TEST_README.md) for comprehensive testing documentation.

```bash
# Run all tests
make test

# Run unit tests only (no MongoDB required)
make test-unit

# Run with coverage
make test-coverage
```

## Production Deployment

### Security Checklist

- [ ] Generate unique `ENCRYPTION_KEY` (never use default)
- [ ] Use HTTPS/TLS in production
- [ ] Set up MongoDB authentication
- [ ] Configure firewall rules
- [ ] Enable MongoDB replica set
- [ ] Set up monitoring and logging
- [ ] Configure rate limiting
- [ ] Use environment-specific Solana RPC
- [ ] Secure AWS credentials

### Recommended Architecture

```
Internet → Load Balancer (HTTPS) → MulaMail Server → MongoDB Replica Set
                                  ↓
                             Solana RPC
                                  ↓
                              AWS S3
```

### Process Manager (systemd)

Create `/etc/systemd/system/mulamail.service`:

```ini
[Unit]
Description=MulaMail Server
After=network.target

[Service]
Type=simple
User=mulamail
WorkingDirectory=/opt/mulamail
Environment="PORT=8080"
Environment="MONGO_URI=mongodb://localhost:27017"
Environment="ENCRYPTION_KEY=your-key-here"
ExecStart=/opt/mulamail/mulamail-server
Restart=always

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable mulamail
sudo systemctl start mulamail
sudo systemctl status mulamail
```

## Monitoring

### Logs

```bash
# View server logs (if using systemd)
sudo journalctl -u mulamail -f

# View Docker logs
docker logs -f mulamail

# View Docker Compose logs
docker-compose logs -f server
```

### Metrics

Add monitoring endpoints (example with Prometheus):

```bash
# Health check
curl http://localhost:8080/api/health

# Add custom metrics endpoint (Phase 2)
curl http://localhost:8080/metrics
```

## License

See main project LICENSE

## Support

For issues and questions:
- GitHub Issues: https://github.com/your-org/mulamail/issues
- Documentation: [whitepaper.md](../whitepaper.md)

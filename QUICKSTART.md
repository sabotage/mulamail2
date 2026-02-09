# MulaMail 2 - Quick Start Guide

Get MulaMail 2 server running in under 5 minutes!

## Prerequisites

- Go 1.22+ installed
- Docker (for MongoDB)
- Terminal access

## Quick Start (Interactive)

The easiest way to get started is using the interactive startup script:

```bash
cd server
./start.sh
```

The script will:
- ✅ Check prerequisites
- ✅ Start MongoDB if needed
- ✅ Generate encryption keys
- ✅ Configure environment
- ✅ Start the server

## Quick Start (Manual)

If you prefer manual setup:

```bash
# 1. Start MongoDB
docker run -d -p 27017:27017 --name mulamail-mongo mongo:latest

# 2. Set encryption key
export ENCRYPTION_KEY=$(openssl rand -hex 32)

# 3. Run server
cd server
go run main.go
```

Server starts on `http://localhost:8080`

## Verify It's Working

```bash
# Health check
curl http://localhost:8080/api/health
# Expected: {"status":"ok"}
```

## Test the API

### Create an Identity Transaction

```bash
curl -X POST http://localhost:8080/api/v1/identity/create-tx \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@mulamail.com",
    "pubkey": "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin"
  }'
```

### Add a Mail Account

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "owner_pubkey": "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
    "account_email": "user@gmail.com",
    "pop3": {
      "host": "pop.gmail.com",
      "port": 995,
      "user": "user@gmail.com",
      "pass": "your-password",
      "use_ssl": true
    },
    "smtp": {
      "host": "smtp.gmail.com",
      "port": 587,
      "user": "user@gmail.com",
      "pass": "your-password",
      "use_ssl": true
    }
  }'
```

### List Accounts

```bash
curl "http://localhost:8080/api/v1/accounts?owner=9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin"
```

## Configuration

### Environment Variables

Required:
- `ENCRYPTION_KEY` - 64-char hex key for AES-256-GCM

Optional:
- `PORT` - Server port (default: 8080)
- `MONGO_URI` - MongoDB connection (default: mongodb://localhost:27017)
- `MONGO_DB` - Database name (default: mulamail)
- `SOLANA_RPC` - Solana RPC endpoint (default: devnet)

### Generate Secure Encryption Key

```bash
# Generate and set
export ENCRYPTION_KEY=$(openssl rand -hex 32)

# Save for future use
echo "export ENCRYPTION_KEY=\"$ENCRYPTION_KEY\"" >> ~/.bashrc
```

## Common Tasks

### Stop the Server

Press `Ctrl+C` for graceful shutdown

### Stop MongoDB

```bash
docker stop mulamail-mongo
```

### View MongoDB Data

```bash
# Connect to MongoDB
docker exec -it mulamail-mongo mongosh

# In mongosh:
use mulamail
db.identities.find()
db.mail_accounts.find()
```

### Change Port

```bash
PORT=3000 go run main.go
```

### Use Mainnet

```bash
export SOLANA_RPC="https://api.mainnet-beta.solana.com"
go run main.go
```

## Next Steps

- 📖 Read [server/README.md](server/README.md) for detailed setup
- 🧪 Run tests: `cd server && make test`
- 📚 Check [whitepaper.md](whitepaper.md) for API docs
- 🔧 See [server/TEST_README.md](server/TEST_README.md) for testing guide

## Troubleshooting

### MongoDB Connection Error

```bash
# Ensure MongoDB is running
docker ps | grep mulamail-mongo

# Restart if needed
docker restart mulamail-mongo
```

### Port Already in Use

```bash
# Use different port
PORT=3000 go run main.go

# Or kill process using port 8080
lsof -ti:8080 | xargs kill -9
```

### Invalid Encryption Key

```bash
# Generate new key (must be 64 hex chars)
export ENCRYPTION_KEY=$(openssl rand -hex 32)
```

## API Endpoints Overview

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/health` | Health check |
| POST | `/api/v1/identity/create-tx` | Create identity transaction |
| POST | `/api/v1/identity/register` | Register identity |
| GET | `/api/v1/identity/resolve` | Resolve email↔pubkey |
| POST | `/api/v1/accounts` | Add mail account |
| GET | `/api/v1/accounts` | List accounts |
| GET | `/api/v1/mail/inbox` | Fetch inbox |
| GET | `/api/v1/mail/message` | Get message |
| POST | `/api/v1/mail/send` | Send mail |

## Development Mode

```bash
# Use devnet (faster, free)
export SOLANA_RPC="https://api.devnet.solana.com"

# Enable hot reload (install air first)
go install github.com/cosmtrek/air@latest
air

# Run tests
make test
```

## Production Notes

⚠️ Before deploying to production:

1. Generate unique `ENCRYPTION_KEY` (never use default!)
2. Use mainnet: `SOLANA_RPC=https://api.mainnet-beta.solana.com`
3. Enable MongoDB authentication
4. Use HTTPS/TLS
5. Set up monitoring
6. Configure rate limiting
7. Review security checklist in [server/README.md](server/README.md)

## Resources

- **Full Documentation**: [server/README.md](server/README.md)
- **Testing Guide**: [server/TEST_README.md](server/TEST_README.md)
- **Test Summary**: [server/TEST_SUMMARY.md](server/TEST_SUMMARY.md)
- **Whitepaper**: [whitepaper.md](whitepaper.md)

## Support

Need help? Check:
- GitHub Issues
- Documentation files
- Test examples in `*_test.go` files

---

**Happy Coding! 🚀**

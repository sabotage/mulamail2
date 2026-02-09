# Running MulaMail 2 Locally (No Cloud Required!)

MulaMail 2 now runs completely locally by default - no AWS, no cloud accounts, just your machine!

## Quick Start (60 seconds)

```bash
cd server
./start.sh
```

That's it! The script will:
1. ✅ Ask if you want local or S3 storage (choose local)
2. ✅ Check for MongoDB (and offer to start it via Docker)
3. ✅ Generate encryption keys
4. ✅ Start the server

## What You Get with Local Storage

✅ **No cloud accounts needed** - Everything runs on your machine
✅ **No AWS bills** - Zero external costs
✅ **Faster** - Direct filesystem access (~1ms vs ~50ms for S3)
✅ **Simpler** - No IAM, no buckets, no credentials
✅ **Private** - Your data never leaves your machine
✅ **Secure** - Files stored with 0600 permissions (owner-only)

## Manual Setup (3 commands)

If you prefer manual setup:

```bash
# 1. Start MongoDB
docker run -d -p 27017:27017 --name mulamail-mongo mongo:latest

# 2. Configure for local storage
export STORAGE_TYPE="local"
export LOCAL_DATA_PATH="./data/vault"
export ENCRYPTION_KEY=$(openssl rand -hex 32)

# 3. Run server
cd server
go run main.go
```

Done! Server running on `http://localhost:8080`

## Directory Structure

After starting, you'll see:

```
server/
├── main.go
├── data/
│   └── vault/          # ← Your encrypted mail stored here
│       └── (files will appear as you use the system)
└── ...
```

## Configuration

All configuration via environment variables (optional):

```bash
# Server
export PORT="8080"                    # HTTP port

# Database
export MONGO_URI="mongodb://localhost:27017"
export MONGO_DB="mulamail"

# Storage (LOCAL - no cloud!)
export STORAGE_TYPE="local"           # Use local filesystem
export LOCAL_DATA_PATH="./data/vault" # Where to store files

# Blockchain
export SOLANA_RPC="https://api.devnet.solana.com"  # Free devnet

# Encryption
export ENCRYPTION_KEY=$(openssl rand -hex 32)  # Generate new key
```

## What Gets Stored Locally

### MongoDB (Database)
- **Location**: Docker volume or `/var/lib/mongodb`
- **Stores**: Identity mappings, mail account configs (passwords encrypted)
- **Size**: ~10MB for typical use

### Vault (Encrypted Mail)
- **Location**: `./data/vault/` (customizable)
- **Stores**: Encrypted email messages, attachments
- **Size**: Depends on your email volume

### Permissions

Files are automatically secured:
- Directories: `755` (rwxr-xr-x)
- Files: `600` (rw-------) - **only you can read!**

## Testing It Works

```bash
# Health check
curl http://localhost:8080/api/health
# Expected: {"status":"ok"}

# Add a mail account
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "owner_pubkey": "testuser",
    "account_email": "your-email@gmail.com",
    "pop3": {
      "host": "pop.gmail.com",
      "port": 995,
      "user": "your-email@gmail.com",
      "pass": "your-app-password",
      "use_ssl": true
    },
    "smtp": {
      "host": "smtp.gmail.com",
      "port": 587,
      "user": "your-email@gmail.com",
      "pass": "your-app-password",
      "use_ssl": true
    }
  }'

# List accounts
curl "http://localhost:8080/api/v1/accounts?owner=testuser"
```

## Where Is My Data?

### View Your Data

```bash
# See stored files
ls -la ./data/vault/

# Check MongoDB data
docker exec -it mulamail-mongo mongosh
> use mulamail
> db.identities.find()
> db.mail_accounts.find()
```

### Backup Your Data

```bash
# Backup everything
tar -czf mulamail-backup.tar.gz ./data/vault/

# Backup MongoDB
docker exec mulamail-mongo mongodump --out=/backup
```

### Clean Up / Reset

```bash
# Remove all mail vault data
rm -rf ./data/vault/*

# Reset MongoDB (careful!)
docker stop mulamail-mongo
docker rm mulamail-mongo
docker volume rm mulamail-data
```

## Storage Usage Examples

### Scenario 1: Personal Use (1 email account, 1000 emails)
- **Database**: ~5MB
- **Vault**: ~50MB
- **Total**: ~55MB

### Scenario 2: Power User (5 accounts, 10,000 emails)
- **Database**: ~25MB
- **Vault**: ~500MB
- **Total**: ~525MB

### Scenario 3: Archive (100,000 emails)
- **Database**: ~50MB
- **Vault**: ~5GB
- **Total**: ~5GB

*Note: Sizes vary based on email content and attachments*

## Moving to Cloud Later?

Easy! If you later want to use S3:

```bash
# 1. Sync your local data to S3
aws s3 sync ./data/vault/ s3://your-bucket/

# 2. Change config
export STORAGE_TYPE="s3"
export S3_BUCKET="your-bucket"

# 3. Restart server
go run main.go
```

See [STORAGE.md](STORAGE.md) for details.

## Advantages of Local Storage

| Feature | Local Storage | S3 Storage |
|---------|---------------|------------|
| **Setup** | Zero config | AWS account + IAM |
| **Speed** | ~1ms | ~50-100ms |
| **Cost** | Free (disk) | $0.023/GB/month |
| **Privacy** | Never leaves machine | Stored on AWS |
| **Dependencies** | None | Internet required |
| **Simplicity** | ⭐⭐⭐⭐⭐ | ⭐⭐ |

## Common Questions

### Q: Is local storage production-ready?
**A:** Yes! It's actually **recommended** for single-server deployments.

### Q: What about backups?
**A:** Use standard filesystem backups (rsync, Time Machine, etc.) or MongoDB replication.

### Q: Can I encrypt the storage directory?
**A:** Yes! Use full-disk encryption (LUKS, FileVault) or encrypted volumes.

### Q: What if I run out of disk space?
**A:** Either clean up old mail, add more disk, or switch to S3.

### Q: Is it safe?
**A:** Yes! Files have 0600 permissions (only server can read). Passwords are encrypted with AES-256-GCM.

## Troubleshooting

### Problem: "Permission denied" creating vault directory

```bash
# Fix permissions
mkdir -p ./data/vault
chmod 755 ./data/vault
```

### Problem: Files not showing up

```bash
# Check directory exists
ls -la ./data/vault/

# Verify storage type
echo $STORAGE_TYPE  # Should be "local"

# Check server logs
# (the server logs "Using local storage: path=...")
```

### Problem: Want to change storage location

```bash
# Before starting server:
export LOCAL_DATA_PATH="/path/to/your/storage"
go run main.go

# Move existing data:
mv ./data/vault/* /path/to/your/storage/
```

## Summary

✅ **Zero cloud setup** - Just run `./start.sh`
✅ **All local** - Your data stays on your machine
✅ **Fast & simple** - Direct filesystem access
✅ **Secure** - Files locked to server process
✅ **Free** - No cloud bills
✅ **Can migrate** - Move to S3 later if needed

**Bottom line**: MulaMail 2 works great locally - no AWS account required! 🎉

---

Need help? Check:
- [QUICKSTART.md](../QUICKSTART.md) - General setup
- [README.md](README.md) - Full documentation
- [STORAGE.md](STORAGE.md) - Storage details

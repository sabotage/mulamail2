# Storage Adapter - Configuration Guide

## Overview

MulaMail 2 uses a **pluggable storage adapter** pattern for storing encrypted mail data. You can choose between:

- **Local File Storage** (default) - Store files on the local filesystem
- **AWS S3** - Store files in cloud object storage

## Quick Start

### Using Local Storage (Default)

```bash
# Local storage is the default - no additional setup needed!
export STORAGE_TYPE="local"
export LOCAL_DATA_PATH="./data/vault"  # optional, this is the default

go run main.go
```

Files will be stored in `./data/vault/` directory with secure permissions (0600).

### Using AWS S3

```bash
export STORAGE_TYPE="s3"
export AWS_REGION="us-east-1"
export S3_BUCKET="mulamail-vault"
# AWS credentials from ~/.aws/credentials or environment

go run main.go
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `STORAGE_TYPE` | No | `local` | Storage backend: `local` or `s3` |
| `LOCAL_DATA_PATH` | No | `./data/vault` | Base directory for local storage |
| `AWS_REGION` | S3 only | `us-east-1` | AWS region for S3 |
| `S3_BUCKET` | S3 only | `mulamail-vault` | S3 bucket name |

### Local Storage Configuration

```bash
# Default configuration
export STORAGE_TYPE="local"
export LOCAL_DATA_PATH="./data/vault"

# Custom location
export LOCAL_DATA_PATH="/var/lib/mulamail/vault"

# Relative to working directory
export LOCAL_DATA_PATH="./custom/storage/path"
```

### S3 Storage Configuration

```bash
export STORAGE_TYPE="s3"
export AWS_REGION="us-west-2"
export S3_BUCKET="my-mulamail-bucket"

# AWS credentials (pick one method):
# 1. AWS credentials file (~/.aws/credentials)
# 2. Environment variables:
export AWS_ACCESS_KEY_ID="your-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
# 3. EC2 instance profile (when running on AWS)
```

## Storage Interface

Both storage backends implement the same interface:

```go
type Storage interface {
    // Put stores raw bytes at the given key
    Put(ctx context.Context, key string, data []byte) error

    // Get retrieves the object at the given key
    Get(ctx context.Context, key string) ([]byte, error)

    // Delete removes the object at the given key
    Delete(ctx context.Context, key string) error

    // List returns all keys with the given prefix
    List(ctx context.Context, prefix string) ([]string, error)
}
```

## Local Storage Details

### Features

✅ **Zero external dependencies** - No cloud account needed
✅ **Fast** - Direct filesystem access
✅ **Secure permissions** - Files created with 0600 (owner read/write only)
✅ **Directory organization** - Automatic nested directory creation
✅ **Path traversal protection** - Sanitizes keys to prevent `../` attacks
✅ **Binary safe** - Handles any binary data

### Directory Structure

```
./data/vault/
├── user1/
│   ├── message1.enc
│   └── message2.enc
├── user2/
│   └── inbox/
│       └── mail.enc
└── attachments/
    └── file.pdf
```

### File Permissions

- Directories: `0755` (rwxr-xr-x)
- Files: `0600` (rw-------)

Files are only readable/writable by the server process owner.

### Storage Location

By default, files are stored in `./data/vault/` relative to the server working directory:

```bash
server/
├── main.go
├── data/
│   └── vault/          # Storage location
│       └── ...files...
└── ...
```

### Cleanup

Local storage automatically cleans up empty directories when files are deleted.

## S3 Storage Details

### Features

✅ **Scalable** - Cloud storage with unlimited capacity
✅ **Durable** - 99.999999999% (11 9's) durability
✅ **Distributed** - Access from multiple servers
✅ **Versioning** - Optional version history
✅ **Encryption** - Server-side encryption available

### S3 Bucket Setup

1. **Create S3 Bucket**
   ```bash
   aws s3 mb s3://mulamail-vault --region us-east-1
   ```

2. **Enable Encryption** (recommended)
   ```bash
   aws s3api put-bucket-encryption \
     --bucket mulamail-vault \
     --server-side-encryption-configuration '{
       "Rules": [{
         "ApplyServerSideEncryptionByDefault": {
           "SSEAlgorithm": "AES256"
         }
       }]
     }'
   ```

3. **Set Bucket Policy** (restrict access)
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Principal": {
         "AWS": "arn:aws:iam::YOUR-ACCOUNT-ID:user/mulamail-service"
       },
       "Action": [
         "s3:GetObject",
         "s3:PutObject",
         "s3:DeleteObject",
         "s3:ListBucket"
       ],
       "Resource": [
         "arn:aws:s3:::mulamail-vault/*",
         "arn:aws:s3:::mulamail-vault"
       ]
     }]
   }
   ```

### IAM Permissions

Minimum required permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket"
    ],
    "Resource": [
      "arn:aws:s3:::mulamail-vault/*",
      "arn:aws:s3:::mulamail-vault"
    ]
  }]
}
```

## Switching Storage Backends

### Local to S3 Migration

```bash
# 1. Upload existing files to S3
aws s3 sync ./data/vault/ s3://mulamail-vault/

# 2. Update configuration
export STORAGE_TYPE="s3"
export S3_BUCKET="mulamail-vault"

# 3. Restart server
go run main.go
```

### S3 to Local Migration

```bash
# 1. Download files from S3
aws s3 sync s3://mulamail-vault/ ./data/vault/

# 2. Update configuration
export STORAGE_TYPE="local"
export LOCAL_DATA_PATH="./data/vault"

# 3. Restart server
go run main.go
```

## Testing

### Local Storage Tests

```bash
cd server
go test ./vault -v -run TestLocalStorage
```

### Integration Tests

```bash
# Test with local storage
export STORAGE_TYPE="local"
go test ./api -v

# Test with S3 (requires AWS credentials)
export STORAGE_TYPE="s3"
export AWS_REGION="us-east-1"
export S3_BUCKET="test-bucket"
go test ./api -v
```

## Performance Comparison

| Feature | Local Storage | S3 Storage |
|---------|---------------|------------|
| **Latency** | ~1ms | ~50-100ms |
| **Throughput** | Disk speed | Network speed |
| **Scalability** | Disk capacity | Unlimited |
| **Cost** | Disk cost | Pay per GB + requests |
| **Durability** | Depends on RAID | 99.999999999% |
| **Multi-server** | Requires NFS/shared disk | Native |

### Recommendations

- **Development**: Use **local storage** (faster, simpler)
- **Single-server production**: Use **local storage** with backups
- **Multi-server production**: Use **S3** for shared access
- **High-performance needs**: Use **local storage** with SSD
- **Distributed systems**: Use **S3** or other object storage

## Security Considerations

### Local Storage

✅ Files stored with secure permissions (0600)
✅ Path traversal protection
✅ No external dependencies
⚠️ Requires proper filesystem encryption (LUKS, FileVault, etc.)
⚠️ Needs separate backup strategy

### S3 Storage

✅ Server-side encryption available
✅ IAM-based access control
✅ Audit logging with CloudTrail
✅ Automatic backups via versioning
⚠️ Data leaves server boundary
⚠️ Requires proper IAM configuration

## Custom Storage Backend

To implement a custom storage backend:

1. **Implement the `Storage` interface**:
   ```go
   type MyStorage struct {
       // your fields
   }

   func (s *MyStorage) Put(ctx context.Context, key string, data []byte) error {
       // implementation
   }

   func (s *MyStorage) Get(ctx context.Context, key string) ([]byte, error) {
       // implementation
   }

   func (s *MyStorage) Delete(ctx context.Context, key string) error {
       // implementation
   }

   func (s *MyStorage) List(ctx context.Context, prefix string) ([]string, error) {
       // implementation
   }
   ```

2. **Add to `main.go`**:
   ```go
   case "mystorage":
       storage, err := vault.NewMyStorage(config)
       if err != nil {
           log.Fatalf("MyStorage init: %v", err)
       }
   ```

3. **Update config**:
   ```bash
   export STORAGE_TYPE="mystorage"
   ```

## Troubleshooting

### Local Storage Issues

**Error**: `create base directory: permission denied`
- **Fix**: Ensure server has write permissions to the directory
  ```bash
  chmod 755 ./data
  mkdir -p ./data/vault
  ```

**Error**: `base directory not writable`
- **Fix**: Check filesystem permissions and disk space
  ```bash
  df -h  # Check disk space
  ls -la ./data  # Check permissions
  ```

### S3 Storage Issues

**Error**: `S3 init: NoCredentialsErr`
- **Fix**: Set up AWS credentials
  ```bash
  aws configure
  # Or set environment variables
  export AWS_ACCESS_KEY_ID="..."
  export AWS_SECRET_ACCESS_KEY="..."
  ```

**Error**: `AccessDenied`
- **Fix**: Verify IAM permissions for the bucket

**Error**: `NoSuchBucket`
- **Fix**: Create the bucket or verify the name
  ```bash
  aws s3 mb s3://mulamail-vault
  ```

## Best Practices

1. **Use local storage for development** - Faster and simpler
2. **Use S3 for production multi-server deployments**
3. **Enable S3 versioning** for backup and recovery
4. **Set up lifecycle policies** to automatically archive old data
5. **Monitor storage usage** and costs
6. **Encrypt local storage filesystems** (LUKS, dm-crypt)
7. **Use IAM roles** instead of access keys when possible
8. **Regular backups** regardless of storage backend

## Examples

### Example 1: Local Development

```bash
export STORAGE_TYPE="local"
export LOCAL_DATA_PATH="./dev-data"
go run main.go
```

### Example 2: Production with S3

```bash
export STORAGE_TYPE="s3"
export AWS_REGION="us-east-1"
export S3_BUCKET="mulamail-prod-vault"
./mulamail-server
```

### Example 3: Docker with Local Volume

```yaml
version: '3.8'
services:
  mulamail:
    image: mulamail-server
    environment:
      - STORAGE_TYPE=local
      - LOCAL_DATA_PATH=/data/vault
    volumes:
      - mulamail-data:/data/vault

volumes:
  mulamail-data:
```

## Summary

✅ **Local storage** is the default - works out of the box
✅ **S3 storage** available for cloud deployments
✅ **Easy to switch** between backends via environment variables
✅ **Same interface** - transparent to application code
✅ **Extensible** - easy to add custom storage backends

For most users, **local storage is recommended** as it's simpler, faster, and requires no external services.

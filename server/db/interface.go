package db

import "context"

// DB defines the interface for database operations
type DB interface {
	CreateIdentity(ctx context.Context, id *Identity) error
	GetIdentityByEmail(ctx context.Context, email string) (*Identity, error)
	GetIdentityByPubKey(ctx context.Context, pubkey string) (*Identity, error)
	CreateMailAccount(ctx context.Context, acc *MailAccount) error
	GetMailAccountsByOwner(ctx context.Context, ownerPubKey string) ([]MailAccount, error)
	GetMailAccount(ctx context.Context, ownerPubKey, accountEmail string) (*MailAccount, error)
}

// Ensure Client implements DB interface
var _ DB = (*Client)(nil)

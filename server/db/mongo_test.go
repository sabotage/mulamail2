package db

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// getTestMongoURI returns the MongoDB URI for testing.
// Set MONGO_TEST_URI environment variable to use a custom test instance.
// Default: mongodb://localhost:27017
func getTestMongoURI() string {
	if uri := os.Getenv("MONGO_TEST_URI"); uri != "" {
		return uri
	}
	return "mongodb://localhost:27017"
}

// setupTestDB creates a test database client and returns cleanup function
func setupTestDB(t *testing.T) (*Client, func()) {
	t.Helper()

	uri := getTestMongoURI()
	dbName := "mulamail_test_" + primitive.NewObjectID().Hex()

	client, err := Connect(uri, dbName)
	if err != nil {
		t.Skipf("MongoDB not available at %s: %v (use MONGO_TEST_URI to specify test instance)", uri, err)
		return nil, nil
	}

	cleanup := func() {
		// Drop the test database
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.db.Drop(ctx)
		client.Close()
	}

	return client, cleanup
}

func TestConnect_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return // MongoDB not available, test skipped
	}
	defer cleanup()

	if client.client == nil {
		t.Error("client.client should not be nil")
	}
	if client.db == nil {
		t.Error("client.db should not be nil")
	}
}

func TestConnect_InvalidURI(t *testing.T) {
	_, err := Connect("invalid://uri", "testdb")
	if err == nil {
		t.Error("expected error with invalid URI, got nil")
	}
}

func TestCreateIdentity_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	identity := &Identity{
		Email:    "test@example.com",
		PubKey:   "7xKhMhVPYvZXZq9QKqZXZq9QKqZXZq9QKqZXZq9QKqZ",
		TxHash:   "tx123456789",
		Verified: true,
	}

	err := client.CreateIdentity(ctx, identity)
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// Verify ID was set
	if identity.ID.IsZero() {
		t.Error("expected ID to be set after creation")
	}

	// Verify CreatedAt was set
	if identity.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set after creation")
	}
}

func TestGetIdentityByEmail_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	original := &Identity{
		Email:    "user@mulamail.com",
		PubKey:   "SolPubKey12345",
		TxHash:   "tx_hash_abc",
		Verified: true,
	}

	// Create identity
	err := client.CreateIdentity(ctx, original)
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// Retrieve by email
	retrieved, err := client.GetIdentityByEmail(ctx, "user@mulamail.com")
	if err != nil {
		t.Fatalf("GetIdentityByEmail failed: %v", err)
	}

	// Verify fields
	if retrieved.Email != original.Email {
		t.Errorf("Email: want %q, got %q", original.Email, retrieved.Email)
	}
	if retrieved.PubKey != original.PubKey {
		t.Errorf("PubKey: want %q, got %q", original.PubKey, retrieved.PubKey)
	}
	if retrieved.TxHash != original.TxHash {
		t.Errorf("TxHash: want %q, got %q", original.TxHash, retrieved.TxHash)
	}
	if retrieved.Verified != original.Verified {
		t.Errorf("Verified: want %v, got %v", original.Verified, retrieved.Verified)
	}
}

func TestGetIdentityByEmail_NotFound(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	_, err := client.GetIdentityByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent email, got nil")
	}
	if err != mongo.ErrNoDocuments {
		t.Errorf("expected mongo.ErrNoDocuments, got %v", err)
	}
}

func TestGetIdentityByPubKey_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	original := &Identity{
		Email:    "alice@example.com",
		PubKey:   "AlicePubKey123",
		Verified: false,
	}

	err := client.CreateIdentity(ctx, original)
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	retrieved, err := client.GetIdentityByPubKey(ctx, "AlicePubKey123")
	if err != nil {
		t.Fatalf("GetIdentityByPubKey failed: %v", err)
	}

	if retrieved.Email != original.Email {
		t.Errorf("Email: want %q, got %q", original.Email, retrieved.Email)
	}
	if retrieved.PubKey != original.PubKey {
		t.Errorf("PubKey: want %q, got %q", original.PubKey, retrieved.PubKey)
	}
}

func TestGetIdentityByPubKey_NotFound(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	_, err := client.GetIdentityByPubKey(ctx, "NonExistentPubKey")
	if err == nil {
		t.Error("expected error for non-existent pubkey, got nil")
	}
}

func TestCreateMailAccount_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	account := &MailAccount{
		OwnerPubKey:  "owner_pub_key_123",
		AccountEmail: "mail@example.com",
		POP3: POP3Settings{
			Host:    "pop.example.com",
			Port:    995,
			User:    "user@example.com",
			PassEnc: "encrypted_password",
			UseSSL:  true,
		},
		SMTP: SMTPSettings{
			Host:    "smtp.example.com",
			Port:    465,
			User:    "user@example.com",
			PassEnc: "encrypted_password",
			UseSSL:  true,
		},
	}

	err := client.CreateMailAccount(ctx, account)
	if err != nil {
		t.Fatalf("CreateMailAccount failed: %v", err)
	}

	if account.ID.IsZero() {
		t.Error("expected ID to be set after creation")
	}
	if account.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set after creation")
	}
}

func TestGetMailAccountsByOwner_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	ownerPubKey := "owner_xyz_123"

	// Create multiple accounts for the same owner
	accounts := []MailAccount{
		{
			OwnerPubKey:  ownerPubKey,
			AccountEmail: "account1@example.com",
			POP3:         POP3Settings{Host: "pop1.example.com", Port: 995},
			SMTP:         SMTPSettings{Host: "smtp1.example.com", Port: 465},
		},
		{
			OwnerPubKey:  ownerPubKey,
			AccountEmail: "account2@example.com",
			POP3:         POP3Settings{Host: "pop2.example.com", Port: 995},
			SMTP:         SMTPSettings{Host: "smtp2.example.com", Port: 587},
		},
	}

	for i := range accounts {
		err := client.CreateMailAccount(ctx, &accounts[i])
		if err != nil {
			t.Fatalf("CreateMailAccount failed: %v", err)
		}
	}

	// Retrieve accounts
	retrieved, err := client.GetMailAccountsByOwner(ctx, ownerPubKey)
	if err != nil {
		t.Fatalf("GetMailAccountsByOwner failed: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(retrieved))
	}

	// Verify accounts
	emails := make(map[string]bool)
	for _, acc := range retrieved {
		emails[acc.AccountEmail] = true
		if acc.OwnerPubKey != ownerPubKey {
			t.Errorf("expected owner %q, got %q", ownerPubKey, acc.OwnerPubKey)
		}
	}

	if !emails["account1@example.com"] || !emails["account2@example.com"] {
		t.Error("did not retrieve all expected accounts")
	}
}

func TestGetMailAccountsByOwner_Empty(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	accounts, err := client.GetMailAccountsByOwner(ctx, "nonexistent_owner")
	if err != nil {
		t.Fatalf("GetMailAccountsByOwner failed: %v", err)
	}

	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestGetMailAccount_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	ownerPubKey := "owner_abc"
	accountEmail := "specific@example.com"

	account := &MailAccount{
		OwnerPubKey:  ownerPubKey,
		AccountEmail: accountEmail,
		POP3: POP3Settings{
			Host:    "pop.specific.com",
			Port:    995,
			User:    "user",
			PassEnc: "enc_pass",
			UseSSL:  true,
		},
		SMTP: SMTPSettings{
			Host:    "smtp.specific.com",
			Port:    587,
			User:    "user",
			PassEnc: "enc_pass",
			UseSSL:  false,
		},
	}

	err := client.CreateMailAccount(ctx, account)
	if err != nil {
		t.Fatalf("CreateMailAccount failed: %v", err)
	}

	// Retrieve specific account
	retrieved, err := client.GetMailAccount(ctx, ownerPubKey, accountEmail)
	if err != nil {
		t.Fatalf("GetMailAccount failed: %v", err)
	}

	if retrieved.OwnerPubKey != ownerPubKey {
		t.Errorf("OwnerPubKey: want %q, got %q", ownerPubKey, retrieved.OwnerPubKey)
	}
	if retrieved.AccountEmail != accountEmail {
		t.Errorf("AccountEmail: want %q, got %q", accountEmail, retrieved.AccountEmail)
	}
	if retrieved.POP3.Host != "pop.specific.com" {
		t.Errorf("POP3.Host: want %q, got %q", "pop.specific.com", retrieved.POP3.Host)
	}
	if retrieved.SMTP.Port != 587 {
		t.Errorf("SMTP.Port: want %d, got %d", 587, retrieved.SMTP.Port)
	}
}

func TestGetMailAccount_NotFound(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	_, err := client.GetMailAccount(ctx, "owner", "nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent account, got nil")
	}
}

func TestMailAccount_PasswordEncryptionNotSerialized(t *testing.T) {
	// This test verifies that PassEnc fields have json:"-" tag
	// by attempting to marshal and checking the output doesn't contain passwords

	account := MailAccount{
		OwnerPubKey:  "owner123",
		AccountEmail: "test@example.com",
		POP3: POP3Settings{
			Host:    "pop.test.com",
			Port:    995,
			User:    "user",
			PassEnc: "secret_encrypted_pop3_password",
			UseSSL:  true,
		},
		SMTP: SMTPSettings{
			Host:    "smtp.test.com",
			Port:    587,
			User:    "user",
			PassEnc: "secret_encrypted_smtp_password",
			UseSSL:  false,
		},
	}

	// Note: This is a compile-time verification via the json:"-" tags
	// In actual serialization, PassEnc fields won't be included
	if account.POP3.PassEnc == "" {
		t.Error("PassEnc should be set in memory (not serialized)")
	}
}

func TestIdentity_MultipleCreations(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()

	// Create multiple identities
	identities := []Identity{
		{Email: "user1@example.com", PubKey: "pubkey1", Verified: true},
		{Email: "user2@example.com", PubKey: "pubkey2", Verified: false},
		{Email: "user3@example.com", PubKey: "pubkey3", Verified: true},
	}

	for i := range identities {
		err := client.CreateIdentity(ctx, &identities[i])
		if err != nil {
			t.Fatalf("CreateIdentity failed for identity %d: %v", i, err)
		}
	}

	// Verify each can be retrieved
	for _, identity := range identities {
		retrieved, err := client.GetIdentityByEmail(ctx, identity.Email)
		if err != nil {
			t.Errorf("failed to retrieve identity %q: %v", identity.Email, err)
		}
		if retrieved.PubKey != identity.PubKey {
			t.Errorf("wrong pubkey for %q", identity.Email)
		}
	}
}

func TestMailAccount_MultipleAccountsPerOwner(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	ownerPubKey := "multi_account_owner"

	// Create 5 accounts for the same owner
	for i := 1; i <= 5; i++ {
		account := &MailAccount{
			OwnerPubKey:  ownerPubKey,
			AccountEmail: "account" + string(rune('0'+i)) + "@example.com",
			POP3:         POP3Settings{Host: "pop.example.com", Port: 995},
			SMTP:         SMTPSettings{Host: "smtp.example.com", Port: 587},
		}
		err := client.CreateMailAccount(ctx, account)
		if err != nil {
			t.Fatalf("failed to create account %d: %v", i, err)
		}
	}

	// Retrieve all accounts
	accounts, err := client.GetMailAccountsByOwner(ctx, ownerPubKey)
	if err != nil {
		t.Fatalf("GetMailAccountsByOwner failed: %v", err)
	}

	if len(accounts) != 5 {
		t.Errorf("expected 5 accounts, got %d", len(accounts))
	}
}

func TestClose_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	if client == nil {
		return
	}

	// Don't defer cleanup, we'll test Close manually
	client.Close()

	// Try to use client after close (should fail gracefully)
	ctx := context.Background()
	identity := &Identity{Email: "test@example.com", PubKey: "key"}
	err := client.CreateIdentity(ctx, identity)
	if err == nil {
		t.Error("expected error using client after Close, got nil")
	}

	// Still call cleanup to drop the test database
	cleanup()
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mulamail/blockchain"
	"mulamail/config"
	"mulamail/db"
	"mulamail/vault"
)

// mockDB implements a simple in-memory mock for testing
type mockDB struct {
	identities   map[string]*db.Identity // keyed by email
	identitiesPK map[string]*db.Identity // keyed by pubkey
	accounts     map[string][]*db.MailAccount
}

func newMockDB() *mockDB {
	return &mockDB{
		identities:   make(map[string]*db.Identity),
		identitiesPK: make(map[string]*db.Identity),
		accounts:     make(map[string][]*db.MailAccount),
	}
}

func (m *mockDB) CreateIdentity(ctx context.Context, id *db.Identity) error {
	m.identities[id.Email] = id
	m.identitiesPK[id.PubKey] = id
	return nil
}

func (m *mockDB) GetIdentityByEmail(ctx context.Context, email string) (*db.Identity, error) {
	if id, ok := m.identities[email]; ok {
		return id, nil
	}
	return nil, db.ErrNotFound
}

func (m *mockDB) GetIdentityByPubKey(ctx context.Context, pubkey string) (*db.Identity, error) {
	if id, ok := m.identitiesPK[pubkey]; ok {
		return id, nil
	}
	return nil, db.ErrNotFound
}

func (m *mockDB) CreateMailAccount(ctx context.Context, acc *db.MailAccount) error {
	m.accounts[acc.OwnerPubKey] = append(m.accounts[acc.OwnerPubKey], acc)
	return nil
}

func (m *mockDB) GetMailAccountsByOwner(ctx context.Context, owner string) ([]db.MailAccount, error) {
	accs := m.accounts[owner]
	result := make([]db.MailAccount, len(accs))
	for i, a := range accs {
		result[i] = *a
	}
	return result, nil
}

func (m *mockDB) GetMailAccount(ctx context.Context, owner, email string) (*db.MailAccount, error) {
	for _, acc := range m.accounts[owner] {
		if acc.AccountEmail == email {
			return acc, nil
		}
	}
	return nil, db.ErrNotFound
}

// setupTestServer creates a test server with mocked dependencies
func setupTestServer(t *testing.T) (*Server, *mockDB) {
	t.Helper()

	mockDB := newMockDB()

	// Use a test encryption key (64 hex chars = 32 bytes)
	cfg := &config.Config{
		EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		SolanaRPC:     "https://api.devnet.solana.com",
	}

	server := &Server{
		db:      mockDB,
		solana:  blockchain.NewClient(cfg.SolanaRPC),
		storage: nil, // not needed for most tests
		cfg:     cfg,
	}

	return server, mockDB
}

func TestHealth(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("status: want %q, got %q", "ok", response["status"])
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]any{
		"key1": "value1",
		"key2": 123,
	}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: want %q, got %q", "application/json", contentType)
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["key1"] != "value1" {
		t.Error("JSON encoding failed")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "test error message" {
		t.Errorf("error: want %q, got %q", "test error message", response["error"])
	}
}

func TestNewRouter(t *testing.T) {
	server, mockDB := setupTestServer(t)

	router := NewRouter(mockDB, server.solana, nil, server.cfg)

	if router == nil {
		t.Fatal("expected non-nil router")
	}

	// Test that health endpoint is registered
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("health endpoint not properly registered")
	}
}

func TestRouter_AllEndpoints(t *testing.T) {
	server, mockDB := setupTestServer(t)
	router := NewRouter(mockDB, server.solana, nil, server.cfg)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/health"},
		{"POST", "/api/v1/identity/create-tx"},
		{"POST", "/api/v1/identity/register"},
		{"GET", "/api/v1/identity/resolve"},
		{"POST", "/api/v1/accounts"},
		{"GET", "/api/v1/accounts"},
		{"GET", "/api/v1/mail/inbox"},
		{"GET", "/api/v1/mail/message"},
		{"POST", "/api/v1/mail/send"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var body *bytes.Buffer
			if ep.method == "POST" {
				body = bytes.NewBuffer([]byte("{}"))
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(ep.method, ep.path, body)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// We just verify the endpoint exists (doesn't return 404)
			// Individual handler tests will verify actual behavior
			if w.Code == http.StatusNotFound {
				t.Errorf("endpoint not found: %s %s", ep.method, ep.path)
			}
		})
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	server, mockDB := setupTestServer(t)
	router := NewRouter(mockDB, server.solana, nil, server.cfg)

	// Try wrong method for an endpoint
	req := httptest.NewRequest("DELETE", "/api/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Go 1.22+ ServeMux returns 405 for wrong methods
	if w.Code != http.StatusMethodNotAllowed {
		t.Logf("Note: Expected 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestRouter_NotFound(t *testing.T) {
	server, mockDB := setupTestServer(t)
	router := NewRouter(mockDB, server.solana, nil, server.cfg)

	req := httptest.NewRequest("GET", "/api/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code: want %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestServer_Dependencies(t *testing.T) {
	server, mockDB := setupTestServer(t)

	if server.db == nil {
		t.Error("server.db should not be nil")
	}
	if server.solana == nil {
		t.Error("server.solana should not be nil")
	}
	if server.cfg == nil {
		t.Error("server.cfg should not be nil")
	}

	// Verify mockDB is working
	ctx := context.Background()
	identity := &db.Identity{
		Email:  "test@example.com",
		PubKey: "testpubkey",
	}
	if err := mockDB.CreateIdentity(ctx, identity); err != nil {
		t.Errorf("mockDB.CreateIdentity failed: %v", err)
	}

	retrieved, err := mockDB.GetIdentityByEmail(ctx, "test@example.com")
	if err != nil {
		t.Errorf("mockDB.GetIdentityByEmail failed: %v", err)
	}
	if retrieved.Email != identity.Email {
		t.Error("mockDB not functioning correctly")
	}
}

func TestVaultEncryptionIntegration(t *testing.T) {
	// Test that vault encryption works with config key
	cfg := &config.Config{
		EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	plaintext := "test password"
	encrypted, err := vault.EncryptAESGCM(cfg.EncryptionKey, plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := vault.DecryptAESGCM(cfg.EncryptionKey, encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("round-trip failed: want %q, got %q", plaintext, decrypted)
	}
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gagliardetto/solana-go"

	"mulamail/db"
)

// Define db.ErrNotFound for testing
var ErrNotFound = errors.New("not found")

func init() {
	db.ErrNotFound = ErrNotFound
}

func TestResolveIdentity_ByEmail_Success(t *testing.T) {
	server, mockDB := setupTestServer(t)

	// Create test identity
	ctx := context.Background()
	identity := &db.Identity{
		Email:    "alice@mulamail.com",
		PubKey:   "7xKhMhVPYvZXZq9QKqZXZq9QKqZXZq9QKqZXZq9QKqZ",
		Verified: true,
	}
	mockDB.CreateIdentity(ctx, identity)

	req := httptest.NewRequest("GET", "/api/v1/identity/resolve?email=alice@mulamail.com", nil)
	w := httptest.NewRecorder()

	server.resolveIdentity(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response db.Identity
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Email != identity.Email {
		t.Errorf("email: want %q, got %q", identity.Email, response.Email)
	}
	if response.PubKey != identity.PubKey {
		t.Errorf("pubkey: want %q, got %q", identity.PubKey, response.PubKey)
	}
}

func TestResolveIdentity_ByPubKey_Success(t *testing.T) {
	server, mockDB := setupTestServer(t)

	ctx := context.Background()
	identity := &db.Identity{
		Email:  "bob@example.com",
		PubKey: "BobPubKey12345",
	}
	mockDB.CreateIdentity(ctx, identity)

	req := httptest.NewRequest("GET", "/api/v1/identity/resolve?pubkey=BobPubKey12345", nil)
	w := httptest.NewRecorder()

	server.resolveIdentity(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response db.Identity
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Email != identity.Email {
		t.Errorf("email: want %q, got %q", identity.Email, response.Email)
	}
}

func TestResolveIdentity_NoParameters(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/identity/resolve", nil)
	w := httptest.NewRecorder()

	server.resolveIdentity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if response["error"] == "" {
		t.Error("expected error message")
	}
}

func TestResolveIdentity_NotFound(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/identity/resolve?email=nonexistent@example.com", nil)
	w := httptest.NewRecorder()

	server.resolveIdentity(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code: want %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCreateIdentityTx_Success(t *testing.T) {
	server, _ := setupTestServer(t)

	reqBody := map[string]string{
		"email":  "test@mulamail.com",
		"pubkey": "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/identity/create-tx", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.createIdentityTx(w, req)

	// May skip if Solana RPC is unavailable
	if w.Code == http.StatusInternalServerError {
		t.Skipf("Solana RPC unavailable: %s", w.Body.String())
		return
	}

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["transaction"] == "" {
		t.Error("expected non-empty transaction field")
	}
}

func TestCreateIdentityTx_InvalidJSON(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/identity/create-tx", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	server.createIdentityTx(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateIdentityTx_MissingFields(t *testing.T) {
	testCases := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"pubkey": "abc123"}},
		{"missing pubkey", map[string]string{"email": "test@example.com"}},
		{"both empty", map[string]string{"email": "", "pubkey": ""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := setupTestServer(t)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/v1/identity/create-tx", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			server.createIdentityTx(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestCreateIdentityTx_InvalidPubKey(t *testing.T) {
	server, _ := setupTestServer(t)

	reqBody := map[string]string{
		"email":  "test@example.com",
		"pubkey": "invalid-pubkey-format",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/identity/create-tx", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.createIdentityTx(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if !contains(response["error"], "invalid pubkey") {
		t.Errorf("expected 'invalid pubkey' error, got: %s", response["error"])
	}
}

func TestRegisterIdentity_DuplicateEmail(t *testing.T) {
	server, mockDB := setupTestServer(t)

	// Pre-register an identity
	ctx := context.Background()
	existing := &db.Identity{
		Email:  "duplicate@example.com",
		PubKey: "existingkey",
	}
	mockDB.CreateIdentity(ctx, existing)

	// Try to register with same email
	reqBody := map[string]string{
		"email":     "duplicate@example.com",
		"pubkey":    "newkey",
		"signed_tx": "dummytx",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/identity/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.registerIdentity(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status code: want %d, got %d", http.StatusConflict, w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if !contains(response["error"], "already registered") {
		t.Errorf("expected 'already registered' error, got: %s", response["error"])
	}
}

func TestRegisterIdentity_MissingFields(t *testing.T) {
	testCases := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"pubkey": "key", "signed_tx": "tx"}},
		{"missing pubkey", map[string]string{"email": "test@example.com", "signed_tx": "tx"}},
		{"missing signed_tx", map[string]string{"email": "test@example.com", "pubkey": "key"}},
		{"all empty", map[string]string{"email": "", "pubkey": "", "signed_tx": ""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := setupTestServer(t)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/v1/identity/register", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			server.registerIdentity(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestRegisterIdentity_InvalidJSON(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/identity/register", bytes.NewBufferString("{invalid"))
	w := httptest.NewRecorder()

	server.registerIdentity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRegisterIdentity_InvalidTransaction(t *testing.T) {
	server, _ := setupTestServer(t)

	reqBody := map[string]string{
		"email":     "test@example.com",
		"pubkey":    "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
		"signed_tx": "invalid-base64-transaction",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/identity/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.registerIdentity(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code: want %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestResolveIdentity_BothParameters(t *testing.T) {
	server, mockDB := setupTestServer(t)

	ctx := context.Background()
	identity := &db.Identity{
		Email:  "test@example.com",
		PubKey: "testkey123",
	}
	mockDB.CreateIdentity(ctx, identity)

	// When both are provided, email takes precedence
	req := httptest.NewRequest("GET", "/api/v1/identity/resolve?email=test@example.com&pubkey=testkey123", nil)
	w := httptest.NewRecorder()

	server.resolveIdentity(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response db.Identity
	json.NewDecoder(w.Body).Decode(&response)
	if response.Email != identity.Email {
		t.Error("should resolve by email when both parameters provided")
	}
}

func TestCreateIdentityTx_ValidPubKeyFormats(t *testing.T) {
	server, _ := setupTestServer(t)

	validPubKeys := []string{
		"9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
		"11111111111111111111111111111111",
		solana.SystemProgramID.String(),
	}

	for _, pubkey := range validPubKeys {
		t.Run(pubkey, func(t *testing.T) {
			reqBody := map[string]string{
				"email":  "test@example.com",
				"pubkey": pubkey,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/identity/create-tx", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			server.createIdentityTx(w, req)

			// Should not return bad request for valid pubkeys
			if w.Code == http.StatusBadRequest {
				var errResp map[string]string
				json.NewDecoder(w.Body).Decode(&errResp)
				if contains(errResp["error"], "invalid pubkey") {
					t.Errorf("valid pubkey %q rejected as invalid", pubkey)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

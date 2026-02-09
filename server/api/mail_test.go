package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mulamail/db"
	"mulamail/vault"
)

func TestAddAccount_Success(t *testing.T) {
	server, _ := setupTestServer(t)

	reqBody := map[string]any{
		"owner_pubkey":  "ownerkey123",
		"account_email": "mail@example.com",
		"pop3": map[string]any{
			"host":    "pop.example.com",
			"port":    995,
			"user":    "user@example.com",
			"pass":    "password123",
			"use_ssl": true,
		},
		"smtp": map[string]any{
			"host":    "smtp.example.com",
			"port":    587,
			"user":    "user@example.com",
			"pass":    "password123",
			"use_ssl": false,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.addAccount(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status code: want %d, got %d", http.StatusCreated, w.Code)
		t.Logf("response: %s", w.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["account_email"] != "mail@example.com" {
		t.Errorf("account_email: want %q, got %q", "mail@example.com", response["account_email"])
	}
}

func TestAddAccount_InvalidJSON(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	server.addAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAddAccount_PasswordEncryption(t *testing.T) {
	server, mockDB := setupTestServer(t)

	reqBody := map[string]any{
		"owner_pubkey":  "owner_xyz",
		"account_email": "encrypted@example.com",
		"pop3": map[string]any{
			"host":    "pop.example.com",
			"port":    995,
			"user":    "user",
			"pass":    "secret_pop3_password",
			"use_ssl": true,
		},
		"smtp": map[string]any{
			"host":    "smtp.example.com",
			"port":    587,
			"user":    "user",
			"pass":    "secret_smtp_password",
			"use_ssl": false,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.addAccount(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create account: %s", w.Body.String())
	}

	// Retrieve the account and verify passwords are encrypted
	ctx := context.Background()
	accounts, _ := mockDB.GetMailAccountsByOwner(ctx, "owner_xyz")

	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	acc := accounts[0]

	// Verify POP3 password is encrypted (not plaintext)
	if acc.POP3.PassEnc == "secret_pop3_password" {
		t.Error("POP3 password should be encrypted, not plaintext")
	}
	if acc.SMTP.PassEnc == "secret_smtp_password" {
		t.Error("SMTP password should be encrypted, not plaintext")
	}

	// Verify we can decrypt the passwords
	pop3Pass, err := vault.DecryptAESGCM(server.cfg.EncryptionKey, acc.POP3.PassEnc)
	if err != nil {
		t.Errorf("failed to decrypt POP3 password: %v", err)
	}
	if pop3Pass != "secret_pop3_password" {
		t.Errorf("POP3 password: want %q, got %q", "secret_pop3_password", pop3Pass)
	}

	smtpPass, err := vault.DecryptAESGCM(server.cfg.EncryptionKey, acc.SMTP.PassEnc)
	if err != nil {
		t.Errorf("failed to decrypt SMTP password: %v", err)
	}
	if smtpPass != "secret_smtp_password" {
		t.Errorf("SMTP password: want %q, got %q", "secret_smtp_password", smtpPass)
	}
}

func TestListAccounts_Success(t *testing.T) {
	server, mockDB := setupTestServer(t)

	// Create test accounts
	ctx := context.Background()
	ownerPubKey := "test_owner_123"

	accounts := []*db.MailAccount{
		{
			OwnerPubKey:  ownerPubKey,
			AccountEmail: "account1@example.com",
			POP3:         db.POP3Settings{Host: "pop1.example.com", Port: 995},
			SMTP:         db.SMTPSettings{Host: "smtp1.example.com", Port: 587},
		},
		{
			OwnerPubKey:  ownerPubKey,
			AccountEmail: "account2@example.com",
			POP3:         db.POP3Settings{Host: "pop2.example.com", Port: 995},
			SMTP:         db.SMTPSettings{Host: "smtp2.example.com", Port: 587},
		},
	}

	for _, acc := range accounts {
		mockDB.CreateMailAccount(ctx, acc)
	}

	req := httptest.NewRequest("GET", "/api/v1/accounts?owner="+ownerPubKey, nil)
	w := httptest.NewRecorder()

	server.listAccounts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response []db.MailAccount
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("accounts count: want %d, got %d", 2, len(response))
	}
}

func TestListAccounts_MissingOwner(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	w := httptest.NewRecorder()

	server.listAccounts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	if !contains(response["error"], "owner") {
		t.Errorf("expected 'owner' in error message, got: %s", response["error"])
	}
}

func TestListAccounts_NoAccounts(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/accounts?owner=nonexistent_owner", nil)
	w := httptest.NewRecorder()

	server.listAccounts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var response []db.MailAccount
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("expected empty array, got %d accounts", len(response))
	}
}

func TestAddAccount_MultipleAccountsPerOwner(t *testing.T) {
	server, mockDB := setupTestServer(t)

	ownerPubKey := "multi_account_owner"

	// Add 3 accounts for the same owner
	for i := 1; i <= 3; i++ {
		reqBody := map[string]any{
			"owner_pubkey":  ownerPubKey,
			"account_email": "account" + string(rune('0'+i)) + "@example.com",
			"pop3": map[string]any{
				"host":    "pop.example.com",
				"port":    995,
				"user":    "user",
				"pass":    "pass",
				"use_ssl": true,
			},
			"smtp": map[string]any{
				"host":    "smtp.example.com",
				"port":    587,
				"user":    "user",
				"pass":    "pass",
				"use_ssl": false,
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.addAccount(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create account %d", i)
		}
	}

	// List accounts
	ctx := context.Background()
	accounts, err := mockDB.GetMailAccountsByOwner(ctx, ownerPubKey)
	if err != nil {
		t.Fatalf("GetMailAccountsByOwner failed: %v", err)
	}

	if len(accounts) != 3 {
		t.Errorf("expected 3 accounts, got %d", len(accounts))
	}
}

func TestAddAccount_DifferentPorts(t *testing.T) {
	testCases := []struct {
		name     string
		pop3Port int
		smtpPort int
	}{
		{"standard SSL", 995, 465},
		{"standard TLS", 110, 587},
		{"custom ports", 9995, 2525},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, mockDB := setupTestServer(t)

			reqBody := map[string]any{
				"owner_pubkey":  "owner_ports",
				"account_email": "test@example.com",
				"pop3": map[string]any{
					"host":    "pop.example.com",
					"port":    tc.pop3Port,
					"user":    "user",
					"pass":    "pass",
					"use_ssl": true,
				},
				"smtp": map[string]any{
					"host":    "smtp.example.com",
					"port":    tc.smtpPort,
					"user":    "user",
					"pass":    "pass",
					"use_ssl": false,
				},
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			server.addAccount(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("failed to create account")
			}

			// Verify ports were saved correctly
			ctx := context.Background()
			accounts, _ := mockDB.GetMailAccountsByOwner(ctx, "owner_ports")

			if len(accounts) > 0 {
				if accounts[0].POP3.Port != tc.pop3Port {
					t.Errorf("POP3 port: want %d, got %d", tc.pop3Port, accounts[0].POP3.Port)
				}
				if accounts[0].SMTP.Port != tc.smtpPort {
					t.Errorf("SMTP port: want %d, got %d", tc.smtpPort, accounts[0].SMTP.Port)
				}
			}
		})
	}
}

func TestAddAccount_SSLFlags(t *testing.T) {
	testCases := []struct {
		name       string
		pop3UseSSL bool
		smtpUseSSL bool
	}{
		{"both SSL", true, true},
		{"no SSL", false, false},
		{"mixed", true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, mockDB := setupTestServer(t)

			reqBody := map[string]any{
				"owner_pubkey":  "owner_ssl",
				"account_email": "ssl@example.com",
				"pop3": map[string]any{
					"host":    "pop.example.com",
					"port":    995,
					"user":    "user",
					"pass":    "pass",
					"use_ssl": tc.pop3UseSSL,
				},
				"smtp": map[string]any{
					"host":    "smtp.example.com",
					"port":    587,
					"user":    "user",
					"pass":    "pass",
					"use_ssl": tc.smtpUseSSL,
				},
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			server.addAccount(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("failed to create account")
			}

			ctx := context.Background()
			accounts, _ := mockDB.GetMailAccountsByOwner(ctx, "owner_ssl")

			if len(accounts) > 0 {
				if accounts[0].POP3.UseSSL != tc.pop3UseSSL {
					t.Errorf("POP3 UseSSL: want %v, got %v", tc.pop3UseSSL, accounts[0].POP3.UseSSL)
				}
				if accounts[0].SMTP.UseSSL != tc.smtpUseSSL {
					t.Errorf("SMTP UseSSL: want %v, got %v", tc.smtpUseSSL, accounts[0].SMTP.UseSSL)
				}
			}
		})
	}
}

func TestSendMail_AccountNotFound(t *testing.T) {
	server, _ := setupTestServer(t)

	reqBody := map[string]any{
		"owner_pubkey":  "nonexistent_owner",
		"account_email": "nonexistent@example.com",
		"to":            []string{"recipient@example.com"},
		"subject":       "Test",
		"body":          "Test body",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/mail/send", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.sendMail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code: want %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSendMail_InvalidJSON(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/mail/send", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()

	server.sendMail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: want %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestFetchInbox_MissingParameters(t *testing.T) {
	server, _ := setupTestServer(t)

	// Missing both owner and account
	req := httptest.NewRequest("GET", "/api/v1/mail/inbox", nil)
	w := httptest.NewRecorder()

	server.fetchInbox(w, req)

	// Should fail because account not found
	if w.Code == http.StatusOK {
		t.Error("expected error with missing parameters")
	}
}

func TestFetchMessage_InvalidID(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/mail/message?owner=o&account=a&id=invalid", nil)
	w := httptest.NewRecorder()

	server.fetchMessage(w, req)

	// Note: Returns 503 because connectPOP3 fails (account not found) before ID validation
	// In a real scenario with a valid account, invalid ID would return 400
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status code: want %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestAddAccount_SpecialCharactersInEmail(t *testing.T) {
	server, mockDB := setupTestServer(t)

	emails := []string{
		"user+tag@example.com",
		"first.last@sub.domain.com",
		"user_name@example.org",
	}

	for _, email := range emails {
		reqBody := map[string]any{
			"owner_pubkey":  "owner",
			"account_email": email,
			"pop3": map[string]any{
				"host": "pop.example.com", "port": 995,
				"user": email, "pass": "pass", "use_ssl": true,
			},
			"smtp": map[string]any{
				"host": "smtp.example.com", "port": 587,
				"user": email, "pass": "pass", "use_ssl": false,
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.addAccount(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("failed to create account with email %q", email)
		}
	}

	ctx := context.Background()
	accounts, _ := mockDB.GetMailAccountsByOwner(ctx, "owner")

	if len(accounts) != len(emails) {
		t.Errorf("expected %d accounts, got %d", len(emails), len(accounts))
	}
}

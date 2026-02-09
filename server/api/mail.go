package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"mulamail/db"
	"mulamail/mail"
	"mulamail/vault"
)

// POST /api/v1/accounts
//
// Registers a new legacy mail account (POP3 + SMTP) for the given owner.
// Passwords are encrypted with AES-256-GCM before being stored.
func (s *Server) addAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OwnerPubKey  string `json:"owner_pubkey"`
		AccountEmail string `json:"account_email"`
		POP3         struct {
			Host   string `json:"host"`
			Port   int    `json:"port"`
			User   string `json:"user"`
			Pass   string `json:"pass"`
			UseSSL bool   `json:"use_ssl"`
		} `json:"pop3"`
		SMTP struct {
			Host   string `json:"host"`
			Port   int    `json:"port"`
			User   string `json:"user"`
			Pass   string `json:"pass"`
			UseSSL bool   `json:"use_ssl"`
		} `json:"smtp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pop3Enc, err := vault.EncryptAESGCM(s.cfg.EncryptionKey, req.POP3.Pass)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypt pop3 pass: "+err.Error())
		return
	}
	smtpEnc, err := vault.EncryptAESGCM(s.cfg.EncryptionKey, req.SMTP.Pass)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypt smtp pass: "+err.Error())
		return
	}

	acc := &db.MailAccount{
		OwnerPubKey:  req.OwnerPubKey,
		AccountEmail: req.AccountEmail,
		POP3: db.POP3Settings{
			Host: req.POP3.Host, Port: req.POP3.Port,
			User: req.POP3.User, PassEnc: pop3Enc, UseSSL: req.POP3.UseSSL,
		},
		SMTP: db.SMTPSettings{
			Host: req.SMTP.Host, Port: req.SMTP.Port,
			User: req.SMTP.User, PassEnc: smtpEnc, UseSSL: req.SMTP.UseSSL,
		},
	}
	if err := s.db.CreateMailAccount(r.Context(), acc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"account_email": acc.AccountEmail})
}

// GET /api/v1/accounts?owner=<pubkey>
func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		writeError(w, http.StatusBadRequest, "owner pubkey required")
		return
	}
	accs, err := s.db.GetMailAccountsByOwner(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, accs)
}

// ---------- shared POP3 helper ----------

// connectPOP3 loads the account from the DB, decrypts the password, connects,
// and authenticates.  The caller is responsible for calling client.Close().
func (s *Server) connectPOP3(r *http.Request) (*mail.POP3Client, error) {
	owner := r.URL.Query().Get("owner")
	account := r.URL.Query().Get("account")

	acc, err := s.db.GetMailAccount(r.Context(), owner, account)
	if err != nil {
		return nil, err
	}

	pass, err := vault.DecryptAESGCM(s.cfg.EncryptionKey, acc.POP3.PassEnc)
	if err != nil {
		return nil, err
	}

	client := mail.NewPOP3Client(mail.POP3Config{
		Host: acc.POP3.Host, Port: acc.POP3.Port,
		User: acc.POP3.User, Pass: pass, UseSSL: acc.POP3.UseSSL,
	})
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if err := client.Auth(); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

// GET /api/v1/mail/inbox?owner=<pubkey>&account=<email>&limit=<N>
//
// Connects to the POP3 server, lists messages, and fetches headers for the
// most recent ones (newest first).  Default limit is 20.
func (s *Server) fetchInbox(w http.ResponseWriter, r *http.Request) {
	client, err := s.connectPOP3(r)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	defer client.Close()

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, e := strconv.Atoi(l); e == nil && n > 0 {
			limit = n
		}
	}

	list, err := client.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "POP3 LIST: "+err.Error())
		return
	}

	// Take the tail of the list (POP3 indices ascend; newest = highest index).
	start := len(list) - limit
	if start < 0 {
		start = 0
	}
	recent := list[start:]

	// Fetch headers in reverse order so the response is newest-first.
	messages := make([]any, 0, len(recent))
	for i := len(recent) - 1; i >= 0; i-- {
		msg, err := client.Top(recent[i].ID, 0)
		if err != nil {
			continue // skip messages that fail
		}
		msg.Size = recent[i].Size
		messages = append(messages, msg)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account":  r.URL.Query().Get("account"),
		"total":    len(list),
		"messages": messages,
	})
}

// GET /api/v1/mail/message?owner=<pubkey>&account=<email>&id=<msg-id>
//
// Downloads the full raw message via RETR.
func (s *Server) fetchMessage(w http.ResponseWriter, r *http.Request) {
	client, err := s.connectPOP3(r)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	defer client.Close()

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}

	raw, err := client.Retrieve(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "POP3 RETR: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"raw": raw})
}

// POST /api/v1/mail/send
//
// Sends a message via the SMTP server associated with the given account.
func (s *Server) sendMail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OwnerPubKey  string   `json:"owner_pubkey"`
		AccountEmail string   `json:"account_email"`
		To           []string `json:"to"`
		Subject      string   `json:"subject"`
		Body         string   `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	acc, err := s.db.GetMailAccount(r.Context(), req.OwnerPubKey, req.AccountEmail)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	smtpPass, err := vault.DecryptAESGCM(s.cfg.EncryptionKey, acc.SMTP.PassEnc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decrypt: "+err.Error())
		return
	}

	client := mail.NewSMTPClient(mail.SMTPConfig{
		Host: acc.SMTP.Host, Port: acc.SMTP.Port,
		User: acc.SMTP.User, Pass: smtpPass, UseSSL: acc.SMTP.UseSSL,
	})
	defer client.Close()

	if err := client.Connect(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "SMTP connect: "+err.Error())
		return
	}
	if err := client.Handshake(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "SMTP handshake: "+err.Error())
		return
	}
	if err := client.Auth(); err != nil {
		writeError(w, http.StatusUnauthorized, "SMTP auth: "+err.Error())
		return
	}
	if err := client.Send(mail.SendRequest{
		From: req.AccountEmail, To: req.To,
		Subject: req.Subject, Body: req.Body,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "SMTP send: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

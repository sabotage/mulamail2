package api

import (
	"encoding/json"
	"net/http"

	"mulamail/blockchain"
	"mulamail/config"
	"mulamail/db"
	"mulamail/vault"
)

// Server wires together every dependency the HTTP handlers need.
type Server struct {
	db      db.DB
	solana  *blockchain.Client
	storage vault.Storage
	cfg     *config.Config
}

// NewRouter registers all routes and returns the top-level handler.
func NewRouter(dbClient db.DB, solana *blockchain.Client, storage vault.Storage, cfg *config.Config) http.Handler {
	s := &Server{db: dbClient, solana: solana, storage: storage, cfg: cfg}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", s.health)

	// Identity (email ↔ Solana pubkey)
	mux.HandleFunc("POST /api/v1/identity/create-tx", s.createIdentityTx)
	mux.HandleFunc("POST /api/v1/identity/register", s.registerIdentity)
	mux.HandleFunc("GET /api/v1/identity/resolve", s.resolveIdentity)

	// Legacy mail-account management
	mux.HandleFunc("POST /api/v1/accounts", s.addAccount)
	mux.HandleFunc("GET /api/v1/accounts", s.listAccounts)

	// Mail operations (POP3 fetch / SMTP send)
	mux.HandleFunc("GET /api/v1/mail/inbox", s.fetchInbox)
	mux.HandleFunc("GET /api/v1/mail/message", s.fetchMessage)
	mux.HandleFunc("POST /api/v1/mail/send", s.sendMail)

	return mux
}

// ---------- shared helpers ----------

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

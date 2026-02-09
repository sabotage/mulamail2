package api

import (
	"encoding/json"
	"net/http"

	"github.com/gagliardetto/solana-go"

	"mulamail/blockchain"
	"mulamail/db"
)

// POST /api/v1/identity/create-tx
//
// Creates an *unsigned* Solana memo transaction that the client will sign
// locally before submitting via /register.  The memo embeds a JSON payload
// that binds the email address to the signer's public key.
//
// Request:  { "email": "alice@example.com", "pubkey": "<base58>" }
// Response: { "transaction": "<base64 unsigned tx>" }
func (s *Server) createIdentityTx(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email  string `json:"email"`
		PubKey string `json:"pubkey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if req.Email == "" || req.PubKey == "" {
		writeError(w, http.StatusBadRequest, "email and pubkey are required")
		return
	}

	pubkey, err := solana.PublicKeyFromBase58(req.PubKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pubkey: "+err.Error())
		return
	}

	txB64, err := blockchain.CreateIdentityMemoTx(r.Context(), s.solana, pubkey, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create tx: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"transaction": txB64})
}

// POST /api/v1/identity/register
//
// Accepts the client-signed transaction, broadcasts it to Solana, and
// persists the identity mapping in MongoDB.
//
// Request:  { "email": "...", "pubkey": "...", "signed_tx": "<base64>" }
// Response: { "identity": {...}, "tx_hash": "<signature>" }
func (s *Server) registerIdentity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		PubKey   string `json:"pubkey"`
		SignedTx string `json:"signed_tx"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Email == "" || req.PubKey == "" || req.SignedTx == "" {
		writeError(w, http.StatusBadRequest, "email, pubkey and signed_tx are required")
		return
	}

	// Duplicate guard.
	if _, err := s.db.GetIdentityByEmail(r.Context(), req.Email); err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}

	// Broadcast to Solana.
	sig, err := s.solana.SendTransaction(r.Context(), req.SignedTx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "broadcast: "+err.Error())
		return
	}

	// Persist.
	identity := &db.Identity{
		Email:    req.Email,
		PubKey:   req.PubKey,
		TxHash:   sig.String(),
		Verified: true,
	}
	if err := s.db.CreateIdentity(r.Context(), identity); err != nil {
		writeError(w, http.StatusInternalServerError, "store identity: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"identity": identity,
		"tx_hash":  sig.String(),
	})
}

// GET /api/v1/identity/resolve?email=...  OR  ?pubkey=...
//
// Looks up the stored identity mapping by either field.
func (s *Server) resolveIdentity(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	pubkey := r.URL.Query().Get("pubkey")
	if email == "" && pubkey == "" {
		writeError(w, http.StatusBadRequest, "provide email or pubkey query parameter")
		return
	}

	var (
		identity *db.Identity
		err      error
	)
	if email != "" {
		identity, err = s.db.GetIdentityByEmail(r.Context(), email)
	} else {
		identity, err = s.db.GetIdentityByPubKey(r.Context(), pubkey)
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "identity not found")
		return
	}
	writeJSON(w, http.StatusOK, identity)
}

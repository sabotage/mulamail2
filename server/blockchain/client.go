package blockchain

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Client wraps the Solana RPC endpoint used by MulaMail.
type Client struct {
	RPC *rpc.Client
}

func NewClient(rpcURL string) *Client {
	return &Client{RPC: rpc.New(rpcURL)}
}

// SendTransaction broadcasts a base64-encoded, already-signed transaction
// and returns its signature (transaction ID).
func (c *Client) SendTransaction(ctx context.Context, signedTxBase64 string) (solana.Signature, error) {
	tx, err := solana.TransactionFromBase64(signedTxBase64)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("parse tx: %w", err)
	}
	sig, err := c.RPC.SendTransaction(ctx, tx)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("send tx: %w", err)
	}
	return sig, nil
}

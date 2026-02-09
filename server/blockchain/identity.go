package blockchain

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// MemoV2ProgramID is the address of the Solana Memo v2 program.
var MemoV2ProgramID = solana.MustPublicKeyFromBase58("MemoSq4gqABAXKbbz9qDC7y18fHFoqnuGc2DUCfEJTg")

// memoInstruction implements solana.Instruction for a Memo v2 write.
type memoInstruction struct {
	memo   string
	signer solana.PublicKey
}

func (i *memoInstruction) ProgramID() solana.PublicKey { return MemoV2ProgramID }

func (i *memoInstruction) Accounts() []*solana.AccountMeta {
	return []*solana.AccountMeta{
		{
			PublicKey:  i.signer,
			IsSigner:  true,
			IsWritable: false,
		},
	}
}

func (i *memoInstruction) Data() ([]byte, error) { return []byte(i.memo), nil }

// CreateIdentityMemoTx builds an *unsigned* memo transaction that anchors the
// email↔pubkey mapping.  The returned base64 string is meant to be sent to
// the client, signed there, and submitted back via SendTransaction.
func CreateIdentityMemoTx(ctx context.Context, c *Client, pubkey solana.PublicKey, email string) (string, error) {
	latest, err := c.RPC.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("get blockhash: %w", err)
	}

	memoText := fmt.Sprintf(`{"action":"identity","email":"%s","pubkey":"%s"}`, email, pubkey.String())

	tx, err := solana.NewTransaction(
		[]solana.Instruction{&memoInstruction{memo: memoText, signer: pubkey}},
		latest.Value.Blockhash,
		solana.TransactionPayer(pubkey),
	)
	if err != nil {
		return "", fmt.Errorf("new tx: %w", err)
	}

	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("marshal tx: %w", err)
	}
	return base64.StdEncoding.EncodeToString(txBytes), nil
}

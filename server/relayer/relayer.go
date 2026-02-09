package relayer

// Relayer sponsors Solana transaction fees on behalf of MulaMail users,
// implementing the "Fee Payer Model" described in whitepaper §V.3.
//
// Phase 1 stub — Phase 2 will load a funded keypair, intercept unsigned
// transactions from the identity and mail flows, attach the fee-payer
// account, and broadcast.

// Relayer holds the state needed to sponsor fees.
type Relayer struct{}

// New returns a new Relayer instance.
func New() *Relayer { return &Relayer{} }

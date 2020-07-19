package distributed

import "github.com/dbogatov/dac-lib/dac"

// KeysHolder ...
type KeysHolder struct {
	pk dac.PK
	sk dac.SK
}

// CredentialsHolder ...
type CredentialsHolder struct {
	KeysHolder
	credentials dac.Credentials
	id          int
	kind        string
}

// CredRequest ...
type CredRequest struct {
	Request []byte
	ID      int
}

// Credentials ...
type Credentials struct {
	Creds []byte
}

// NonRevocationRequest ...
type NonRevocationRequest struct {
	PK []byte
}

// NonRevocationHandle ...
type NonRevocationHandle struct {
	Handle []byte
}

// TransactionProposal ...
type TransactionProposal struct {
	Hash       []byte
	AuthorID   int // for checking auditing correctness
	Chaincode  string
	Signature  []byte // dac.NymSignature
	Author     []byte // marshalled dac.Proof
	PkNym      []byte
	IndexValue []byte
}

// Transaction ...
type Transaction struct {
	Signature          []byte //dac.NymSignature
	Proposal           TransactionProposal
	AuditProof         []byte // dac.AuditingProof
	AuditEnc           []byte // dac.AuditingEncryption
	Endorsements       []Endorsement
	NonRevocationProof []byte // dac.RevocationProof
	Epoch              int
	AuthorPK           []byte
}

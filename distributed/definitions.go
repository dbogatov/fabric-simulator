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

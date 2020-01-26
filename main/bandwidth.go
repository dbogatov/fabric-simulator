package main

import "github.com/dbogatov/dac-lib/dac"

type transferable interface {
	size() int
	name() string
}

func recordBandwidth(from, to string, object transferable) {
	// TODO
	log.Infof("%s sent %d bytes of %s to %s\n", from, object.size(), object.name(), to)
}

/// Credentials

// Credentials ...
type Credentials struct {
	*dac.Credentials
}

func (creds Credentials) size() int {
	return len(creds.ToBytes())
}

func (creds Credentials) name() string {
	return "credentials"
}

/// CredRequest

// CredRequest ...
type CredRequest struct {
	*dac.CredRequest
}

func (credReq CredRequest) size() int {
	return len(credReq.ToBytes())
}

func (credReq CredRequest) name() string {
	return "cred-request"
}

/// Nonce

// Nonce ...
type Nonce struct {
	bytes []byte
}

func (nonce Nonce) size() int {
	return len(nonce.bytes)
}

func (nonce Nonce) name() string {
	return "nonce"
}

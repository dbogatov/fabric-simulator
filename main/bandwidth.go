package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/dbogatov/dac-lib/dac"
)

type transferable interface {
	size() int
	name() string
}

var bandwidthLoggingMutex = &sync.Mutex{}

func recordBandwidth(from, to string, object transferable) {
	start := time.Now()
	time.Sleep(time.Duration((float64(object.size()) / float64(sysParams.bandwidth))) * time.Second)
	end := time.Now()

	bandwidthLoggingMutex.Lock()
	event, err := json.Marshal(NetworkEvent{
		From:   from,
		To:     to,
		Object: object.name(),
		Size:   object.size(),
		Start:  start.Format(time.RFC3339Nano),
		End:    end.Format(time.RFC3339Nano),
	})
	if err != nil {
		panic(err)
	}
	log.Printf("%s,\n", string(event))

	logger.Debugf("%s sent %d bytes of %s to %s\n", from, object.size(), object.name(), to)
	bandwidthLoggingMutex.Unlock()
}

// NetworkEvent ...
type NetworkEvent struct {
	From   string
	To     string
	Object string
	Size   int
	Start  string
	End    string
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

/// TransactionProposal

func (tp TransactionProposal) size() int {
	return tp.payloadSize
}

func (tp TransactionProposal) name() string {
	return "transaction-proposal"
}

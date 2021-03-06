package simulator

import (
	"sync"
	"time"
)

// CryptoEvent ...
type CryptoEvent string

const (
	credDelegation CryptoEvent = "cred-delegate"
	credProve      CryptoEvent = "cred-prove"
	credVerify     CryptoEvent = "cred-verify"

	nonRevokeGrant  CryptoEvent = "non-revoke-grant"
	nonRevokeProve  CryptoEvent = "non-revoke-prove"
	nonRevokeVerify CryptoEvent = "non-revoke-verify"

	auditEncrypt CryptoEvent = "audit-enc"
	auditDecrypt CryptoEvent = "audit-dec"
	auditProve   CryptoEvent = "audit-prove"
	auditVerify  CryptoEvent = "audit-verify"

	sha3hash CryptoEvent = "hash"

	signNym   CryptoEvent = "sign-nym"
	verifyNym CryptoEvent = "verify-nym"

	signSchnorr   CryptoEvent = "sign-schnorr"
	verifySchnorr CryptoEvent = "verify-schnorr"
)

var recordCryptoEventLock = &sync.Mutex{}

func recordCryptoEvent(event CryptoEvent) {
	recordCryptoEventLock.Lock()
	defer recordCryptoEventLock.Unlock()

	if current, exists := execParams.cryptoEvents[event]; exists {
		execParams.cryptoEvents[event] = current + 1
	} else {
		execParams.cryptoEvents[event] = 1
	}
}

// TransactionTimingInfo ...
type TransactionTimingInfo struct {
	start time.Time
	end   time.Time

	endorsementsStart time.Time
	endorsementsEnd   time.Time

	validationStart time.Time
	validationEnd   time.Time
}

var recordTransactionTimingInfoLock = &sync.Mutex{}

func recordTransactionTimingInfo(info TransactionTimingInfo) {

	recordTransactionTimingInfoLock.Lock()
	defer recordTransactionTimingInfoLock.Unlock()

	execParams.transactionTimings = append(execParams.transactionTimings, info)
}

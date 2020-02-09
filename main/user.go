package main

import (
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"gonum.org/v1/gonum/stat/distuv"
)

// User ...
type User struct {
	CredentialsHolder
	nonRevocationHandler *dac.GrothSignature
	revocationPK         dac.PK
	epoch                int
	org                  int
	poisson              distuv.Poisson
}

func (user *User) submitTransaction(message string) {

	logger.Infof("user-%d starts transaction with a message %s", user.id, message)

	timingInfo := TransactionTimingInfo{
		start: time.Now(),
	}

	prg := newRand()

	hash := sha3([]byte(message))
	endorsers := make([]int, sysParams.endorsements)

	endorser := peerByHash(sha3([]byte(message)), sysParams.peers)

	for peer := 0; peer < sysParams.endorsements; peer++ {
		endorsers[peer] = (endorser + peer) % sysParams.peers
	}

	proposal, pkNym, skNym := MakeTransactionProposal(hash, *user)
	timingInfo.endorsementsStart = time.Now()
	for _, endorser := range endorsers {
		sysParams.network.peers[endorser].endorsementChannel <- proposal
	}

	schnorr := dac.MakeSchnorr(prg, false)
	endorsements := make([]Endorsement, 0)
	for i := 0; i < sysParams.endorsements; i++ {
		endorsement := <-proposal.doneChannel
		if e := schnorr.Verify(sysParams.network.peers[endorsement.endorser].pk, endorsement.signature, proposal.getMessage()); e != nil {
			panic(e)
		}
		recordCryptoEvent(verifySchnorr)
		endorsements = append(endorsements, endorsement)
	}

	timingInfo.endorsementsEnd = time.Now()

	logger.Debugf("%s has got all endorsements", user.name())

	if sysParams.revoke {
		if user.epoch != sysParams.network.epoch {
			logger.Debugf("user-%d (%s) detected epoch change; requesting new handle...", user.id, message)
			user.epoch = sysParams.network.epoch
			user.requestNonRevocation()
		}
	}

	tx := &Transaction{
		signature:    dac.SignNym(prg, pkNym, skNym, user.sk, sysParams.h, proposal.getMessage()), // ideally we add endorsements here but its fine for simulations
		proposal:     *proposal,
		endorsements: endorsements,
		epoch:        user.epoch,
		doneChannel:  make(chan bool, sysParams.peers), // need to receive OK from all peers (50%+1, technically)
	}
	recordCryptoEvent(signNym)

	if sysParams.revoke {
		tx.nonRevocationProof = dac.RevocationProve(prg, *user.nonRevocationHandler, user.sk, skNym, FP256BN.NewBIGint(user.epoch), sysParams.h, sysParams.ys[0])
		recordCryptoEvent(nonRevokeProve)
	}

	if sysParams.audit {

		// fresh auditing encryption and proof every transaction
		auditEnc, auditR := dac.AuditingEncrypt(newRand(), sysParams.network.auditor.pk, user.pk)
		recordCryptoEvent(auditEncrypt)

		tx.auditEnc = auditEnc
		tx.auditProof = dac.AuditingProve(prg, auditEnc, user.pk, user.sk, pkNym, skNym, sysParams.network.auditor.pk, auditR, sysParams.h)
		recordCryptoEvent(auditProve)
	}

	orderer := peerByHash(sha3([]byte(fmt.Sprintf("%s-order", message))), sysParams.peers)
	timingInfo.validationStart = time.Now()
	sysParams.network.peers[orderer].orderingChannel <- tx

	// wait for all peers to commit the transaction
	for peer := 0; peer < sysParams.peers; peer++ {
		<-tx.doneChannel
	}

	timingInfo.validationEnd = time.Now()
	timingInfo.end = time.Now()
	recordTransactionTimingInfo(timingInfo)

	sysParams.network.recordTransaction(tx)

	logger.Infof("%s transaction completed", user.name())
}

func (user *User) requestNonRevocation() {

	nrr := &NonRevocationRequest{
		userPk:      user.revocationPK,
		doneChannel: make(chan *NonRevocationHandle),
	}
	sysParams.network.revocationAuthority.requestChannel <- nrr

	user.nonRevocationHandler = &(<-nrr.doneChannel).handle
}

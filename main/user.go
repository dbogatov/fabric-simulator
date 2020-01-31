package main

import (
	"fmt"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

// User ...
type User struct {
	CredentialsHolder
	nonRevocationHandler *dac.GrothSignature
	epoch                int
	id                   int
	org                  int
}

// MakeUser ...
func MakeUser(credHolder CredentialsHolder, id, org int) (user *User) {
	// TODO
	user = &User{
		CredentialsHolder: credHolder,
		// revocationPK:      dac.PK,
		id:  id,
		org: org,
	}

	return
}

func (user *User) submitTransaction(message string) {

	prg := amcl.NewRAND()

	hash := sha3([]byte(message))
	endorsers := make([]int, sysParams.endorsements)

	endorser := peerByHash(sha3([]byte(message)), sysParams.peers)

	for peer := 0; peer < sysParams.endorsements; peer++ {
		endorsers[peer] = (endorser + peer) % sysParams.peers
	}

	proposal, pkNym, skNym := MakeTransactionProposal(hash, *user)
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
		endorsements = append(endorsements, endorsement)
	}

	logger.Debugf("%s has got all endorsements", user.name)

	if sysParams.revoke {
		if user.epoch != sysParams.network.epoch {
			user.epoch = sysParams.network.epoch
			user.requestNonRevocation()
		}
	}

	tx := &Transaction{
		payloadSize:  200,                                                                         // TODO
		signature:    dac.SignNym(prg, pkNym, skNym, user.sk, sysParams.h, proposal.getMessage()), // ideally we add endorsements here but its fine for simulations
		proposal:     *proposal,
		endorsements: endorsements,
		epoch:        user.epoch,
		doneChannel:  make(chan bool, sysParams.peers), // need to receive OK from all peers (50%+1, technically)
	}

	if sysParams.revoke {
		tx.nonRevocationProof = dac.RevocationProve(prg, *user.nonRevocationHandler, user.sk, skNym, FP256BN.NewBIGint(user.epoch), sysParams.h, sysParams.ys[0]) // TODO check ys
	}

	if sysParams.audit {

		// fresh auditing encryption and proof every transaction
		auditEnc, auditR := dac.AuditingEncrypt(amcl.NewRAND(), sysParams.network.auditor.pk, user.pk)

		tx.auditEnc = auditEnc
		tx.auditProof = dac.AuditingProve(prg, auditEnc, user.pk, user.sk, pkNym, skNym, sysParams.network.auditor.pk, auditR, sysParams.h)
	}

	orderer := peerByHash(sha3([]byte(fmt.Sprintf("%s-order", message))), sysParams.peers)
	sysParams.network.peers[orderer].orderingChannel <- tx

	// wait for all peers to commit the transaction
	for peer := 0; peer < sysParams.peers; peer++ {
		<-tx.doneChannel
	}

	sysParams.network.recordTransaction(tx)

	logger.Infof("%s transaction completed", user.name)
}

func (user *User) requestNonRevocation() {

	nrr := &NonRevocationRequest{
		userPk:      user.pk,
		doneChannel: make(chan *NonRevocationHandle),
	}
	sysParams.network.revocationAuthority.requestChannel <- nrr

	user.nonRevocationHandler = &(<-nrr.doneChannel).handle
}

package main

import (
	"fmt"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

// User ...
type User struct {
	CredentialsHolder
	id  int
	org int
}

// MakeUser ...
func MakeUser(credHolder CredentialsHolder, id, org int) (user *User) {

	user = &User{
		CredentialsHolder: credHolder,
		id:                id,
		org:               org,
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

	// fresh auditing encryption and proof every transaction
	auditEnc, auditR := dac.AuditingEncrypt(amcl.NewRAND(), sysParams.network.auditor.pk, user.pk)

	tx := &Transaction{
		payloadSize:  200,                                                                         // TODO
		signature:    dac.SignNym(prg, pkNym, skNym, user.sk, sysParams.h, proposal.getMessage()), // ideally we add endorsements here but its fine for simulations
		proposal:     *proposal,
		auditEnc:     auditEnc,
		auditProof:   dac.AuditingProve(prg, auditEnc, user.pk, user.sk, pkNym, skNym, sysParams.network.auditor.pk, auditR, sysParams.h),
		endorsements: endorsements,
		doneChannel:  make(chan bool, sysParams.peers), // need to receive OK from all peers (50%+1, technically)
	}

	orderer := peerByHash(sha3([]byte(fmt.Sprintf("%s-order", message))), sysParams.peers)
	sysParams.network.peers[orderer].orderingChannel <- tx

	// wait for all peers to commit the transaction
	for peer := 0; peer < sysParams.peers; peer++ {
		<-tx.doneChannel
	}

	logger.Infof("%s transaction completed", user.name)
}

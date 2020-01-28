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
	prg *amcl.RAND
}

func MakeUser(credHolder CredentialsHolder, id, org int, seed []byte) (user *User) {
	prg := amcl.NewRAND()
	prg.Clean()
	prg.Seed(len(seed), seed)

	user = &User{
		CredentialsHolder: credHolder,
		id:                id,
		org:               org,
		prg:               prg,
	}

	return
}

func (user *User) submitTransaction(message string) {

	hash := sha3([]byte(message))
	endorsers := make([]int, sysParams.endorsements)

	endorser := peerByHash(sha3([]byte(message)), sysParams.peers)

	for peer := 0; peer < sysParams.endorsements; peer++ {
		endorsers[peer] = (endorser + peer) % sysParams.peers
	}

	proposal, pkNym, skNym := MakeTransactionProposal(user.prg, hash, *user)
	for _, endorser := range endorsers {
		sysParams.network.peers[endorser].endorsementChannel <- proposal
	}

	endorsements := make([]Endorsement, 0)
	for i := 0; i < sysParams.endorsements; i++ {
		endorsements = append(endorsements, <-proposal.doneChannel)
	}

	logger.Infof("%s has go all endorsements", user.name)

	tx := &Transaction{
		payloadSize:  200,                                                                              // TODO
		signature:    dac.SignNym(user.prg, pkNym, skNym, user.sk, sysParams.h, proposal.getMessage()), // ideally we add endorsements here but its fine for simulations
		proposal:     *proposal,
		endorsements: endorsements,
		doneChannel:  make(chan bool, sysParams.peers), // need to receive OK from all peers (50%+1, technically)
	}

	orderer := peerByHash(sha3([]byte(fmt.Sprintf("%s-order", message))), sysParams.peers)
	sysParams.network.peers[orderer].orderingChannel <- tx

	// TODO
	<-tx.doneChannel

	logger.Infof("%s transaction completed", user.name)
}

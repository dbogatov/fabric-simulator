package main

import (
	"fmt"
	"math/big"

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

	hash := sha3(message)
	endorsers := make([]int, sysParams.endorsements)

	for peer := 0; peer < sysParams.endorsements; peer++ {
		endorsers[peer] = peerByHash(sha3(fmt.Sprintf("%s-%d", message, peer)), sysParams.endorsements)
	}

	proposal := MakeTransactionProposal(user.prg, hash, *user)
	for _, endorser := range endorsers {
		sysParams.network.peers[endorser].tpChannel <- proposal
	}

	endorsements := make([]Endorsement, 0)
	for i := 0; i < sysParams.endorsements; i++ {
		endorsements = append(endorsements, <-proposal.doneChannel)
	}

	logger.Infof("%s has go all endorsements", user.name)

	// TODO
}

func peerByHash(hash []byte, peers int) (peer int) {
	input := new(big.Int)
	input.SetBytes(hash)

	divisor := new(big.Int)
	divisor.SetInt64(int64(peers))

	result := new(big.Int)
	result = result.Mod(input, divisor)

	peer = int(result.Int64())

	return
}

func sha3(message string) (hash []byte) {

	hash = make([]byte, 32)
	raw := []byte(message)
	sha3 := amcl.NewSHA3(amcl.SHA3_HASH256)
	for i := 0; i < len(raw); i++ {
		sha3.Process(raw[i])
	}
	sha3.Hash(hash[:])

	return
}

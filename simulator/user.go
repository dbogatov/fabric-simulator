package simulator

import (
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/dbogatov/fabric-simulator/helpers"
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

	prg := helpers.NewRand()

	hash := helpers.Sha3([]byte(message))
	recordCryptoEvent(sha3hash)
	endorsers := make([]int, sysParams.Endorsements)

	endorser := helpers.PeerByHash(helpers.Sha3([]byte(message)), sysParams.Peers)
	recordCryptoEvent(sha3hash)

	for peer := 0; peer < sysParams.Endorsements; peer++ {
		endorsers[peer] = (endorser + peer) % sysParams.Peers
	}

	proposal, pkNym, skNym := MakeTransactionProposal(hash, *user)
	timingInfo.endorsementsStart = time.Now()
	for _, endorser := range endorsers {
		execParams.network.peers[endorser].endorsementChannel <- proposal
	}

	schnorr := dac.MakeSchnorr(prg, false)
	endorsements := make([]Endorsement, 0)
	for i := 0; i < sysParams.Endorsements; i++ {
		endorsement := <-proposal.doneChannel
		if e := schnorr.Verify(execParams.network.peers[endorsement.endorser].pk, endorsement.signature, proposal.getMessage()); e != nil {
			panic(e)
		}
		recordCryptoEvent(verifySchnorr)
		endorsements = append(endorsements, endorsement)
	}

	timingInfo.endorsementsEnd = time.Now()

	logger.Debugf("%s has got all endorsements", user.name())

	if sysParams.Revoke {
		if user.epoch != execParams.network.epoch {
			logger.Debugf("user-%d (%s) detected epoch change; requesting new handle...", user.id, message)
			user.epoch = execParams.network.epoch
			user.requestNonRevocation()
		}
	}

	tx := &Transaction{
		signature:    dac.SignNym(prg, pkNym, skNym, user.sk, sysParams.H, proposal.getMessage()), // ideally we add endorsements here but its fine for simulations
		proposal:     *proposal,
		endorsements: endorsements,
		epoch:        user.epoch,
		doneChannel:  make(chan bool, sysParams.Peers), // need to receive OK from all peers (50%+1, technically)
	}
	recordCryptoEvent(signNym)

	if sysParams.Revoke {
		tx.nonRevocationProof = dac.RevocationProve(prg, *user.nonRevocationHandler, user.sk, skNym, FP256BN.NewBIGint(user.epoch), sysParams.H, sysParams.Ys[0])
		recordCryptoEvent(nonRevokeProve)
	}

	if sysParams.Audit {

		// fresh auditing encryption and proof every transaction
		auditEnc, auditR := dac.AuditingEncrypt(helpers.NewRand(), execParams.network.auditor.pk, user.pk)
		recordCryptoEvent(auditEncrypt)

		tx.auditEnc = auditEnc
		tx.auditProof = dac.AuditingProve(prg, auditEnc, user.pk, user.sk, pkNym, skNym, execParams.network.auditor.pk, auditR, sysParams.H)
		recordCryptoEvent(auditProve)
	}

	orderer := helpers.PeerByHash(helpers.Sha3([]byte(fmt.Sprintf("%s-order", message))), sysParams.Peers)
	recordCryptoEvent(sha3hash)
	timingInfo.validationStart = time.Now()
	execParams.network.peers[orderer].orderingChannel <- tx

	// wait for all peers to commit the transaction
	for peer := 0; peer < sysParams.Peers; peer++ {
		<-tx.doneChannel
	}

	timingInfo.validationEnd = time.Now()
	timingInfo.end = time.Now()
	recordTransactionTimingInfo(timingInfo)

	execParams.network.recordTransaction(tx)

	logger.Infof("%s transaction completed", user.name())
}

func (user *User) requestNonRevocation() {

	nrr := &NonRevocationRequest{
		userPk:      user.revocationPK,
		doneChannel: make(chan *NonRevocationHandle),
	}
	execParams.network.revocationAuthority.requestChannel <- nrr

	user.nonRevocationHandler = &(<-nrr.doneChannel).handle
}

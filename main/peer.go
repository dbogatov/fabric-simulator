package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"golang.org/x/sync/semaphore"
)

type operation int

const (
	endorsement  operation = 0
	ordering     operation = 1
	verification operation = 2
)

// Peer ...
type Peer struct {
	KeysHolder

	id int

	ctx context.Context

	endorsementSemaphore *semaphore.Weighted
	validationSemaphore  *semaphore.Weighted

	endorsementChannel chan *TransactionProposal
	orderingChannel    chan *Transaction
	validationChannel  chan *Transaction
	exitChannel        chan bool

	cache map[operation][][32]byte
}

// MakePeer ...
func MakePeer(id int) (peer *Peer) {
	sk, pk := dac.GenerateKeys(amcl.NewRAND(), 0)

	peer = &Peer{
		id:                   id,
		ctx:                  context.TODO(),
		endorsementSemaphore: semaphore.NewWeighted(int64(sysParams.concurrentEndorsements)),
		validationSemaphore:  semaphore.NewWeighted(int64(sysParams.concurrentValidations)),
		endorsementChannel:   make(chan *TransactionProposal),
		orderingChannel:      make(chan *Transaction),
		validationChannel:    make(chan *Transaction),
		exitChannel:          make(chan bool),
		KeysHolder: KeysHolder{
			pk: pk,
			sk: sk,
		},
		cache: make(map[operation][][32]byte, 3),
	}
	for _, op := range []operation{endorsement, ordering, verification} {
		peer.cache[op] = make([][32]byte, 0)
	}

	go peer.run()

	return
}

func (peer *Peer) run() {
	for {
		select {
		case tp := <-peer.endorsementChannel:
			recordBandwidth(tp.from, fmt.Sprintf("peer-%d (endorser)", peer.id), tp)
			if e := peer.endorsementSemaphore.Acquire(peer.ctx, 1); e != nil {
				panic(e)
			}
			go peer.endorse(tp)
			continue
		case tx := <-peer.orderingChannel:
			recordBandwidth(tx.proposal.from, fmt.Sprintf("peer-%d (orderer)", peer.id), tx)
			go peer.order(tx)
			continue
		case tx := <-peer.validationChannel:
			if tx.orderer != peer.id {
				recordBandwidth(fmt.Sprintf("peer-%d (orderer)", tx.orderer), fmt.Sprintf("peer-%d", peer.id), tx)
			}
			if e := peer.validationSemaphore.Acquire(peer.ctx, 1); e != nil {
				panic(e)
			}
			go peer.validate(tx)
			continue
		case <-peer.exitChannel:
		}
		break
	}
}

func (peer *Peer) validate(tx *Transaction) {

	defer peer.validationSemaphore.Release(1)

	if e := tx.signature.VerifyNym(sysParams.h, tx.proposal.pkNym, tx.proposal.getMessage()); e != nil {
		panic(e)
	}

	if len(tx.endorsements) < sysParams.endorsements {
		panic("too few endorsements")
	}

	schnorr := dac.MakeSchnorr(amcl.NewRAND(), false)
	for _, endorsement := range tx.endorsements {
		if e := schnorr.Verify(sysParams.network.peers[endorsement.endorser].pk, endorsement.signature, tx.proposal.getMessage()); e != nil {
			panic(e)
		}
	}

	peer.validateIdentity(tx.proposal.author, tx.proposal.pkNym, tx.proposal.indices, verification)

	if e := tx.auditProof.Verify(tx.auditEnc, tx.proposal.pkNym, sysParams.network.auditor.pk, sysParams.h); e != nil {
		panic(e)
	}

	// somewhere here are read/write conflict check and ledger update
	// but they are negligible in comparison to crypto

	executeChaincode()

	tx.doneChannel <- true
}

func (peer *Peer) order(tx *Transaction) {

	peer.validateIdentity(tx.proposal.author, tx.proposal.pkNym, tx.proposal.indices, ordering)

	tx.orderer = peer.id

	for _, other := range sysParams.network.peers {
		other.validationChannel <- tx
	}

	logger.Debugf("Peer %d has ordered a transaction", peer.id)
}

func (peer *Peer) endorse(tp *TransactionProposal) {

	defer peer.endorsementSemaphore.Release(1)

	// Verify signature
	if e := tp.signature.VerifyNym(sysParams.h, tp.pkNym, tp.getMessage()); e != nil {
		panic(e)
	}
	// Verify author
	// Ideally should verify that tp.indices[0].Attribute is equal to the expected value that permits using the blockchain
	peer.validateIdentity(tp.author, tp.pkNym, tp.indices, endorsement)

	// Verify read / write permissions (should be cached)
	peer.validateIdentity(tp.author, tp.pkNym, tp.indices, endorsement)

	// Execute proposal
	executeChaincode()

	// All set!
	schnorr := dac.MakeSchnorr(amcl.NewRAND(), false)

	logger.Debugf("Peer %d endorsed transaction payload %s", peer.id, tp.from)
	endorsement := Endorsement{
		payloadSize: 200, // TODO
		signature:   schnorr.Sign(peer.sk, tp.getMessage()),
		endorser:    peer.id,
	}
	recordBandwidth(fmt.Sprintf("peer-%d", peer.id), tp.from, endorsement)

	tp.doneChannel <- endorsement
}

func (peer *Peer) validateIdentity(proof []byte, pkNym interface{}, indices dac.Indices, op operation) {

	var key [32]byte
	copy(key[:], sha3(proof)[:4])
	for _, cached := range peer.cache[op] {
		if cached == key {
			return
		}
	}
	proofObj := dac.ProofFromBytes(proof)
	if e := proofObj.VerifyProof(sysParams.rootPk, sysParams.ys, sysParams.h, pkNym, indices, []byte{}); e != nil {
		panic(e)
	}

	peer.cache[op] = append(peer.cache[op], key)
}

func executeChaincode() {
	// TODO
	time.Sleep(50 * time.Millisecond)
}

// Endorsement ...
type Endorsement struct {
	payloadSize int
	signature   dac.SchnorrSignature
	endorser    int
}

func (endorsement Endorsement) size() int {
	return endorsement.payloadSize
}

func (endorsement Endorsement) name() string {
	return "endorsement"
}

// Transaction ...
type Transaction struct {
	payloadSize  int // TODO
	signature    dac.NymSignature
	proposal     TransactionProposal
	auditProof   dac.AuditingProof
	auditEnc     dac.AuditingEncryption
	endorsements []Endorsement
	orderer      int
	doneChannel  chan bool
}

func (transaction Transaction) size() int {
	return transaction.payloadSize
}

func (transaction Transaction) name() string {
	return "transaction"
}

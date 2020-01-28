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

	endorsementChannel chan *TransactionProposal
	orderingChannel    chan *Transaction
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
		endorsementChannel:   make(chan *TransactionProposal),
		orderingChannel:      make(chan *Transaction),
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
			recordBandwidth(tp.from, fmt.Sprintf("peer-%d", peer.id), tp)
			if e := peer.endorsementSemaphore.Acquire(peer.ctx, 1); e != nil {
				panic(e)
			}
			go peer.endorse(tp)
			continue
		case tx := <-peer.orderingChannel:
			recordBandwidth(tx.proposal.from, fmt.Sprintf("peer-%d", peer.id), tx)
			go peer.order(tx)
			continue
		case <-peer.exitChannel:
		}
		break
	}
}

func (peer *Peer) order(tx *Transaction) {

	// TODO
	tx.doneChannel <- true

	logger.Debugf("Peer %d has ordered transaction", peer.id)
}

func (peer *Peer) endorse(tp *TransactionProposal) {

	defer peer.endorsementSemaphore.Release(1)

	// Verify signature
	if e := tp.signature.VerifyNym(sysParams.h, tp.pkNym, tp.getMessage()); e != nil {
		panic(e)
	}
	// Verify author
	// Ideally should verify that tp.indices[0].Attribute is equal to the expected value that permits using the blockchain
	peer.validate(tp.author, tp.pkNym, tp.indices, endorsement)

	// Verify read / write permissions (should be cached)
	peer.validate(tp.author, tp.pkNym, tp.indices, endorsement)

	// Execute proposal
	// TODO
	time.Sleep(50 * time.Millisecond)

	// All set!
	schnorr := dac.MakeSchnorr(amcl.NewRAND(), true)

	logger.Debugf("Peer %d endorsed transaction payload %s", peer.id, tp.from)
	endorsement := Endorsement{
		payloadSize: 200, // TODO
		signature:   schnorr.Sign(peer.sk, tp.getMessage()),
	}
	recordBandwidth(fmt.Sprintf("peer-%d", peer.id), tp.from, endorsement)

	tp.doneChannel <- endorsement
}

func (peer *Peer) validate(proof []byte, pkNym interface{}, indices dac.Indices, op operation) {

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

// Endorsement ...
type Endorsement struct {
	payloadSize int
	signature   dac.SchnorrSignature
}

func (endorsement Endorsement) size() int {
	return endorsement.payloadSize
}

func (endorsement Endorsement) name() string {
	return "endorsement"
}

// Transaction ...
type Transaction struct {
	payloadSize  int
	signature    dac.NymSignature
	proposal     TransactionProposal
	endorsements []Endorsement
	doneChannel  chan bool
}

func (transaction Transaction) size() int {
	return transaction.payloadSize
}

func (transaction Transaction) name() string {
	return "transaction"
}

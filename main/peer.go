package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"golang.org/x/sync/semaphore"
)

// Peer ...
type Peer struct {
	KeysHolder

	id int

	ctx context.Context

	endorsementSemaphore *semaphore.Weighted

	tpChannel   chan *TransactionProposal
	exitChannel chan bool
}

// MakePeer ...
func MakePeer(id int) (peer *Peer) {
	sk, pk := dac.GenerateKeys(amcl.NewRAND(), 0)

	peer = &Peer{
		id:                   id,
		ctx:                  context.TODO(),
		endorsementSemaphore: semaphore.NewWeighted(int64(sysParams.concurrentEndorsements)),
		tpChannel:            make(chan *TransactionProposal),
		exitChannel:          make(chan bool),
		KeysHolder: KeysHolder{
			pk: pk,
			sk: sk,
		},
	}

	go peer.run()

	return
}

func (peer *Peer) run() {
	for {
		select {
		case tp := <-peer.tpChannel:
			recordBandwidth(tp.from, fmt.Sprintf("peer-%d", peer.id), tp)
			if e := peer.endorsementSemaphore.Acquire(peer.ctx, 1); e != nil {
				panic(e)
			}
			go peer.endorse(tp)
			continue
		case <-peer.exitChannel:
		}
		break
	}
}

func (peer *Peer) endorse(tp *TransactionProposal) {

	defer peer.endorsementSemaphore.Release(1)

	// Verify signature
	if e := tp.signature.VerifyNym(sysParams.h, tp.pkNym, tp.getMessage()); e != nil {
		panic(e)
	}
	// Verify author
	proof := dac.ProofFromBytes(tp.author)
	// Ideally should verify that tp.indices[0].Attribute is equal to the expected value that permits using the blockchain
	if e := proof.VerifyProof(sysParams.rootPk, sysParams.ys, sysParams.h, tp.pkNym, tp.indices, []byte{}); e != nil {
		panic(e)
	}

	// If require read / write permission (which it always does here), check again
	if e := proof.VerifyProof(sysParams.rootPk, sysParams.ys, sysParams.h, tp.pkNym, tp.indices, []byte{}); e != nil {
		panic(e)
	}

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

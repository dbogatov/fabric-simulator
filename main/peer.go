package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
)

// Peer ...
type Peer struct {
	id int

	ctx context.Context

	endorsementSemaphore *semaphore.Weighted

	tpChannel   chan TransactionProposal
	exitChannel chan bool
}

// MakePeer ...
func MakePeer(id int) (peer *Peer) {
	peer = &Peer{
		id:                   id,
		ctx:                  context.TODO(),
		endorsementSemaphore: semaphore.NewWeighted(int64(sysParams.concurrentEndorsements)),
		tpChannel:            make(chan TransactionProposal),
		exitChannel:          make(chan bool),
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

func (peer *Peer) endorse(tp TransactionProposal) {

	defer peer.endorsementSemaphore.Release(1)

	// TODO
	time.Sleep(500 * time.Millisecond)

	logger.Debugf("Peer %d endorsed transaction payload %d", peer.id, tp.id)
	tp.doneChannel <- Endorsement{
		payloadSize: 200,
		signature:   tp.id * 3,
	}
}

// TransactionProposal ...
type TransactionProposal struct {
	payloadSize int
	id          int
	from        string
	doneChannel chan Endorsement
}

// Endorsement ...
type Endorsement struct {
	payloadSize int
	signature   int
}

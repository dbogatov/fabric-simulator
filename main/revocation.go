package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"golang.org/x/sync/semaphore"
)

// RevocationAuthority ...
type RevocationAuthority struct {
	KeysHolder

	semaphore *semaphore.Weighted
	ctx       context.Context

	requestChannel chan *NonRevocationRequest
	exitChannel    chan bool
}

// MakeRevocationAuthority ...
func MakeRevocationAuthority() (revocation *RevocationAuthority) {

	groth := dac.MakeGroth(newRand(), true, sysParams.ys[1])
	sk, pk := groth.Generate()

	revocation = &RevocationAuthority{
		semaphore:      semaphore.NewWeighted(int64(sysParams.concurrentRevocations)),
		ctx:            context.TODO(),
		requestChannel: make(chan *NonRevocationRequest),
		exitChannel:    make(chan bool),
		KeysHolder: KeysHolder{
			pk: pk,
			sk: sk,
		},
	}

	go revocation.run()

	return
}

func (revocation *RevocationAuthority) run() {
	for {
		select {
		case nrr := <-revocation.requestChannel:
			recordBandwidth(fmt.Sprintf("user-%d", nrr.userID), "revocation-authority", nrr)
			if e := revocation.semaphore.Acquire(revocation.ctx, 1); e != nil {
				panic(e)
			}

			go revocation.grant(nrr)
			continue
		case <-time.After(time.Duration(sysParams.epoch) * time.Second):

			if sysParams.revoke {
				sysParams.network.epoch++
			}

			continue
		case <-revocation.exitChannel:
		}
		break
	}
}

func (revocation *RevocationAuthority) grant(nrr *NonRevocationRequest) {

	defer revocation.semaphore.Release(1)

	nrh := &NonRevocationHandle{
		handle: dac.SignNonRevoke(newRand(), revocation.sk, nrr.userPk, FP256BN.NewBIGint(sysParams.network.epoch), sysParams.ys[1]),
	}

	recordBandwidth("revocation-authority", fmt.Sprintf("user-%d", nrr.userID), nrh)

	logger.Infof("Non-revocation granted to user-%d", nrr.userID)

	nrr.doneChannel <- nrh
}

// NonRevocationRequest ...
type NonRevocationRequest struct {
	userPk      dac.PK
	userID      int
	doneChannel chan *NonRevocationHandle
}

func (nrr NonRevocationRequest) size() int {
	return 25 // TODO
}

func (nrr NonRevocationRequest) name() string {
	return "non-revocation-request"
}

// NonRevocationHandle ...
type NonRevocationHandle struct {
	handle dac.GrothSignature
}

func (nrh NonRevocationHandle) size() int {
	return 25 // TODO
}

func (nrh NonRevocationHandle) name() string {
	return "non-revocation-handle"
}

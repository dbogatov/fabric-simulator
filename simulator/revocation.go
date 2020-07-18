package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/dbogatov/fabric-simulator/helpers"
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

	groth := dac.MakeGroth(helpers.NewRand(), true, sysParams.Ys[1])
	sk, pk := groth.Generate()

	revocation = &RevocationAuthority{
		semaphore:      semaphore.NewWeighted(int64(sysParams.ConcurrentRevocations)),
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
		case <-time.After(time.Duration(sysParams.Epoch) * time.Second):

			if sysParams.Revoke && execParams.network != nil {
				execParams.network.epoch++
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
		handle: dac.SignNonRevoke(helpers.NewRand(), revocation.sk, nrr.userPk, FP256BN.NewBIGint(execParams.network.epoch), sysParams.Ys[1]),
	}

	recordCryptoEvent(nonRevokeGrant)
	recordBandwidth("revocation-authority", fmt.Sprintf("user-%d", nrr.userID), nrh)

	logger.Debugf("Non-revocation granted to user-%d", nrr.userID)

	nrr.doneChannel <- nrh
}

// NonRevocationRequest ...
type NonRevocationRequest struct {
	userPk      dac.PK
	userID      int
	doneChannel chan *NonRevocationHandle
}

func (nrr NonRevocationRequest) size() int {
	return 1 + 2*32 + CertificateSize
}

func (nrr NonRevocationRequest) name() string {
	return "non-revocation-request"
}

// NonRevocationHandle ...
type NonRevocationHandle struct {
	handle dac.GrothSignature
}

func (nrh NonRevocationHandle) size() int {
	return 3*(1+2*32) + 4*32
}

func (nrh NonRevocationHandle) name() string {
	return "non-revocation-handle"
}

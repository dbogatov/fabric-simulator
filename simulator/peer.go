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
	sk, pk := dac.GenerateKeys(helpers.NewRand(), 0)

	peer = &Peer{
		id:                   id,
		ctx:                  context.TODO(),
		endorsementSemaphore: semaphore.NewWeighted(int64(sysParams.ConcurrentEndorsements)),
		validationSemaphore:  semaphore.NewWeighted(int64(sysParams.ConcurrentValidations)),
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
			recordBandwidth(fmt.Sprintf("user-%d", tp.authorID), fmt.Sprintf("peer-%d", peer.id), tp)
			if e := peer.endorsementSemaphore.Acquire(peer.ctx, 1); e != nil {
				panic(e)
			}
			go peer.endorse(tp)
			continue
		case tx := <-peer.orderingChannel:
			recordBandwidth(fmt.Sprintf("user-%d", tx.proposal.authorID), fmt.Sprintf("peer-%d", peer.id), tx)
			go peer.order(tx)
			continue
		case tx := <-peer.validationChannel:
			if tx.orderer != peer.id {
				recordBandwidth(fmt.Sprintf("peer-%d", tx.orderer), fmt.Sprintf("peer-%d", peer.id), tx)
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

	if e := tx.signature.VerifyNym(sysParams.H, tx.proposal.pkNym, tx.proposal.getMessage()); e != nil {
		panic(e)
	}
	recordCryptoEvent(verifyNym)

	if len(tx.endorsements) < sysParams.Endorsements {
		panic("too few endorsements")
	}

	schnorr := dac.MakeSchnorr(helpers.NewRand(), false)
	for _, endorsement := range tx.endorsements {
		if e := schnorr.Verify(execParams.network.peers[endorsement.endorser].pk, endorsement.signature, tx.proposal.getMessage()); e != nil {
			panic(e)
		}
	}
	recordCryptoEvent(verifySchnorr)

	peer.validateIdentity(tx.proposal.author, tx.proposal.pkNym, tx.proposal.indices, verification)

	if sysParams.Audit {
		if e := tx.auditProof.Verify(tx.auditEnc, tx.proposal.pkNym, execParams.network.auditor.pk, sysParams.H); e != nil {
			panic(e)
		}
		recordCryptoEvent(auditVerify)
	}

	if sysParams.Revoke {
		// Verify non-revocation
		if e := tx.nonRevocationProof.Verify(tx.proposal.pkNym, FP256BN.NewBIGint(tx.epoch), sysParams.H, execParams.network.revocationAuthority.pk, sysParams.Ys[1]); e != nil {
			panic(e)
		}
		recordCryptoEvent(nonRevokeVerify)
	}

	// somewhere here are read/write conflict check and ledger update
	// but they are negligible in comparison to crypto

	executeChaincode()

	tx.doneChannel <- true
}

func (peer *Peer) order(tx *Transaction) {

	peer.validateIdentity(tx.proposal.author, tx.proposal.pkNym, tx.proposal.indices, ordering)

	tx.orderer = peer.id

	for _, other := range execParams.network.peers {
		other.validationChannel <- tx
	}

	logger.Debugf("peer-%d has ordered a transaction", peer.id)
}

func (peer *Peer) endorse(tp *TransactionProposal) {

	defer peer.endorsementSemaphore.Release(1)

	// Verify signature
	if e := tp.signature.VerifyNym(sysParams.H, tp.pkNym, tp.getMessage()); e != nil {
		panic(e)
	}
	recordCryptoEvent(verifyNym)
	// Verify author
	// Ideally should verify that tp.indices[0].Attribute is equal to the expected value that permits using the blockchain
	peer.validateIdentity(tp.author, tp.pkNym, tp.indices, endorsement)

	// Verify read / write permissions (should be cached)
	peer.validateIdentity(tp.author, tp.pkNym, tp.indices, endorsement)

	// Execute proposal
	executeChaincode()

	// All set!
	schnorr := dac.MakeSchnorr(helpers.NewRand(), false)

	logger.Debugf("peer-%d endorsed transaction payload %s", peer.id, fmt.Sprintf("user-%d", tp.authorID))
	endorsement := Endorsement{
		signature: schnorr.Sign(peer.sk, tp.getMessage()),
		endorser:  peer.id,
	}
	recordCryptoEvent(signSchnorr)
	recordBandwidth(fmt.Sprintf("peer-%d", peer.id), fmt.Sprintf("user-%d", tp.authorID), endorsement)

	tp.doneChannel <- endorsement
}

func (peer *Peer) validateIdentity(proof []byte, pkNym interface{}, indices dac.Indices, op operation) {

	var key [32]byte
	copy(key[:], helpers.Sha3(proof)[:4])
	recordCryptoEvent(sha3hash)
	for _, cached := range peer.cache[op] {
		if cached == key {
			return
		}
	}
	proofObj := dac.ProofFromBytes(proof)
	if e := proofObj.VerifyProof(sysParams.RootPk, sysParams.Ys, sysParams.H, pkNym, indices, []byte{}); e != nil {
		panic(e)
	}
	recordCryptoEvent(credVerify)

	peer.cache[op] = append(peer.cache[op], key)
}

func executeChaincode() {
	// TODO
	time.Sleep(50 * time.Millisecond)
}

// Endorsement ...
type Endorsement struct {
	signature dac.SchnorrSignature
	endorser  int
}

func (endorsement Endorsement) size() int {
	// Schnorr(ECP2 + BIG) + endorser ID
	return 5*32 + CertificateSize
}

func (endorsement Endorsement) name() string {
	return "endorsement"
}

// Transaction ...
type Transaction struct {
	signature          dac.NymSignature
	proposal           TransactionProposal
	auditProof         dac.AuditingProof
	auditEnc           dac.AuditingEncryption
	endorsements       []Endorsement
	nonRevocationProof dac.RevocationProof
	epoch              int
	orderer            int
	doneChannel        chan bool
}

func (transaction Transaction) size() int {
	auditingSize := 0
	if sysParams.Audit {
		auditingSize = 4*32 + 2*4*32
	}
	revocationSize := 0
	if sysParams.Revoke {
		revocationSize = 3*32 + 3*(1+2*32) + 4*32 + 4
	}
	return len(transaction.signature.ToBytes()) + transaction.proposal.size() + auditingSize + len(transaction.endorsements)*transaction.endorsements[0].size() + revocationSize
}

func (transaction Transaction) name() string {
	return "transaction"
}

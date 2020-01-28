package main

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

// TransactionProposal ...
type TransactionProposal struct {
	payloadSize int
	hash        []byte
	from        string
	chaincode   string
	doneChannel chan Endorsement
	signature   dac.NymSignature
	author      []byte // marshalled dac.Proof
	pkNym       interface{}
	indices     dac.Indices
	// TODO auditing and revocation
}

func MakeTransactionProposal(prg *amcl.RAND, hash []byte, user User) (tp *TransactionProposal) {

	skNym, pkNym := dac.GenerateNymKeys(prg, user.sk, sysParams.h)
	indices := dac.Indices{
		dac.Index{
			I:         1,
			J:         1,
			Attribute: user.credentials.Attributes[1][1],
		},
	}

	proof, e := user.credentials.Prove(
		prg,
		user.sk,
		sysParams.rootPk,
		indices,
		[]byte{},
		sysParams.ys,
		sysParams.h,
		skNym,
	)

	if e != nil {
		panic(e)
	}
	author := proof.ToBytes()

	tp = &TransactionProposal{
		chaincode:   "chaincode: hash | policy: write",
		from:        user.name,
		hash:        hash,
		author:      author,
		pkNym:       pkNym,
		indices:     indices,
		doneChannel: make(chan Endorsement, sysParams.endorsements),
	}

	tp.signature = dac.SignNym(prg, pkNym, skNym, user.sk, sysParams.h, tp.getMessage())

	return
}

func (tp *TransactionProposal) getMessage() (message []byte) {

	message = make([]byte, 0)

	message = append(message, tp.hash...)
	message = append(message, []byte(tp.chaincode)...)
	message = append(message, []byte(tp.from)...)
	message = append(message, tp.author...)

	return
}

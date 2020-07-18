package simulator

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-simulator/helpers"
)

// TransactionProposal ...
type TransactionProposal struct {
	hash        []byte
	authorID    int // for checking auditing correctness
	chaincode   string
	doneChannel chan Endorsement
	signature   dac.NymSignature
	author      []byte // marshalled dac.Proof
	pkNym       interface{}
	indices     dac.Indices
}

// MakeTransactionProposal ...
func MakeTransactionProposal(hash []byte, user User) (tp *TransactionProposal, pkNym interface{}, skNym dac.SK) {

	prg := helpers.NewRand()

	skNym, pkNym = dac.GenerateNymKeys(prg, user.sk, sysParams.H)
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
		sysParams.RootPk,
		indices,
		[]byte{},
		sysParams.Ys,
		sysParams.H,
		skNym,
	)
	recordCryptoEvent(credProve)

	if e != nil {
		panic(e)
	}
	author := proof.ToBytes()

	tp = &TransactionProposal{
		chaincode:   "chaincode: hash | policy: write",
		authorID:    user.id,
		hash:        hash,
		author:      author,
		pkNym:       pkNym,
		indices:     indices,
		doneChannel: make(chan Endorsement, sysParams.Endorsements),
	}

	tp.signature = dac.SignNym(prg, pkNym, skNym, user.sk, sysParams.H, tp.getMessage())
	recordCryptoEvent(signNym)

	return
}

func (tp *TransactionProposal) getMessage() (message []byte) {

	message = make([]byte, 0)

	message = append(message, tp.hash...)
	message = append(message, []byte(tp.chaincode)...)
	message = append(message, byte(tp.authorID))
	message = append(message, tp.author...)

	return
}

func (tp TransactionProposal) size() int {
	// hash + chaincode + signature + proof + pkNym + attribute (value + 2 ints)
	return len(tp.hash) + len(tp.chaincode) + len(tp.signature.ToBytes()) + len(tp.author) + 4*32 + 4*32 + 2*4
}

func (tp TransactionProposal) name() string {
	return "transaction-proposal"
}

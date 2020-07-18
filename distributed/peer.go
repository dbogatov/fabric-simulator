package distributed

import (
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-simulator/helpers"
)

// RPCPeer ...
type RPCPeer struct {
	keys KeysHolder

	id int

	cache [][32]byte
}

// MakeRPCPeer ...
func MakeRPCPeer(prg *amcl.RAND, id int) (rpcPeer *RPCPeer) {
	sk, pk := dac.GenerateKeys(helpers.NewRand(), 0)

	rpcPeer = &RPCPeer{
		id: id,
		keys: KeysHolder{
			sk: sk,
			pk: pk,
		},
		cache: make([][32]byte, 0),
	}

	return
}

// Endorse ...
func (peer *RPCPeer) Endorse(args *TransactionProposal, reply *Endorsement) (e error) {

	logger.Debug("Endorsement request")

	signature := dac.NymSignatureFromBytes(args.Signature)
	pkNym, _ := dac.PointFromBytes(args.PkNym)
	indexValue, _ := dac.PointFromBytes(args.IndexValue)
	indices := dac.Indices{
		dac.Index{
			I:         1,
			J:         1,
			Attribute: indexValue,
		},
	}

	// Verify signature
	if e := signature.VerifyNym(sysParams.H, pkNym, args.getMessage()); e != nil {
		logger.Fatal("signature.VerifyNym():", e)
	}

	// Verify author
	// Ideally should verify that tp.indices[0].Attribute is equal to the expected value that permits using the blockchain
	peer.validateIdentity(args.Author, pkNym, indices)

	// Verify read / write permissions (should be cached)
	peer.validateIdentity(args.Author, pkNym, indices)

	// Execute proposal
	executeChaincode()

	// All set!
	schnorr := dac.MakeSchnorr(helpers.NewRand(), false)
	schnorrSignature := schnorr.Sign(peer.keys.sk, args.getMessage())

	logger.Debugf("peer-%d endorsed transaction payload %s", peer.id, fmt.Sprintf("user-%d", args.AuthorID))
	reply.Signature = schnorrSignature.ToBytes()
	reply.PK = dac.PointToBytes(peer.keys.pk)

	return
}

func (peer *RPCPeer) validateIdentity(proof []byte, pkNym interface{}, indices dac.Indices) {

	var key [32]byte
	copy(key[:], helpers.Sha3(proof)[:4])

	for _, cached := range peer.cache {
		if cached == key {
			return
		}
	}
	proofObj := dac.ProofFromBytes(proof)
	if e := proofObj.VerifyProof(sysParams.RootPk, sysParams.Ys, sysParams.H, pkNym, indices, []byte{}); e != nil {
		logger.Fatal("proofObj.VerifyProof():", e)
	}

	peer.cache = append(peer.cache, key)
}

func executeChaincode() {
	time.Sleep(50 * time.Millisecond)
}

// Endorsement ...
type Endorsement struct {
	Signature []byte // dac.SchnorrSignature
	PK        []byte
}

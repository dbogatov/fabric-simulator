package distributed

import (
	"fmt"
	"net/rpc"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/dbogatov/fabric-simulator/helpers"
)

// RPCPeer ...
type RPCPeer struct {
	keys KeysHolder

	id int

	cache [][32]byte

	revocationPK dac.PK
}

// MakeRPCPeer ...
func MakeRPCPeer(prg *amcl.RAND, id int, auditSk dac.SK) (rpcPeer *RPCPeer) {
	sk, pk := dac.GenerateKeys(helpers.NewRand(), 0)

	rpcPeer = &RPCPeer{
		id: id,
		keys: KeysHolder{
			sk: sk,
			pk: pk,
		},
		cache: make([][32]byte, 0),
	}

	revocationPk := makeRPCCallSync(sysParams.RevocationRPCAddress, "RPCRevocation.GetPK", new(int), new([]byte)).(*[]byte)
	revocationAuthorityPk, _ := dac.PointFromBytes(*revocationPk)

	rpcPeer.revocationPK = revocationAuthorityPk

	return
}

// Validate ...
func (peer *RPCPeer) Validate(args *Transaction, reply *bool) (e error) {

	pkNym, _ := dac.PointFromBytes(args.Proposal.PkNym)
	indexValue, _ := dac.PointFromBytes(args.Proposal.IndexValue)
	indices := dac.Indices{
		dac.Index{
			I:         1,
			J:         1,
			Attribute: indexValue,
		},
	}

	if e := dac.NymSignatureFromBytes(args.Signature).VerifyNym(sysParams.H, pkNym, args.Proposal.getMessage()); e != nil {
		panic(e)
	}

	if len(args.Endorsements) < sysParams.Endorsements {
		logger.Fatal("RPCPeer.Validate(): too few endorsements")
	}

	schnorr := dac.MakeSchnorr(helpers.NewRand(), false)
	for _, endorsement := range args.Endorsements {
		endorserPK, _ := dac.PointFromBytes(endorsement.PK)
		endorserSignature := dac.SchnorrSignatureFromBytes(endorsement.Signature)
		if e := schnorr.Verify(endorserPK, *endorserSignature, args.Proposal.getMessage()); e != nil {
			logger.Fatal("RPCPeer.Validate(): endorsement is invalid")
		}
	}

	peer.validateIdentity(args.Proposal.Author, pkNym, indices)

	if sysParams.Audit {
		auditProof := dac.AuditingProofFromBytes(args.AuditProof)
		auditEnc := dac.AuditingEncryptionFromBytes(args.AuditEnc)
		if e := auditProof.Verify(*auditEnc, pkNym, sysParams.AuditPK, sysParams.H); e != nil {
			logger.Fatal("RPCPeer.Validate(): audit proof is invalid")
		}
	}

	if sysParams.Revoke {
		nrhProof := dac.RevocationProofFromBytes(args.NonRevocationProof)
		if e := nrhProof.Verify(pkNym, FP256BN.NewBIGint(args.Epoch), sysParams.H, peer.revocationPK, sysParams.Ys[1]); e != nil {
			logger.Fatal("RPCPeer.Validate(): NRH is invalid")
		}
	}

	// somewhere here are read/write conflict check and ledger update
	// but they are negligible in comparison to crypto

	executeChaincode()

	*reply = true

	logger.Debug("Transaction validated!")

	return
}

// Order ...
func (peer *RPCPeer) Order(args *Transaction, reply *bool) (e error) {

	pkNym, _ := dac.PointFromBytes(args.Proposal.PkNym)
	indexValue, _ := dac.PointFromBytes(args.Proposal.IndexValue)
	indices := dac.Indices{
		dac.Index{
			I:         1,
			J:         1,
			Attribute: indexValue,
		},
	}

	peer.validateIdentity(args.Proposal.Author, pkNym, indices)

	logger.Debug("Validate TX identity, sending to others")

	// SEND TO OTHERS (including self)
	validateCalls := make([]*rpc.Call, 0)
	for _, other := range sysParams.PeerRPCAddresses {

		call := makeRPCCall(other, "RPCPeer.Validate", args, new(bool))
		validateCalls = append(validateCalls, call)
	}

	for _, validateCall := range validateCalls {

		<-validateCall.Done
		if validateCall.Error != nil {
			logger.Fatal(validateCall.Error)
		}
		if !*validateCall.Reply.(*bool) {
			logger.Fatal("Validation failed")
		}
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
	reply.ID = peer.id

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
	ID        int
}

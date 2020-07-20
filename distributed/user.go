package distributed

import (
	"fmt"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/dbogatov/fabric-simulator/helpers"
	"gonum.org/v1/gonum/stat/distuv"
)

// User ...
type User struct {
	creds   CredentialsHolder
	epoch   int
	poisson distuv.Poisson
	nrh     dac.GrothSignature

	revocationAuthorityPk dac.PK
	revocationPk          dac.PK
}

const userLevel = 2

// MakeUser ...
func MakeUser(prg *amcl.RAND, id int) (user *User) {

	userSk, userPk := dac.GenerateKeys(prg, userLevel)

	nonce := makeRPCCallSync(sysParams.OrgRPCAddress, "RPCOrganization.GetNonce", new(int), new([]byte)).(*[]byte)

	credRequest := &CredRequest{
		Request: dac.MakeCredRequest(prg, userSk, *nonce, userLevel).ToBytes(),
		ID:      id,
	}
	creds := makeRPCCallSync(sysParams.OrgRPCAddress, "RPCOrganization.ProcessCredRequest", credRequest, new(Credentials)).(*Credentials)

	credentials := dac.CredentialsFromBytes(creds.Creds)

	if e := credentials.Verify(userSk, sysParams.RootPk, sysParams.Ys); e != nil {
		logger.Fatal("credentials.Verify():", e)
	}

	revocationPk := makeRPCCallSync(sysParams.RevocationRPCAddress, "RPCRevocation.GetPK", new(int), new([]byte)).(*[]byte)
	revocationAuthorityPk, _ := dac.PointFromBytes(*revocationPk)

	user = &User{
		creds: CredentialsHolder{
			KeysHolder: KeysHolder{
				pk: userPk,
				sk: userSk,
			},
			credentials: *credentials,
			kind:        fmt.Sprintf("user-%d", id),
			id:          id,
		},
		epoch: -1,
		poisson: distuv.Poisson{
			Lambda: 3600.0 / float64(sysParams.Frequency),
		},

		revocationAuthorityPk: revocationAuthorityPk,
		revocationPk:          FP256BN.ECP_generator().Mul(userSk),
	}

	logger.Notice("Received credentials")

	user.runTransactions()

	return
}

func (user *User) runTransactions() {

	for i := 0; i < sysParams.Transactions; i++ {

		// subsequent sleeps Poisson
		if sysParams.Frequency > 0 {
			sleep := time.Duration((3600.0/user.poisson.Rand())*1000) * time.Millisecond
			logger.Debugf("Will wait %d ms", sleep.Milliseconds())
			time.Sleep(sleep)
		}

		message := helpers.RandomString(helpers.NewRand(), 16)
		user.submitTransaction(message)
	}
}

func (user *User) submitTransaction(message string) {

	startTime := time.Now()

	logger.Noticef("Transaction \"%s\" started", message)

	prg := helpers.NewRand()

	hash := helpers.Sha3([]byte(message))
	endorsers := make([]int, 0)

	firstEndorser := helpers.PeerByHash(helpers.Sha3([]byte(message)), sysParams.Peers)

	for peer := 0; peer < sysParams.Endorsements; peer++ {
		endorsers = append(endorsers, (firstEndorser+peer)%sysParams.Peers)
	}

	proposal, pkNym, skNym := user.MakeTransactionProposal(hash)
	endorsements := make([]Endorsement, 0)

	schnorr := dac.MakeSchnorr(prg, false)
	endorseCallClients := make([]rpcCallClient, 0)
	for _, endorser := range endorsers {
		callClient := makeRPCCall(sysParams.PeerRPCAddresses[endorser], "RPCPeer.Endorse", proposal, new(Endorsement))
		endorseCallClients = append(endorseCallClients, callClient)
	}

	for _, endorseCallClient := range endorseCallClients {

		<-endorseCallClient.call.Done
		if endorseCallClient.call.Error != nil {
			logger.Fatal(endorseCallClient.call.Error)
		}
		endorsement := endorseCallClient.call.Reply.(*Endorsement)
		endorseCallClient.client.Close()

		endorsements = append(endorsements, *endorsement)

		logger.Infof("Got endorsement from %d", endorsement.ID)

		endorserPK, _ := dac.PointFromBytes(endorsement.PK)
		endorserSignature := dac.SchnorrSignatureFromBytes(endorsement.Signature)
		if e := schnorr.Verify(endorserPK, *endorserSignature, proposal.getMessage()); e != nil {
			logger.Fatal("schnorr.Verify():", e)
		}
	}

	txSignature := dac.SignNym(prg, pkNym, skNym, user.creds.sk, sysParams.H, proposal.getMessage())

	tx := &Transaction{
		Signature:    txSignature.ToBytes(),
		Proposal:     *proposal,
		Endorsements: endorsements,
		Epoch:        user.epoch,
		AuthorPK:     dac.PointToBytes(user.creds.pk),
	}

	if sysParams.Revoke {
		user.updateNRH()

		nrhProof := dac.RevocationProve(prg, user.nrh, user.creds.sk, skNym, FP256BN.NewBIGint(user.epoch), sysParams.H, sysParams.Ys[0])
		tx.NonRevocationProof = nrhProof.ToBytes()
	}

	if sysParams.Audit {

		// fresh auditing encryption and proof every transaction
		auditEnc, auditR := dac.AuditingEncrypt(helpers.NewRand(), sysParams.AuditPK, user.creds.pk)

		tx.AuditEnc = auditEnc.ToBytes()
		auditProof := dac.AuditingProve(prg, auditEnc, user.creds.pk, user.creds.sk, pkNym, skNym, sysParams.AuditPK, auditR, sysParams.H)
		tx.AuditProof = auditProof.ToBytes()
	}

	orderer := helpers.PeerByHash(helpers.Sha3([]byte(fmt.Sprintf("%s-order", message))), sysParams.Peers)

	makeRPCCallSync(sysParams.PeerRPCAddresses[orderer], "RPCPeer.Order", tx, new(bool))

	endTime := time.Now()

	logger.Noticef("Transaction \"%s\" completed in %d ms", message, endTime.Sub(startTime).Milliseconds())
}

func (user *User) updateNRH() {

	epoch := makeRPCCallSync(sysParams.RevocationRPCAddress, "RPCRevocation.GetEpoch", new(int), new(int)).(*int)

	if user.epoch != *epoch {
		logger.Debugf("Detected epoch change; requesting new handle...")
		user.epoch = *epoch

		nrr := &NonRevocationRequest{
			PK: dac.PointToBytes(user.revocationPk),
		}
		nrh := makeRPCCallSync(sysParams.RevocationRPCAddress, "RPCRevocation.ProcessNRR", nrr, new(NonRevocationHandle)).(*NonRevocationHandle)

		handle := dac.GrothSignatureFromBytes(nrh.Handle)
		groth := dac.MakeGroth(helpers.NewRand(), true, sysParams.Ys[1])

		if e := groth.Verify(user.revocationAuthorityPk, *handle, []interface{}{user.revocationPk, FP256BN.ECP_generator().Mul(FP256BN.NewBIGint(user.epoch))}); e != nil {
			logger.Fatal("groth.Verify():", e)
		}
		user.nrh = *handle
		logger.Debug("Non-revocation handle updated")
	} else {
		logger.Debug("Non-revocation handle is up-to-date")
	}
}

// MakeTransactionProposal ...
func (user *User) MakeTransactionProposal(hash []byte) (tp *TransactionProposal, pkNym interface{}, skNym dac.SK) {

	prg := helpers.NewRand()

	skNym, pkNym = dac.GenerateNymKeys(prg, user.creds.sk, sysParams.H)
	indices := dac.Indices{
		dac.Index{
			I:         1,
			J:         1,
			Attribute: user.creds.credentials.Attributes[1][1],
		},
	}

	proof, e := user.creds.credentials.Prove(
		prg,
		user.creds.sk,
		sysParams.RootPk,
		indices,
		[]byte{},
		sysParams.Ys,
		sysParams.H,
		skNym,
	)

	if e != nil {
		logger.Fatal("credentials.Prove():", e)
	}
	author := proof.ToBytes()

	tp = &TransactionProposal{
		Chaincode:  "chaincode: hash | policy: write",
		AuthorID:   user.creds.id,
		Hash:       hash,
		Author:     author,
		PkNym:      dac.PointToBytes(pkNym),
		IndexValue: dac.PointToBytes(indices[0].Attribute),
	}

	signature := dac.SignNym(prg, pkNym, skNym, user.creds.sk, sysParams.H, tp.getMessage())
	tp.Signature = signature.ToBytes()

	return
}

func (tp *TransactionProposal) getMessage() (message []byte) {

	message = make([]byte, 0)

	message = append(message, tp.Hash...)
	message = append(message, []byte(tp.Chaincode)...)
	message = append(message, byte(tp.AuthorID))
	message = append(message, tp.Author...)

	return
}

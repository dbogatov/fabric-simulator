package distributed

import (
	"fmt"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-simulator/helpers"
)

// RPCRoot ...
type RPCRoot struct {
	creds   CredentialsHolder
	starter []byte
}

// MakeRPCRoot ...
func MakeRPCRoot(prg *amcl.RAND, rootSk dac.SK) (rpcRoot *RPCRoot) {
	rpcRoot = &RPCRoot{
		creds: CredentialsHolder{
			KeysHolder: KeysHolder{
				pk: sysParams.RootPk,
				sk: rootSk,
			},
			credentials: *dac.MakeCredentials(sysParams.RootPk),
			kind:        "root",
			id:          0,
		},
		starter: nil,
	}

	rpcRoot.starter = rpcRoot.creds.credentials.ToBytes()

	return
}

// GetNonce ...
func (rpcRoot *RPCRoot) GetNonce(args *int, reply *[]byte) (e error) {
	prg := helpers.NewRand()

	*reply = helpers.RandomBytes(prg, helpers.NonceSize)

	logger.Debug("Nonce requested")

	return
}

// ProcessCredRequest ...
func (rpcRoot *RPCRoot) ProcessCredRequest(args *CredRequest, reply *Credentials) (e error) {

	credRequest := dac.CredRequestFromBytes(args.Request)
	prg := helpers.NewRand()

	if e := credRequest.Validate(); e != nil {
		logger.Fatal("credRequest.Validate():", e)
	}

	attributes := []interface{}{
		dac.ProduceAttributes(orgLevel, fmt.Sprintf("org-%d", args.ID))[0],
		dac.ProduceAttributes(orgLevel, "has-right-to-post")[0],
	}

	credsOrg := dac.CredentialsFromBytes(rpcRoot.starter)
	if e := credsOrg.Delegate(rpcRoot.creds.sk, credRequest.Pk, attributes, prg, sysParams.Ys); e != nil {
		logger.Fatal("credsOrg.Delegate():", e)
	}

	*&reply.Creds = credsOrg.ToBytes()

	logger.Debug("Credentials granted")

	return
}

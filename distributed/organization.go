package distributed

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-simulator/helpers"
)

// RPCOrganization ...
type RPCOrganization struct {
	CredentialsHolder
}

const orgLevel = 1

// MakeRPCOrganization ...
func MakeRPCOrganization(prg *amcl.RAND, id int) (rpcOrg *RPCOrganization) {

	orgSk, orgPk := dac.GenerateKeys(prg, orgLevel)

	client, err := rpc.DialHTTP("tcp", sysParams.RootRPCAddress)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	void := 0
	nonce := new([]byte)
	getNonceCall := client.Go("RPCRoot.GetNonce", &void, nonce, nil)
	<-getNonceCall.Done

	credRequest := &CredRequest{
		Request: dac.MakeCredRequest(prg, orgSk, *nonce, orgLevel).ToBytes(),
		ID:      id,
	}
	creds := new(Credentials)
	processCredRequestCall := client.Go("RPCRoot.ProcessCredRequest", credRequest, creds, nil)
	<-processCredRequestCall.Done

	credentials := dac.CredentialsFromBytes(creds.Creds)

	if e := credentials.Verify(orgSk, sysParams.RootPk, sysParams.Ys); e != nil {
		logger.Fatal("credentials.Verify():", e)
	}

	rpcOrg = &RPCOrganization{
		CredentialsHolder{
			KeysHolder: KeysHolder{
				pk: orgPk,
				sk: orgSk,
			},
			credentials: *credentials,
			kind:        fmt.Sprintf("org-%d", id),
			id:          id,
		},
	}

	logger.Info("Received credentials")

	return
}

// GetNonce ...
func (rpcOrg *RPCOrganization) GetNonce(args *int, reply *[]byte) (e error) {
	prg := helpers.NewRand()

	*reply = helpers.RandomBytes(prg, helpers.NonceSize)

	logger.Debug("Nonce requested")

	return
}

// ProcessCredRequest ...
func (rpcOrg *RPCOrganization) ProcessCredRequest(args *CredRequest, reply *Credentials) (e error) {

	credRequest := dac.CredRequestFromBytes(args.Request)
	prg := helpers.NewRand()

	if e := credRequest.Validate(); e != nil {
		logger.Fatal("credRequest.Validate():", e)
	}

	attributes := []interface{}{
		dac.ProduceAttributes(userLevel, fmt.Sprintf("user-%d", args.ID))[0],
		dac.ProduceAttributes(userLevel, "has-right-to-post")[0],
	}

	credsUser := dac.CredentialsFromBytes(rpcOrg.credentials.ToBytes())
	if e := credsUser.Delegate(rpcOrg.sk, credRequest.Pk, attributes, prg, sysParams.Ys); e != nil {
		logger.Fatal("credsOrg.Delegate():", e)
	}

	*&reply.Creds = credsUser.ToBytes()

	logger.Debug("Credentials granted")

	return
}

package distributed

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
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

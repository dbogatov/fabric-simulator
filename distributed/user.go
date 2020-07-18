package distributed

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

// User ...
type User struct {
	creds CredentialsHolder
}

const userLevel = 2

// MakeUser ...
func MakeUser(prg *amcl.RAND, id int) (user *User) {

	userSk, userPk := dac.GenerateKeys(prg, userLevel)

	client, err := rpc.DialHTTP("tcp", sysParams.OrgRPCAddress)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	void := 0
	nonce := new([]byte)
	getNonceCall := client.Go("RPCOrganization.GetNonce", &void, nonce, nil)
	<-getNonceCall.Done

	credRequest := &CredRequest{
		Request: dac.MakeCredRequest(prg, userSk, *nonce, userLevel).ToBytes(),
		ID:      id,
	}
	creds := new(Credentials)
	processCredRequestCall := client.Go("RPCOrganization.ProcessCredRequest", credRequest, creds, nil)
	<-processCredRequestCall.Done

	credentials := dac.CredentialsFromBytes(creds.Creds)

	if e := credentials.Verify(userSk, sysParams.RootPk, sysParams.Ys); e != nil {
		logger.Fatal("credentials.Verify():", e)
	}

	user = &User{
		CredentialsHolder{
			KeysHolder: KeysHolder{
				pk: userPk,
				sk: userSk,
			},
			credentials: *credentials,
			kind:        fmt.Sprintf("user-%d", id),
			id:          id,
		},
	}

	logger.Info("Received credentials")

	return
}

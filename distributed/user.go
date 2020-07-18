package distributed

import (
	"fmt"
	"log"
	"net/rpc"
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

	revocationRPC         *rpc.Client
	revocationAuthorityPk dac.PK
	revocationPk          dac.PK
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

	clientRevocation, err := rpc.DialHTTP("tcp", sysParams.RevocationRPCAddress)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	revocationPk := new([]byte)
	getPkCall := clientRevocation.Go("RPCRevocation.GetPK", &void, revocationPk, nil)
	<-getPkCall.Done
	revocationPkBytes, _ := dac.PointFromBytes(*revocationPk)

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
		revocationRPC:         clientRevocation,
		revocationAuthorityPk: revocationPkBytes,
		revocationPk:          FP256BN.ECP_generator().Mul(userSk),
	}

	logger.Info("Received credentials")

	user.runTransactions()

	return
}

func (user *User) runTransactions() {

	for i := 0; i < sysParams.Transactions; i++ {

		// subsequent sleeps Poisson
		if sysParams.Frequency > 0 {
			sleep := time.Duration((3600.0/user.poisson.Rand())*1000) * time.Millisecond
			logger.Debugf("user-%d will wait %d ms", user.creds.id, sleep.Milliseconds())
			time.Sleep(sleep)
		}

		message := helpers.RandomString(helpers.NewRand(), 16)
		user.submitTransaction(message)
	}
}

func (user *User) submitTransaction(message string) {

	if sysParams.Revoke {

		void := 0
		epoch := new(int)
		getEpochCall := user.revocationRPC.Go("RPCRevocation.GetEpoch", &void, epoch, nil)
		<-getEpochCall.Done

		if user.epoch != *epoch {
			logger.Debugf("user-%d (%s) detected epoch change; requesting new handle...", user.creds.id, message)
			user.epoch = *epoch

			nrr := &NonRevocationRequest{
				PK: dac.PointToBytes(user.revocationPk),
			}
			nrh := new(NonRevocationHandle)
			getNRHCall := user.revocationRPC.Go("RPCRevocation.ProcessNRR", nrr, nrh, nil)
			<-getNRHCall.Done

			handle := dac.GrothSignatureFromBytes(nrh.Handle)
			groth := dac.MakeGroth(helpers.NewRand(), true, sysParams.Ys[1])

			if e := groth.Verify(user.revocationAuthorityPk, *handle, []interface{}{user.revocationPk, FP256BN.ECP_generator().Mul(FP256BN.NewBIGint(user.epoch))}); e != nil {
				logger.Fatal("groth.Verify():", e)
			}
			logger.Debug("Non-revocation handle updated")
		} else {
			logger.Debug("Non-revocation handle is up-to-date")
		}
	}

}

package distributed

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-simulator/helpers"
	"github.com/op/go-logging"
)

var logger *logging.Logger

// SetLogger ...
func SetLogger(l *logging.Logger) {
	logger = l
}

var sysParams helpers.SystemParameters

// Simulate ...
func Simulate(rootSk dac.SK, params *helpers.SystemParameters, root bool, organization int) (e error) {

	sysParams = *params

	prg := helpers.NewRand()

	if root {
		logger.Info("Running as ROOT")

		rpcRoot := MakeRPCRoot(prg, rootSk)

		runRPCServer(rpcRoot)
	} else if organization > 0 {
		logger.Infof("Running as ORGANIZATION %d", organization)

		rpcOrg := MakeRPCOrganization(prg, organization)

		runRPCServer(rpcOrg)
	}

	return
}

func runCredentialGeneration() {
	client, err := rpc.DialHTTP("tcp", sysParams.RootRPCAddress)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	void := 0
	nonce := new([]byte)

	getNonceCall := client.Go("RPCRoot.GetNonce", &void, nonce, nil)
	<-getNonceCall.Done

	logger.Info(nonce)
}

func runRPCServer(rpcEntity interface{}) {

	logger.Infof("Listening to %d", sysParams.RPCPort)

	rpc.Register(rpcEntity)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", sysParams.RPCPort))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}

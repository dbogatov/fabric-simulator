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
func Simulate(rootSk, auditSk dac.SK, params *helpers.SystemParameters, root bool, organization, peer, user int, revocation bool) (e error) {

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
	} else if peer > 0 {
		logger.Infof("Running as PEER %d", peer)

		rpcPeer := MakeRPCPeer(prg, peer, auditSk)

		runRPCServer(rpcPeer)
	} else if user > 0 {
		logger.Infof("Running as USER %d", organization)

		MakeUser(prg, user)
	} else if revocation {
		logger.Info("Running as REVOCATION")

		rpcRevocation := MakeRPCRevocation(prg)

		runRPCServer(rpcRevocation)
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

func makeRPCCall(address, method string, arg, reply interface{}) *rpc.Call {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	return client.Go(method, arg, reply, nil)
}

func makeRPCCallSync(address, method string, arg, reply interface{}) interface{} {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	call := client.Go(method, arg, reply, nil)
	<-call.Done
	return call.Reply
}

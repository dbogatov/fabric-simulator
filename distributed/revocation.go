package distributed

import (
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/dbogatov/fabric-simulator/helpers"
)

// RPCRevocation ...
type RPCRevocation struct {
	keys KeysHolder
}

var epoch int = 1

// MakeRPCRevocation ...
func MakeRPCRevocation(prg *amcl.RAND) (rpcRevocation *RPCRevocation) {

	groth := dac.MakeGroth(helpers.NewRand(), true, sysParams.Ys[1])
	sk, pk := groth.Generate()

	rpcRevocation = &RPCRevocation{
		keys: KeysHolder{
			pk: pk,
			sk: sk,
		},
	}

	go func() {
		for {
			select {
			case <-time.After(time.Duration(sysParams.Epoch) * time.Second):
				epoch++
				logger.Debugf("Epoch advanced to %d", epoch)
				continue
			}
		}
	}()

	return
}

// GetEpoch ...
func (rpcRevocation *RPCRevocation) GetEpoch(args *int, reply *int) (e error) {

	*reply = epoch

	logger.Debug("Epoch read")

	return
}

// GetPK ...
func (rpcRevocation *RPCRevocation) GetPK(args *int, reply *[]byte) (e error) {

	*reply = dac.PointToBytes(rpcRevocation.keys.pk)

	logger.Debug("PK requested")

	return
}

// ProcessNRR ...
func (rpcRevocation *RPCRevocation) ProcessNRR(args *NonRevocationRequest, reply *NonRevocationHandle) (e error) {

	prg := helpers.NewRand()
	nrr, _ := dac.PointFromBytes(args.PK)

	nrh := dac.SignNonRevoke(prg, rpcRevocation.keys.sk, nrr, FP256BN.NewBIGint(epoch), sysParams.Ys[1])

	*&reply.Handle = nrh.ToBytes()

	logger.Debug("Non-revocation handle granted")

	return
}

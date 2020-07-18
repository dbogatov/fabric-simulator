package helpers

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/op/go-logging"
)

// YsNum ...
const YsNum = 10

// NonceSize ...
const NonceSize = 32

// SystemParameters ...
type SystemParameters struct {
	Ys                     [][]interface{}
	H                      *FP256BN.ECP2 // because we have users on level 2
	RootPk                 dac.PK
	Orgs                   int
	Users                  int
	Peers                  int
	Endorsements           int
	Epoch                  int
	Transactions           int
	Frequency              int
	ConcurrentEndorsements int
	ConcurrentValidations  int
	ConcurrentRevocations  int
	BandwidthGlobal        int // B/s
	BandwidthLocal         int // B/s
	Revoke                 bool
	Audit                  bool
	RPCPort                int
	RootRPCAddress         string
	OrgRPCAddress          string
	RevocationRPCAddress   string
}

// MakeSystemParameters ...
func MakeSystemParameters(logger *logging.Logger, prg *amcl.RAND, orgs, users, peers, endorsements, epoch, bandwidthGlobal, bandwidthLocal, concurrentEndorsements, concurrentValidations, concurrentRevocations, transactions, frequency int, revoke, audit bool, rpcPort int, rootRPCAddress, orgRPCAddress, revocationRPCAddress string) (sysParams *SystemParameters, rootSk dac.SK) {

	sysParams = &SystemParameters{
		Orgs:                   orgs,
		Users:                  users,
		Peers:                  peers,
		Endorsements:           endorsements,
		Epoch:                  epoch,
		BandwidthGlobal:        bandwidthGlobal,
		BandwidthLocal:         bandwidthLocal,
		Frequency:              frequency,
		ConcurrentEndorsements: concurrentEndorsements,
		ConcurrentValidations:  concurrentValidations,
		ConcurrentRevocations:  concurrentRevocations,
		Transactions:           transactions,
		Revoke:                 revoke,
		Audit:                  audit,
		H:                      FP256BN.ECP2_generator().Mul(FP256BN.Randomnum(FP256BN.NewBIGints(FP256BN.CURVE_Order), prg)),
		RPCPort:                rpcPort,
		RootRPCAddress:         rootRPCAddress,
		OrgRPCAddress:          orgRPCAddress,
		RevocationRPCAddress:   revocationRPCAddress,
	}

	logger.Noticef("%+v\n", sysParams)

	sysParams.Ys = make([][]interface{}, 2)
	sysParams.Ys[0] = dac.GenerateYs(false, YsNum, prg)
	sysParams.Ys[1] = dac.GenerateYs(true, YsNum, prg)

	rootSk, sysParams.RootPk = dac.GenerateKeys(prg, 0)

	return
}

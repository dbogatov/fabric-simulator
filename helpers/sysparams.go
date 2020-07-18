package helpers

import (
	"github.com/dbogatov/dac-lib/dac"
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
}

// MakeSystemParameters ...
func MakeSystemParameters(logger *logging.Logger, orgs, users, peers, endorsements, epoch, bandwidthGlobal, bandwidthLocal, concurrentEndorsements, concurrentValidations, concurrentRevocations, transactions, frequency int, revoke, audit bool) (sysParams *SystemParameters, rootSk dac.SK) {

	prg := NewRand()

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
	}

	logger.Noticef("%+v\n", sysParams)

	sysParams.Ys = make([][]interface{}, 2)
	sysParams.Ys[0] = dac.GenerateYs(false, YsNum, prg)
	sysParams.Ys[1] = dac.GenerateYs(true, YsNum, prg)

	rootSk, sysParams.RootPk = dac.GenerateKeys(prg, 0)

	return
}

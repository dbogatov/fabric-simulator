package main

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

// YsNum ...
const YsNum = 10

// NonceSize ...
const NonceSize = 32

// SystemParameters ...
type SystemParameters struct {
	ys                     [][]interface{}
	h                      *FP256BN.ECP2 // because we have users on level 2
	rootPk                 dac.PK
	orgs                   int
	users                  int
	peers                  int
	endorsements           int
	epoch                  int
	transactions           int
	concurrentEndorsements int
	concurrentValidations  int
	concurrentRevocations  int
	bandwidth              int // B/s
	global                 bool
	revoke                 bool
	audit                  bool
	network                *Network
}

// MakeSystemParameters ...
func MakeSystemParameters(orgs, users, peers, endorsements, epoch, bandwidth, concurrentEndorsements, concurrentValidations, concurrentRevocations, transactions int, revoke, audit, global bool) (sysParams *SystemParameters, rootSk dac.SK) {

	logger.Critical(bandwidth)

	prg := newRand()

	sysParams = &SystemParameters{
		orgs:                   orgs,
		users:                  users,
		peers:                  peers,
		endorsements:           endorsements,
		epoch:                  epoch,
		bandwidth:              bandwidth,
		global:                 global,
		concurrentEndorsements: concurrentEndorsements,
		concurrentValidations:  concurrentValidations,
		concurrentRevocations:  concurrentRevocations,
		transactions:           transactions,
		revoke:                 revoke,
		audit:                  audit,
		h:                      FP256BN.ECP2_generator().Mul(FP256BN.Randomnum(FP256BN.NewBIGints(FP256BN.CURVE_Order), prg)),
	}

	logger.Infof("%+v\n", sysParams)

	sysParams.ys = make([][]interface{}, 2)
	sysParams.ys[0] = dac.GenerateYs(false, YsNum, prg)
	sysParams.ys[1] = dac.GenerateYs(true, YsNum, prg)

	rootSk, sysParams.rootPk = dac.GenerateKeys(prg, 0)

	return
}

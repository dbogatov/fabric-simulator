package main

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

// YsNum ...
const YsNum = 10

// NonceSize ...
const NonceSize = 32

// SystemParameters ...
type SystemParameters struct {
	ys                     [][]interface{}
	h                      *FP256BN.ECP
	rootPk                 dac.PK
	orgs                   int
	users                  int
	peers                  int
	transactions           int
	concurrentEndorsements int
	concurrentValidations  int
	bandwidth              int // B/s
	revoke                 bool
	audit                  bool
}

// MakeSystemParameters ...
func MakeSystemParameters(prg *amcl.RAND, orgs, users, peers, bandwidth, concurrentEndorsements, concurrentValidations, transactions int, revoke, audit bool) (sysParams *SystemParameters, rootSk dac.SK) {

	sysParams = &SystemParameters{
		orgs:                   orgs,
		users:                  users,
		peers:                  peers,
		bandwidth:              bandwidth,
		concurrentEndorsements: concurrentEndorsements,
		concurrentValidations:  concurrentValidations,
		transactions:           transactions,
		revoke:                 revoke,
		audit:                  audit,
		h:                      FP256BN.ECP_generator().Mul(FP256BN.Randomnum(FP256BN.NewBIGints(FP256BN.CURVE_Order), prg)),
	}

	sysParams.ys = make([][]interface{}, 2)
	sysParams.ys[0] = dac.GenerateYs(false, YsNum, prg)
	sysParams.ys[1] = dac.GenerateYs(true, YsNum, prg)

	rootSk, sysParams.rootPk = dac.GenerateKeys(prg, 0)

	return
}

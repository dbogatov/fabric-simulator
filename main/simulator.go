package main

import (
	"strconv"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

var sysParams SystemParameters

func simulate(orgs, users, peers, epoch, seed, bandwidth int, revoke, audit bool, idemix string) (e error) {

	logger.Infof("Seed is %d, bandwidth is %d B/s\n", seed, bandwidth)
	logger.Infof("%d organizations %d users each managed by %d peers\n", orgs, users, peers)
	logger.Infof("Epochs are %d seconds long\n", epoch)
	logger.Infof("Revocations enabled: %t, auditings enabled %t\n", revoke, audit)
	logger.Infof("\"%s\" version of idemix is used\n", idemix)

	prg := amcl.NewRAND()
	prg.Clean()
	prg.Seed(1, []byte(strconv.Itoa(seed)))

	sys, rootSk := MakeSystemParameters(prg, orgs, users, bandwidth)
	sysParams = *sys

	MakeNetwork(prg, rootSk)

	return
}

// KeysHolder ...
type KeysHolder struct {
	pk dac.PK
	sk dac.SK
}

// CredentialsHolder ...
type CredentialsHolder struct {
	KeysHolder
	credentials dac.Credentials
	name        string
}

// Organization ...
type Organization struct {
	CredentialsHolder
	id int
}

// User ...
type User struct {
	CredentialsHolder
	id  int
	org int
}

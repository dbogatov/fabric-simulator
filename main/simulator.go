package main

import (
	"strconv"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

var sysParams SystemParameters

func simulate(orgs, users, peers, epoch, seed int, revoke, audit bool, idemix string) (e error) {

	log.Infof("Seed is %d\n", seed)
	log.Infof("%d organizations %d users each managed by %d peers\n", orgs, users, peers)
	log.Infof("Epochs are %d seconds long\n", epoch)
	log.Infof("Revocations enabled: %t, auditings enabled %t\n", revoke, audit)
	log.Infof("\"%s\" version of idemix is used\n", idemix)

	prg := amcl.NewRAND()
	prg.Clean()
	prg.Seed(1, []byte(strconv.Itoa(seed)))

	sys, rootSk := MakeSystemParameters(prg, orgs, users)
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
}

// User ...
type User struct {
	CredentialsHolder
}

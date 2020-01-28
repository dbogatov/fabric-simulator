package main

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

var sysParams SystemParameters

func simulate(prg *amcl.RAND, rootSk dac.SK) (e error) {

	sysParams.network = MakeNetwork(prg, rootSk)

	sysParams.network.users[0].submitTransaction("message")

	sysParams.network.stop()

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

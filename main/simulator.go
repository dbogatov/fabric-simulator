package main

import (
	"sync"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

var sysParams SystemParameters

func simulate(rootSk dac.SK) (e error) {

	sysParams.network = MakeNetwork(amcl.NewRAND(), rootSk)

	var wgUser sync.WaitGroup
	wgUser.Add(sysParams.orgs * sysParams.users)

	for user := 0; user < sysParams.orgs*sysParams.users; user++ {
		go func(user int) {
			defer wgUser.Done()

			for i := 0; i < sysParams.transactions; i++ {
				message := string(randomBytes(amcl.NewRAND(), 16))
				sysParams.network.users[user].submitTransaction(message)
			}

		}(user)
	}

	wgUser.Wait()

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

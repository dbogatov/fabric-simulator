package main

import (
	"fmt"
	"sync"

	"github.com/dbogatov/dac-lib/dac"
)

var sysParams SystemParameters

func simulate(rootSk dac.SK) (e error) {

	sysParams.network = MakeNetwork(newRand(), rootSk)

	var wgUser sync.WaitGroup
	wgUser.Add(sysParams.orgs * sysParams.users)

	for user := 0; user < sysParams.orgs*sysParams.users; user++ {

		if sysParams.revoke {
			sysParams.network.users[user].requestNonRevocation()
		}

		go func(user int) {
			defer wgUser.Done()

			for i := 0; i < sysParams.transactions; i++ {
				message := randomString(newRand(), 16)
				sysParams.network.users[user].submitTransaction(message)
			}

		}(user)
	}

	wgUser.Wait()

	// Auditing
	if sysParams.audit {

		logger.Infof("Audit started over %d transactions", len(sysParams.network.transactions))

		for _, transaction := range sysParams.network.transactions {
			authorPk := transaction.auditEnc.AuditingDecrypt(sysParams.network.auditor.sk)
			if !dac.PkEqual(authorPk, sysParams.network.users[transaction.proposal.authorID].CredentialsHolder.pk) {
				panic("auditing failed")
			}
		}

		logger.Info("Audit completed")

	}

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
	id          int
	kind        string
}

func (credHolder CredentialsHolder) name() string {
	return fmt.Sprintf("%s-%d", credHolder.kind, credHolder.id)
}

// Organization ...
type Organization struct {
	CredentialsHolder
}

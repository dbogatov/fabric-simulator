package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/dbogatov/dac-lib/dac"
)

var sysParams SystemParameters

func simulate(rootSk dac.SK) (e error) {

	start := time.Now()

	sysParams.network = MakeNetwork(newRand(), rootSk)

	var wgUser sync.WaitGroup
	wgUser.Add(sysParams.orgs * sysParams.users)

	for user := 0; user < sysParams.orgs*sysParams.users; user++ {

		if sysParams.revoke {
			sysParams.network.users[user].requestNonRevocation()
		}

		go func(user int) {
			defer wgUser.Done()

			// first sleep uniform
			time.Sleep(time.Duration(rand.Intn(sysParams.frequency*1000)) * time.Millisecond)

			for i := 0; i < sysParams.transactions; i++ {
				userObj := sysParams.network.users[user]

				// subsequent sleeps Poisson
				sleep := time.Duration((3600.0/userObj.poisson.Rand())*1000) * time.Millisecond
				logger.Debugf("user-%d will wait %d ms", user, sleep.Milliseconds())
				time.Sleep(sleep)

				message := randomString(newRand(), 16)
				userObj.submitTransaction(message)
			}

		}(user)
	}

	wgUser.Wait()

	// Auditing
	if sysParams.audit {

		logger.Noticef("Audit started over %d transactions", len(sysParams.network.transactions))

		for _, transaction := range sysParams.network.transactions {
			authorPk := transaction.auditEnc.AuditingDecrypt(sysParams.network.auditor.sk)
			if !dac.PkEqual(authorPk, sysParams.network.users[transaction.proposal.authorID].CredentialsHolder.pk) {
				panic("auditing failed")
			}
		}

		logger.Notice("Audit completed")

	}

	sysParams.network.stop()

	logger.Noticef("Simulation completed in %d seconds", int(math.Round(time.Since(start).Seconds())))

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

package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/dbogatov/dac-lib/dac"
)

var sysParams SystemParameters

// Simulate ...
func Simulate(rootSk dac.SK, params *SystemParameters) (e error) {

	sysParams = *params

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
			if sysParams.frequency > 0 {
				time.Sleep(time.Duration(rand.Intn(sysParams.frequency*1000)) * time.Millisecond)
			}

			for i := 0; i < sysParams.transactions; i++ {
				userObj := sysParams.network.users[user]

				// subsequent sleeps Poisson
				if sysParams.frequency > 0 {
					sleep := time.Duration((3600.0/userObj.poisson.Rand())*1000) * time.Millisecond
					logger.Debugf("user-%d will wait %d ms", user, sleep.Milliseconds())
					time.Sleep(sleep)
				}

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
			recordCryptoEvent(auditDecrypt)
			if !dac.PkEqual(authorPk, sysParams.network.users[transaction.proposal.authorID].CredentialsHolder.pk) {
				panic("auditing failed")
			}
		}

		logger.Notice("Audit completed")

	}

	sysParams.network.stop()

	logger.Noticef("Simulation completed in %d seconds", int(math.Round(time.Since(start).Seconds())))

	printStats()

	return
}

func printStats() {

	// crypto events
	logger.Critical("Crypto events:")
	for event, times := range sysParams.cryptoEvents {
		logger.Criticalf("\t%-20s : %3d : (%4.1f per transaction)\n", event, times, float64(times)/float64(len(sysParams.network.transactions)))
	}

	// transaction timings
	logger.Criticalf("For %d transactions", len(sysParams.transactionTimings))
	printTimingBasics := func(
		start func(TransactionTimingInfo) time.Time,
		end func(TransactionTimingInfo) time.Time,
		description string,
	) {
		var min, max, total, avg time.Duration
		min = time.Duration(3600 * time.Second)
		total = 0
		max = 0

		for _, info := range sysParams.transactionTimings {
			elapsed := end(info).Sub(start(info))
			if elapsed < min {
				min = elapsed
			}
			if elapsed > max {
				max = elapsed
			}
			total += elapsed
		}
		avg = time.Duration(total.Nanoseconds() / int64(len(sysParams.transactionTimings)))

		logger.Criticalf("%15s : min %4d ms, max %4d ms, avg %4d ms\n", description, min.Milliseconds(), max.Milliseconds(), avg.Milliseconds())
	}

	printTimingBasics(
		func(info TransactionTimingInfo) time.Time { return info.start },
		func(info TransactionTimingInfo) time.Time { return info.end },
		"total",
	)
	printTimingBasics(
		func(info TransactionTimingInfo) time.Time { return info.endorsementsStart },
		func(info TransactionTimingInfo) time.Time { return info.endorsementsEnd },
		"endorsements",
	)
	printTimingBasics(
		func(info TransactionTimingInfo) time.Time { return info.validationStart },
		func(info TransactionTimingInfo) time.Time { return info.validationEnd },
		"validations",
	)
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

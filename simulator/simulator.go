package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-simulator/helpers"
	"github.com/op/go-logging"
)

var logger *logging.Logger

// SetLogger ...
func SetLogger(l *logging.Logger) {
	logger = l
}

var sysParams helpers.SystemParameters
var execParams ExecutionParameters = ExecutionParameters{
	cryptoEvents:       make(map[CryptoEvent]int, 0),
	transactionTimings: make([]TransactionTimingInfo, 0),
}

// Simulate ...
func Simulate(rootSk dac.SK, params *helpers.SystemParameters) (e error) {

	sysParams = *params

	start := time.Now()

	execParams.network = MakeNetwork(helpers.NewRand(), rootSk)

	var wgUser sync.WaitGroup
	wgUser.Add(sysParams.Orgs * sysParams.Users)

	for user := 0; user < sysParams.Orgs*sysParams.Users; user++ {

		if sysParams.Revoke {
			execParams.network.users[user].requestNonRevocation()
		}

		go func(user int) {
			defer wgUser.Done()

			// first sleep uniform
			if sysParams.Frequency > 0 {
				time.Sleep(time.Duration(rand.Intn(sysParams.Frequency*1000)) * time.Millisecond)
			}

			for i := 0; i < sysParams.Transactions; i++ {
				userObj := execParams.network.users[user]

				// subsequent sleeps Poisson
				if sysParams.Frequency > 0 {
					sleep := time.Duration((3600.0/userObj.poisson.Rand())*1000) * time.Millisecond
					logger.Debugf("user-%d will wait %d ms", user, sleep.Milliseconds())
					time.Sleep(sleep)
				}

				message := helpers.RandomString(helpers.NewRand(), 16)
				userObj.submitTransaction(message)
			}

		}(user)
	}

	wgUser.Wait()

	// Auditing
	if sysParams.Audit {

		logger.Noticef("Audit started over %d transactions", len(execParams.network.transactions))

		for _, transaction := range execParams.network.transactions {
			authorPk := transaction.auditEnc.AuditingDecrypt(execParams.network.auditor.sk)
			recordCryptoEvent(auditDecrypt)
			if !dac.PkEqual(authorPk, execParams.network.users[transaction.proposal.authorID].CredentialsHolder.pk) {
				panic("auditing failed")
			}
		}

		logger.Notice("Audit completed")

	}

	execParams.network.stop()

	logger.Noticef("Simulation completed in %d seconds", int(math.Round(time.Since(start).Seconds())))

	if len(execParams.transactionTimings) > 0 {
		printStats()
	}

	return
}

func printStats() {

	// crypto events
	logger.Critical("Crypto events:")
	for event, times := range execParams.cryptoEvents {
		logger.Criticalf("\t%-20s : %3d : (%4.1f per transaction)\n", event, times, float64(times)/float64(len(execParams.network.transactions)))
	}

	// transaction timings
	logger.Criticalf("For %d transactions", len(execParams.transactionTimings))
	printTimingBasics := func(
		start func(TransactionTimingInfo) time.Time,
		end func(TransactionTimingInfo) time.Time,
		description string,
	) {
		var min, max, total, avg time.Duration
		var totals = make([]time.Duration, 0, len(execParams.transactionTimings))
		min = time.Duration(3600000 * time.Second)
		total = 0
		max = 0

		for _, info := range execParams.transactionTimings {
			elapsed := end(info).Sub(start(info))
			if elapsed < min {
				min = elapsed
			}
			if elapsed > max {
				max = elapsed
			}
			total += elapsed
			totals = append(totals, elapsed)
		}
		avg = time.Duration(total.Nanoseconds() / int64(len(execParams.transactionTimings)))

		sort.Slice(totals, func(i, j int) bool {
			return totals[i] < totals[j]
		})

		logger.Criticalf("%15s : min %4d ms, max %4d ms, avg %4d ms, median: %d ms\n", description, min.Milliseconds(), max.Milliseconds(), avg.Milliseconds(), totals[len(totals)/2].Milliseconds())
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

// ExecutionParameters ...
type ExecutionParameters struct {
	network            *Network
	cryptoEvents       map[CryptoEvent]int
	transactionTimings []TransactionTimingInfo
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

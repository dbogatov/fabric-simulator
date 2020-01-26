package main

import (
	"fmt"
	"sync"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

var sysParams SystemParameters

func simulate(prg *amcl.RAND, rootSk dac.SK) (e error) {

	network := MakeNetwork(prg, rootSk)

	// playing with endorsements
	var wg sync.WaitGroup
	wg.Add(sysParams.peers)

	for peer := 0; peer < sysParams.peers; peer++ {
		go func(peer int) {
			defer wg.Done()

			tp := TransactionProposal{
				id:          peer * 2,
				payloadSize: 100,
				from:        fmt.Sprintf("user-%d", peer),
				doneChannel: make(chan Endorsement),
			}
			network.peers[peer].tpChannel <- tp
			endorsement := <-tp.doneChannel
			logger.Debugf("Got endorsement %d", endorsement.signature)
		}(peer)
	}

	wg.Wait()

	for index := 0; index < 10; index++ {
		tp := TransactionProposal{
			id:          index * 10,
			payloadSize: 100,
			from:        fmt.Sprintf("user-%d", index),
			doneChannel: make(chan Endorsement),
		}
		network.peers[0].tpChannel <- tp
		endorsement := <-tp.doneChannel
		logger.Debugf("Got endorsement %d", endorsement.signature)
	}

	network.stop()

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

package simulator

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/dbogatov/dac-lib/dac"
)

type transferable interface {
	size() int
	name() string
}

// CertificateSize ...
const CertificateSize = 734 // TODO http://fm4dd.com/openssl/certexamples.shtm

var globalBandwidthLock = &sync.Mutex{}
var bandwidthLoggingLock = &sync.Mutex{}
var getLocksLock = &sync.Mutex{}
var networkEventID uint64 = 1

var connectionsMap = make(map[string]*sync.Mutex)

func recordBandwidth(from, to string, object transferable) {

	getLocks := func(key string) *sync.Mutex {
		getLocksLock.Lock()
		defer getLocksLock.Unlock()

		lock, exists := connectionsMap[key]
		if !exists {
			lock = &sync.Mutex{}
			connectionsMap[key] = lock
		}
		return lock
	}

	getWaitTime := func(bandwidth int) time.Duration {
		return time.Duration(1000*(float64(object.size())/float64(bandwidth))) * time.Millisecond
	}

	fromLock := getLocks(from)
	toLock := getLocks(to)

	var wg sync.WaitGroup
	wg.Add(3)

	spinWait := func(lock *sync.Mutex, waitTime time.Duration) {
		defer wg.Done()

		lock.Lock()
		defer lock.Unlock()

		time.Sleep(waitTime)
	}

	waitTimeGlobal := getWaitTime(sysParams.bandwidthGlobal)
	waitTimeLocal := getWaitTime(sysParams.bandwidthLocal)
	start := time.Now()

	go spinWait(fromLock, waitTimeLocal)
	go spinWait(toLock, waitTimeLocal)
	go spinWait(globalBandwidthLock, waitTimeGlobal)

	wg.Wait()

	end := time.Now()

	bandwidthLoggingLock.Lock()

	event, err := json.Marshal(NetworkEvent{
		From:            from,
		To:              to,
		Object:          object.name(),
		Size:            object.size(),
		Start:           start.Format(time.RFC3339Nano),
		End:             end.Format(time.RFC3339Nano),
		LocalBandwidth:  sysParams.bandwidthLocal,
		GlobalBandwidth: sysParams.bandwidthGlobal,
		ID:              networkEventID,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("%s,\n", string(event))

	logger.Debugf("%s sent %d bytes of %s to %s\n", from, object.size(), object.name(), to)

	networkEventID++

	bandwidthLoggingLock.Unlock()
}

// NetworkEvent ...
type NetworkEvent struct {
	From            string
	To              string
	Object          string
	Size            int
	Start           string
	End             string
	GlobalBandwidth int
	LocalBandwidth  int
	ID              uint64
}

/// Credentials

// Credentials ...
type Credentials struct {
	*dac.Credentials
}

func (creds Credentials) size() int {
	return len(creds.ToBytes())
}

func (creds Credentials) name() string {
	return "credentials"
}

/// CredRequest

// CredRequest ...
type CredRequest struct {
	*dac.CredRequest
}

func (credReq CredRequest) size() int {
	return len(credReq.ToBytes())
}

func (credReq CredRequest) name() string {
	return "cred-request"
}

/// Nonce

// Nonce ...
type Nonce struct {
	bytes []byte
}

func (nonce Nonce) size() int {
	return len(nonce.bytes)
}

func (nonce Nonce) name() string {
	return "nonce"
}

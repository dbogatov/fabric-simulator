package simulator

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/op/go-logging"
)

var logger *logging.Logger

// SetLogger ...
func SetLogger(l *logging.Logger) {
	logger = l
}

func randomBytes(prg *amcl.RAND, n int) (bytes []byte) {

	bytes = make([]byte, n)
	for i := 0; i < n; i++ {
		bytes[i] = prg.GetByte()
	}

	return
}

func peerByHash(hash []byte, peers int) (peer int) {
	input := new(big.Int)
	input.SetBytes(hash)

	divisor := new(big.Int)
	divisor.SetInt64(int64(peers))

	result := new(big.Int)
	result = result.Mod(input, divisor)

	peer = int(result.Int64())

	return
}

func sha3(raw []byte) (hash []byte) {

	recordCryptoEvent(sha3hash)

	hash = make([]byte, 32)
	sha3 := amcl.NewSHA3(amcl.SHA3_HASH256)
	for i := 0; i < len(raw); i++ {
		sha3.Process(raw[i])
	}
	sha3.Hash(hash[:])

	return
}

func randomString(prg *amcl.RAND, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		r := prg.GetByte()
		b[i] = charset[int(r)%len(charset)]
	}
	return string(b)
}

func randomULong(prg *amcl.RAND) uint64 {
	var raw [8]byte
	for i := 0; i < 8; i++ {
		raw[i] = prg.GetByte()
	}
	return binary.BigEndian.Uint64(raw[:])
}

var randMutex = &sync.Mutex{}

func newRand() (prg *amcl.RAND) {

	randMutex.Lock()
	defer randMutex.Unlock()

	prg = amcl.NewRAND()
	goPrg := rand.New(rand.NewSource(time.Now().UnixNano()))
	var raw [32]byte
	for i := 0; i < 32; i++ {
		raw[i] = byte(goPrg.Int())
	}
	prg.Seed(32, raw[:])

	return
}

func newRandSeed(seed []byte) (prg *amcl.RAND) {

	prg = amcl.NewRAND()
	prg.Seed(len(seed), seed)

	return
}

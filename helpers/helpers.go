package helpers

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/dbogatov/fabric-amcl/amcl"
)

// RandomBytes ...
func RandomBytes(prg *amcl.RAND, n int) (bytes []byte) {

	bytes = make([]byte, n)
	for i := 0; i < n; i++ {
		bytes[i] = prg.GetByte()
	}

	return
}

// PeerByHash ...
func PeerByHash(hash []byte, peers int) (peer int) {
	input := new(big.Int)
	input.SetBytes(hash)

	divisor := new(big.Int)
	divisor.SetInt64(int64(peers))

	result := new(big.Int)
	result = result.Mod(input, divisor)

	peer = int(result.Int64())

	return
}

// Sha3 ...
func Sha3(raw []byte) (hash []byte) {

	hash = make([]byte, 32)
	sha3 := amcl.NewSHA3(amcl.SHA3_HASH256)
	for i := 0; i < len(raw); i++ {
		sha3.Process(raw[i])
	}
	sha3.Hash(hash[:])

	return
}

// RandomString ...
func RandomString(prg *amcl.RAND, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		r := prg.GetByte()
		b[i] = charset[int(r)%len(charset)]
	}
	return string(b)
}

// RandomULong ...
func RandomULong(prg *amcl.RAND) uint64 {
	var raw [8]byte
	for i := 0; i < 8; i++ {
		raw[i] = prg.GetByte()
	}
	return binary.BigEndian.Uint64(raw[:])
}

var randMutex = &sync.Mutex{}

// NewRand ...
func NewRand() (prg *amcl.RAND) {

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

// NewRandSeed ...
func NewRandSeed(seed []byte) (prg *amcl.RAND) {

	prg = amcl.NewRAND()
	prg.Seed(len(seed), seed)

	return
}

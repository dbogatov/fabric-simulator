package main

import (
	"math/big"

	"github.com/dbogatov/fabric-amcl/amcl"
)

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

	hash = make([]byte, 32)
	sha3 := amcl.NewSHA3(amcl.SHA3_HASH256)
	for i := 0; i < len(raw); i++ {
		sha3.Process(raw[i])
	}
	sha3.Hash(hash[:])

	return
}

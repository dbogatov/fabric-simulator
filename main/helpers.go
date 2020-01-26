package main

import "github.com/dbogatov/fabric-amcl/amcl"

func randomBytes(prg *amcl.RAND, n int) (bytes []byte) {

	bytes = make([]byte, n)
	for i := 0; i < n; i++ {
		bytes[i] = prg.GetByte()
	}

	return
}

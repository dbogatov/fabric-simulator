package revocation

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"golang.org/x/sync/semaphore"
)

type signatureIndex struct {
	signatureBytes []byte
	index          int
}

// StartRequests ...
func StartRequests(runs, concurrent int, trust bool) {
	// a hack to get revocation PK for all clients
	_, groth, _, revokePk, epoch := generateAuthority()
	g := FP256BN.ECP_generator()

	prg := amcl.NewRAND()
	prg.Seed(1, []byte{0x14})

	pks := make([]dac.PK, runs)
	pkBytes := make([][]byte, runs)
	for i := 0; i < runs; i++ {
		_, pk := dac.GenerateKeys(prg, 1)
		pks[i] = pk
		pkBytes[i] = dac.PointToBytes(pk)
	}

	var wgRequest sync.WaitGroup
	wgRequest.Add(runs)
	signatures := make(chan signatureIndex, runs)
	ctx := context.TODO()
	sem := semaphore.NewWeighted(int64(concurrent))

	logger.Noticef("Starting... (%d runs, %d concurrent)", runs, concurrent)
	start := time.Now()

	for i := 0; i < runs; i++ {
		if e := sem.Acquire(ctx, 1); e != nil {
			panic(e)
		}

		go func(i int, sem *semaphore.Weighted) {
			defer wgRequest.Done()
			defer sem.Release(1)

			response, err := http.Post("http://localhost:8765", "application/octet-stream", bytes.NewBuffer(pkBytes[i]))
			if err != nil {
				panic(err)
			}
			defer response.Body.Close()

			if !trust {
				body, _ := ioutil.ReadAll(response.Body)

				signatures <- signatureIndex{
					signatureBytes: body,
					index:          i,
				}
			}
			logger.Debugf("Signature %d received", i)
		}(i, sem)
	}

	wgRequest.Wait()

	elapsed := time.Now().Sub(start)
	logger.Noticef("Requests completed in %d ms (%.1f requests per second).", elapsed.Milliseconds(), float64(runs)/float64(elapsed.Seconds()))

	close(signatures)

	if !trust {
		logger.Notice("Verifying all signatures... (can take up quite some time)")

		var wgVerify sync.WaitGroup
		wgVerify.Add(runs)

		for signature := range signatures {
			go func(signature dac.GrothSignature, i int) {
				defer wgVerify.Done()

				if e := groth.Verify(revokePk, signature, []interface{}{pks[i], g.Mul(epoch)}); e != nil {
					panic(e)
				}
			}(*dac.GrothSignatureFromBytes(signature.signatureBytes), signature.index)
		}

		wgVerify.Wait()
	}

	logger.Notice("Done.")
}

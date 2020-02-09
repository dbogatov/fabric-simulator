package revocation

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

// StartRequests ...
func StartRequests() {

	// a hack to get revocation PK for all clients
	_, groth, _, revokePk, epoch := generateAuthority()
	g := FP256BN.ECP_generator()

	prg := amcl.NewRAND()
	prg.Seed(1, []byte{0x14})

	_, pk := dac.GenerateKeys(prg, 1)

	pkBytes := dac.PointToBytes(pk)
	const RUNS = 100

	var wgUser sync.WaitGroup
	wgUser.Add(RUNS)

	logger.Notice("Starting...")

	for i := 0; i < RUNS; i++ {
		go func(i int) {
			defer wgUser.Done()

			response, err := http.Post("http://localhost:8765", "application/octet-stream", bytes.NewBuffer(pkBytes))
			if err != nil {
				panic(err)
			}
			defer response.Body.Close()

			body, _ := ioutil.ReadAll(response.Body)

			signature := dac.GrothSignatureFromBytes(body)
			if e := groth.Verify(revokePk, *signature, []interface{}{pk, g.Mul(epoch)}); e != nil {
				panic(e)
			}

			logger.Debugf("Signature %d verified", i)
		}(i)
	}

	wgUser.Wait()

	logger.Notice("Done.")
}

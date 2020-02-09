package revocation

import (
	"io/ioutil"
	"net/http"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

var sk dac.SK
var pk dac.PK
var ys []interface{}
var epoch *FP256BN.BIG

// RunServer ...
func RunServer() {
	logger.Notice("Server starting. Ctl+C to stop")

	ys, _, sk, pk, epoch = generateAuthority()

	http.HandleFunc("/", handleRevocationRequest)
	http.ListenAndServe(":8765", nil)
}

func handleRevocationRequest(w http.ResponseWriter, r *http.Request) {
	// static PRG... OK for simulations
	prg := amcl.NewRAND()
	prg.Seed(1, []byte{0x15})

	userPKBytes, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	userPK, e := dac.PointFromBytes(userPKBytes)
	if e != nil {
		panic(e)
	}

	signature := dac.SignNonRevoke(prg, sk, userPK, epoch, ys)
	signatureBytes := signature.ToBytes()

	w.Write(signatureBytes)
}

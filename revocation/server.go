package revocation

import (
	"io/ioutil"
	"net/http"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

var skRevoke dac.SK
var ys []interface{}
var epoch *FP256BN.BIG

// RunServer ...
func RunServer() {
	logger.Notice("Server starting. Ctl+C to stop")

	ys, _, skRevoke, _, epoch = generateAuthority()

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

	signature := dac.SignNonRevoke(prg, skRevoke, userPK, epoch, ys)
	signatureBytes := signature.ToBytes()

	logger.Debug("Granting handle.")

	w.Write(signatureBytes)
}

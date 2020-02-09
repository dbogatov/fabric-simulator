package revocation

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/op/go-logging"
)

var logger *logging.Logger

// SetLogger ...
func SetLogger(l *logging.Logger) {
	logger = l
}

func generateAuthority() (ys []interface{}, groth *dac.Groth, sk dac.SK, pk dac.PK, epoch *FP256BN.BIG) {

	prg := amcl.NewRAND()
	prg.Seed(1, []byte{0x13})

	ys = dac.GenerateYs(true, 10, prg)
	groth = dac.MakeGroth(prg, true, ys)
	sk, pk = groth.Generate()

	epoch = FP256BN.NewBIGint(5)

	return
}

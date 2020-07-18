package distributed

import (
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-simulator/helpers"
	"github.com/op/go-logging"
)

var logger *logging.Logger

// SetLogger ...
func SetLogger(l *logging.Logger) {
	logger = l
}

// Simulate ...
func Simulate(rootSk dac.SK, params *helpers.SystemParameters) (e error) {

	logger.Info("hello")

	return
}

// sysParams\.([a-z]+)
// sysParams.\u$1

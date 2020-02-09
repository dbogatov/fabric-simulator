package revocation

import "github.com/op/go-logging"

var logger *logging.Logger

// SetLogger ...
func SetLogger(l *logging.Logger) {
	logger = l
}

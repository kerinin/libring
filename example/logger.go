package main

import (
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("libring.example")

func init() {
	logging.SetLevel(logging.DEBUG, "libring.example")
}

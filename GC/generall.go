package GC

import (
	"log"
)

var PRINT_LOG = true

func printLogF(str string, is ...interface{}) {
	if PRINT_LOG {
		log.Printf(str, is...)
	}
}
package main

import (
	"github.com/apache/pulsar/pulsar-function-go/pf"

	pulsarFlogoTrigger "github.com/jdattatr-tibco/messaging-contrib/pulsar/trigger/function"
)

func main() {
	pf.Start(pulsarFlogoTrigger.Invoke)
}

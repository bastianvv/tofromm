package main

import (
	_ "embed"

	"github.com/bastianvv/tofromm/cmd"
)

//go:embed packaging/io.github.bastianvv.tofromm-daemon.service
var daemonService []byte

func main() {
	cmd.DaemonServiceFile = daemonService
	cmd.Execute()
}

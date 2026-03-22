package main

import (
	"os"

	"github.com/aiperceivable/unirelease/cmd"
)

// version is set by -ldflags at build time.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

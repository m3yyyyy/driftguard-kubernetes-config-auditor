package main

import (
	"os"

	"github.com/m3yyyyy/driftguard-kubernetes-config-auditor/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}

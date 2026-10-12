package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/Findddx/pku-drive-cli/internal/cli"
	versionpkg "github.com/Findddx/pku-drive-cli/internal/version"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	configureProcessSecurity()
	ctx, stop := signal.NotifyContext(context.Background(), processSignals()...)
	defer stop()
	code := cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, wireDependencies(), versionpkg.Info{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
	})
	os.Exit(code)
}

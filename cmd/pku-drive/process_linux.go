//go:build linux

package main

import (
	"os"
	"syscall"
)

func configureProcessSecurity() { syscall.Umask(0o077) }

func processSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM} }

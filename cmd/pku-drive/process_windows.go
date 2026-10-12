//go:build windows

package main

import "os"

// Windows access control is enforced per file and directory; it has no umask.
func configureProcessSecurity() {}

func processSignals() []os.Signal { return []os.Signal{os.Interrupt} }

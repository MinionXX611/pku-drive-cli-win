//go:build !windows

package cli

func commandSupported(string) bool { return true }

func shareModeSupported() bool { return true }

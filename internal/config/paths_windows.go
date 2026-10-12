//go:build windows

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultPaths keeps user configuration in Roaming AppData and disposable
// state in Local AppData. Both roots are per-user on supported Windows hosts.
func DefaultPaths() (Paths, error) {
	configHome := os.Getenv("APPDATA")
	stateHome := os.Getenv("LOCALAPPDATA")
	if configHome == "" || stateHome == "" {
		fallback, err := os.UserConfigDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve Windows application-data directory: %w", err)
		}
		if configHome == "" {
			configHome = fallback
		}
		if stateHome == "" {
			stateHome = fallback
		}
	}
	configDir := filepath.Join(configHome, "pku-drive")
	stateDir := filepath.Join(stateHome, "pku-drive")
	return Paths{
		ConfigDir:       configDir,
		ConfigFile:      filepath.Join(configDir, "config.json"),
		CredentialsFile: filepath.Join(configDir, "credentials.json"),
		CredentialLock:  filepath.Join(configDir, "credentials.lock"),
		UploadStateDir:  filepath.Join(stateDir, "uploads"),
	}, nil
}

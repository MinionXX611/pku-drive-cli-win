//go:build !windows

package config

import (
	"os"
	"path/filepath"
)

// DefaultPaths returns paths following the XDG base directory specification.
func DefaultPaths() (Paths, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	stateHome := os.Getenv("XDG_STATE_HOME")
	if configHome == "" || stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		if stateHome == "" {
			stateHome = filepath.Join(home, ".local", "state")
		}
	}
	configDir := filepath.Join(configHome, "pku-drive-cli")
	return Paths{
		ConfigDir:       configDir,
		ConfigFile:      filepath.Join(configDir, "config.json"),
		CredentialsFile: filepath.Join(configDir, "credentials.json"),
		CredentialLock:  filepath.Join(configDir, "credentials.lock"),
		UploadStateDir:  filepath.Join(stateHome, "pku-drive-cli", "uploads"),
	}, nil
}

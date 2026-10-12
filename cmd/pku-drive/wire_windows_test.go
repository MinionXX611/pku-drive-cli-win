//go:build windows

package main

import (
	"path/filepath"
	"testing"

	"github.com/Findddx/pku-drive-cli/internal/config"
)

func TestWindowsMVPWiresOnlySupportedOperations(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config", "pku-drive")
	runtime, err := newWiredRuntime(wireOptions{defaultPaths: func() (config.Paths, error) {
		return config.Paths{
			ConfigDir:       configDir,
			ConfigFile:      filepath.Join(configDir, "config.json"),
			CredentialsFile: filepath.Join(configDir, "credentials.json"),
			CredentialLock:  filepath.Join(configDir, "credentials.lock"),
			UploadStateDir:  filepath.Join(root, "state", "pku-drive", "uploads"),
		}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	deps := runtime.dependencies
	if deps.Login == nil || deps.Status == nil || deps.List == nil || deps.Delete == nil || deps.Logout == nil {
		t.Fatal("Windows MVP dependencies are incomplete")
	}
	if deps.Mkdir != nil || deps.Put != nil || deps.Get != nil || deps.OpenShare != nil {
		t.Fatal("Windows MVP unexpectedly enabled unsupported local-I/O operations")
	}
}

//go:build windows

package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsUsesWindowsApplicationData(t *testing.T) {
	roaming := filepath.Join(t.TempDir(), "Roaming")
	local := filepath.Join(t.TempDir(), "Local")
	t.Setenv("APPDATA", roaming)
	t.Setenv("LOCALAPPDATA", local)
	paths, err := DefaultPaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.ConfigDir != filepath.Join(roaming, "pku-drive") || paths.ConfigFile != filepath.Join(roaming, "pku-drive", "config.json") || paths.CredentialsFile != filepath.Join(roaming, "pku-drive", "credentials.json") || paths.CredentialLock != filepath.Join(roaming, "pku-drive", "credentials.lock") || paths.UploadStateDir != filepath.Join(local, "pku-drive", "uploads") {
		t.Fatalf("unexpected Windows paths: %+v", paths)
	}
}

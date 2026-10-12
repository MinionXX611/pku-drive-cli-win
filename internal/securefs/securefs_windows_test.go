//go:build windows

package securefs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedJSONRoundTrip(t *testing.T) {
	dir, err := EnsurePrivateDir(filepath.Join(t.TempDir(), "pku-drive"))
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if err := WriteJSONAtomicAt(dir, "credentials.json", map[string]string{"token": "secret"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir.path, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "{\"token\":\"secret\"}\n" || !bytesHavePrefix(data, dpapiFileMagic) {
		t.Fatal("credentials were not stored with DPAPI protection")
	}
	if err := WriteJSONAtomicAt(dir, "credentials.json", map[string]string{"token": "refreshed"}); err != nil {
		t.Fatalf("replace existing protected credentials: %v", err)
	}
	var got map[string]string
	if err := ReadJSON0600At(dir, "credentials.json", &got); err != nil || got["token"] != "refreshed" {
		t.Fatalf("credentials = %#v, err = %v", got, err)
	}
}

func TestProtectedJSONRejectsSymlink(t *testing.T) {
	dir, err := EnsurePrivateDir(filepath.Join(t.TempDir(), "pku-drive"))
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	target := filepath.Join(dir.path, "target.json")
	if err := os.WriteFile(target, []byte("not used"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir.path, "credentials.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := WriteJSONAtomicAt(dir, "credentials.json", map[string]string{"token": "secret"}); err == nil {
		t.Fatal("expected symlink destination rejection")
	}
}

func TestCredentialLockHonorsContext(t *testing.T) {
	dir, err := EnsurePrivateDir(filepath.Join(t.TempDir(), "pku-drive"))
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = WithLockAt(context.Background(), dir, "credentials.lock", func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	cancel()
	err = WithLockAt(ctx, dir, "credentials.lock", func() error { return nil })
	close(release)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lock error = %v, want cancellation", err)
	}
}

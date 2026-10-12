//go:build windows

package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Findddx/pku-drive-cli/internal/cli"
	"github.com/Findddx/pku-drive-cli/internal/version"
)

func TestWindowsMVPRejectsUnsupportedCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"put", "local.txt", "/remote.txt", "--json"}, strings.NewReader(""), &stdout, &stderr, cli.Dependencies{}, version.Info{})
	if code != 2 || stdout.String() != "{\"category\":\"usage\",\"message\":\"command is not supported in this build\",\"ok\":false}\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestWindowsMVPInvokesDelete(t *testing.T) {
	var stdout, stderr bytes.Buffer
	called := false
	deps := cli.Dependencies{Delete: func(_ context.Context, path string, recursive bool) (cli.DeleteResult, error) {
		called = path == "/docs/old.txt" && !recursive
		return cli.DeleteResult{RemotePath: path, Status: "recycled"}, nil
	}}
	code := cli.Run(context.Background(), []string{"rm", "/docs/old.txt"}, strings.NewReader(""), &stdout, &stderr, deps, version.Info{})
	if code != 0 || !called || stdout.String() != "Moved to recycle bin: /docs/old.txt\n" {
		t.Fatalf("code=%d called=%v stdout=%q stderr=%q", code, called, stdout.String(), stderr.String())
	}
}

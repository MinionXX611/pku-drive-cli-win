//go:build windows

package oauth

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// SystemBrowser opens authorization URLs through the registered Windows
// browser handler. The URL is passed as an argument rather than interpolated
// into cmd.exe, so query values cannot be interpreted as shell syntax.
type SystemBrowser struct {
	Command func(context.Context, string, ...string) error
	Warning io.Writer
}

func NewSystemBrowser() *SystemBrowser {
	return &SystemBrowser{
		Command: func(ctx context.Context, name string, args ...string) error {
			return exec.CommandContext(ctx, name, args...).Start()
		},
		Warning: os.Stderr,
	}
}

func (b *SystemBrowser) Open(ctx context.Context, rawURL string) error {
	command := func(ctx context.Context, name string, args ...string) error {
		return exec.CommandContext(ctx, name, args...).Start()
	}
	warning := io.Discard
	if b != nil {
		if b.Command != nil {
			command = b.Command
		}
		if b.Warning != nil {
			warning = b.Warning
		}
	}
	if err := command(ctx, "rundll32.exe", "url.dll,FileProtocolHandler", rawURL); err != nil {
		_, _ = fmt.Fprintln(warning, "browser launch failed; open the authorization URL manually")
	}
	return nil
}

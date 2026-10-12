//go:build !windows

package oauth

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type SystemBrowser struct {
	LookupEnv func(string) (string, bool)
	Command   func(context.Context, string, ...string) error
	Warning   io.Writer
}

func NewSystemBrowser() *SystemBrowser {
	return &SystemBrowser{
		LookupEnv: os.LookupEnv,
		Command: func(ctx context.Context, name string, args ...string) error {
			return exec.CommandContext(ctx, name, args...).Run()
		},
		Warning: os.Stderr,
	}
}

func (b *SystemBrowser) Open(ctx context.Context, rawURL string) error {
	lookup := os.LookupEnv
	command := func(ctx context.Context, name string, args ...string) error {
		return exec.CommandContext(ctx, name, args...).Run()
	}
	warning := io.Discard
	if b != nil {
		if b.LookupEnv != nil {
			lookup = b.LookupEnv
		}
		if b.Command != nil {
			command = b.Command
		}
		if b.Warning != nil {
			warning = b.Warning
		}
	}
	display, _ := lookup("DISPLAY")
	waylandDisplay, _ := lookup("WAYLAND_DISPLAY")
	if display == "" && waylandDisplay == "" {
		return nil
	}
	if err := command(ctx, "xdg-open", rawURL); err != nil {
		_, _ = fmt.Fprintln(warning, "browser launch failed; open the authorization URL manually")
	}
	return nil
}

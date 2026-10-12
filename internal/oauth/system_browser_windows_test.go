//go:build windows

package oauth

import (
	"context"
	"io"
	"testing"
)

func TestWindowsSystemBrowserUsesRegisteredURLHandler(t *testing.T) {
	var gotName string
	var gotArgs []string
	browser := &SystemBrowser{Warning: io.Discard, Command: func(_ context.Context, name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	}}
	if err := browser.Open(context.Background(), "https://disk.pku.edu.cn/oauth"); err != nil {
		t.Fatal(err)
	}
	if gotName != "rundll32.exe" || len(gotArgs) != 2 || gotArgs[0] != "url.dll,FileProtocolHandler" || gotArgs[1] != "https://disk.pku.edu.cn/oauth" {
		t.Fatalf("browser command = %q %#v", gotName, gotArgs)
	}
}

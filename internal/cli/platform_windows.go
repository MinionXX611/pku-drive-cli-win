//go:build windows

package cli

// The first Windows release intentionally exposes only the remote operations
// whose native filesystem behavior is not part of the execution path.
func commandSupported(command string) bool {
	switch command {
	case "login", "status", "ls", "rm", "logout", "version":
		return true
	default:
		return false
	}
}

func shareModeSupported() bool { return false }

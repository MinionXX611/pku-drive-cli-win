//go:build windows

package anyshare

import "os"

// Windows MVP has no machine-wide, administrator-managed SPKI pin location.
// A valid process-level override remains supported through
// PKU_DRIVE_OBJECT_SPKI_SHA256; returning ErrNotExist preserves the normal
// system trust-store path when no override is configured.
type systemObjectPinLocation struct{}

var pkuSystemObjectPinLocation systemObjectPinLocation

func readSystemObjectPinConfig(systemObjectPinLocation) (objectPinConfig, error) {
	return objectPinConfig{}, os.ErrNotExist
}

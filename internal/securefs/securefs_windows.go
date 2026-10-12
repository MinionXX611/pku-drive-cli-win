//go:build windows

// Package securefs provides small, platform-specific secure persistence helpers.
package securefs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	fileAttributeReparsePoint = 0x00000400
	cryptProtectUIForbidden   = 0x00000001
	lockfileFailImmediately   = 0x00000001
	lockfileExclusiveLock     = 0x00000002
	errorLockViolation        = syscall.Errno(33)
)

var (
	crypt32              = syscall.NewLazyDLL("crypt32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	cryptProtectData     = crypt32.NewProc("CryptProtectData")
	cryptUnprotectData   = crypt32.NewProc("CryptUnprotectData")
	localFree            = kernel32.NewProc("LocalFree")
	lockFileEx           = kernel32.NewProc("LockFileEx")
	unlockFileEx         = kernel32.NewProc("UnlockFileEx")
	moveFileEx           = kernel32.NewProc("MoveFileExW")
	dpapiFileMagic       = []byte("PKUDRIVE-DPAPI-v1\x00")
	errCallbackNoRestore = errors.New("callback input restore failed")
)

type dataBlob struct {
	Length uint32
	Data   *byte
}

// Dir remembers the identity of a private directory. Every operation checks
// that the original directory still occupies its path before using it, which
// rejects junction and leaf-swap redirection attacks.
type Dir struct {
	path   string
	info   os.FileInfo
	closed bool
}

func (d *Dir) Close() error {
	if d != nil {
		d.closed = true
	}
	return nil
}

func OpenPrivateDir(path string) (*Dir, error) {
	if err := validateAbsolutePath(path); err != nil {
		return nil, err
	}
	return openPrivateDir(path)
}

func EnsurePrivateDir(path string) (*Dir, error) {
	if err := validateAbsolutePath(path); err != nil {
		return nil, err
	}
	if err := ensureDirectoryTree(path); err != nil {
		return nil, err
	}
	return openPrivateDir(path)
}

func EnsurePrivateDirAt(parent *Dir, name string) (*Dir, error) {
	if parent == nil || parent.closed {
		return nil, errors.New("invalid parent directory")
	}
	if err := validateEntryName(name); err != nil {
		return nil, err
	}
	if err := parent.ensureCurrent(); err != nil {
		return nil, err
	}
	path := filepath.Join(parent.path, name)
	if err := ensureDirectoryTree(path); err != nil {
		return nil, err
	}
	return openPrivateDir(path)
}

func openPrivateDir(path string) (*Dir, error) {
	info, err := inspectDirectory(path)
	if err != nil {
		return nil, err
	}
	return &Dir{path: path, info: info}, nil
}

func (d *Dir) ensureCurrent() error {
	if d == nil || d.closed || d.info == nil {
		return errors.New("private directory is closed")
	}
	info, err := inspectDirectory(d.path)
	if err != nil {
		return err
	}
	if !os.SameFile(d.info, info) {
		return errors.New("private directory changed while in use")
	}
	return nil
}

func ensureDirectoryTree(path string) error {
	volume := filepath.VolumeName(path)
	root := volume + string(filepath.Separator)
	remaining := strings.TrimLeft(strings.TrimPrefix(path, volume), "\\/")
	if remaining == "" {
		return nil
	}
	current := root
	for _, component := range strings.FieldsFunc(remaining, func(r rune) bool { return r == '\\' || r == '/' }) {
		if component == "" || component == "." || component == ".." {
			return fmt.Errorf("unsafe directory component: %q", component)
		}
		current = filepath.Join(current, component)
		if _, err := os.Lstat(current); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("create private directory: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("inspect private directory: %w", err)
		}
		if _, err := inspectDirectory(current); err != nil {
			return err
		}
	}
	return nil
}

func inspectDirectory(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || isReparsePoint(path) {
		return nil, fmt.Errorf("unsafe private directory: %s", path)
	}
	return info, nil
}

func isReparsePoint(path string) bool {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return true
	}
	attributes, err := syscall.GetFileAttributes(pathPointer)
	return err != nil || attributes&fileAttributeReparsePoint != 0
}

func validateAbsolutePath(path string) error {
	if path == "" || strings.ContainsRune(path, '\x00') || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.HasSuffix(path, string(filepath.Separator)) {
		return fmt.Errorf("unsafe path: %q", path)
	}
	return nil
}

func validateEntryName(name string) error {
	if name == "" || strings.ContainsRune(name, '\x00') || strings.ContainsAny(name, "\\/") || filepath.Base(name) != name || name == "." || name == ".." {
		return fmt.Errorf("unsafe file name: %q", name)
	}
	return nil
}

func splitAbsoluteEntryPath(path string) (string, string, error) {
	if err := validateAbsolutePath(path); err != nil {
		return "", "", err
	}
	parent, name := filepath.Dir(path), filepath.Base(path)
	if err := validateEntryName(name); err != nil {
		return "", "", err
	}
	return parent, name, nil
}

func WriteJSONAtomic(path string, value any) error {
	parent, name, err := splitAbsoluteEntryPath(path)
	if err != nil {
		return err
	}
	dir, err := OpenPrivateDir(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	return WriteJSONAtomicAt(dir, name, value)
}

func WriteJSONAtomicAt(dir *Dir, name string, value any) error {
	path, err := dir.entryPath(name)
	if err != nil {
		return err
	}
	plain, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	plain = append(plain, '\n')
	protected, err := protectForCurrentUser(plain)
	if err != nil {
		return err
	}
	if err := validateExistingRegular(path); err != nil {
		return err
	}
	tmpPath, file, err := createPrivateTemp(dir.path, name)
	if err != nil {
		return err
	}
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := file.Write(append(dpapiFileMagic, protected...)); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := dir.ensureCurrent(); err != nil {
		return err
	}
	if err := validateExistingRegular(path); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, path); err != nil {
		return fmt.Errorf("replace protected JSON file: %w", err)
	}
	keepTemp = false
	return nil
}

func ReadJSON0600(path string, value any) error {
	parent, name, err := splitAbsoluteEntryPath(path)
	if err != nil {
		return err
	}
	dir, err := OpenPrivateDir(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	return ReadJSON0600At(dir, name, value)
}

func ReadJSON0600At(dir *Dir, name string, value any) error {
	path, err := dir.entryPath(name)
	if err != nil {
		return err
	}
	if err := validateRegular(path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read protected JSON file: %w", err)
	}
	if !bytesHavePrefix(data, dpapiFileMagic) {
		return errors.New("private JSON file is not DPAPI protected")
	}
	plain, err := unprotectForCurrentUser(data[len(dpapiFileMagic):])
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(plain)))
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("decode JSON: multiple values")
		}
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

func Remove0600(path string) error {
	parent, name, err := splitAbsoluteEntryPath(path)
	if err != nil {
		return err
	}
	dir, err := OpenPrivateDir(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	return Remove0600At(dir, name)
}

func Remove0600At(dir *Dir, name string) error {
	path, err := dir.entryPath(name)
	if err != nil {
		return err
	}
	if err := validateRegular(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := dir.ensureCurrent(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove protected JSON file: %w", err)
	}
	return nil
}

func (d *Dir) entryPath(name string) (string, error) {
	if err := validateEntryName(name); err != nil {
		return "", err
	}
	if err := d.ensureCurrent(); err != nil {
		return "", err
	}
	return filepath.Join(d.path, name), nil
}

func validateExistingRegular(path string) error {
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return validateRegular(path)
}

func validateRegular(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || isReparsePoint(path) {
		return fmt.Errorf("unsafe non-regular private file: %s", path)
	}
	return nil
}

func createPrivateTemp(dirPath, name string) (string, *os.File, error) {
	var random [16]byte
	for range 10 {
		if _, err := rand.Read(random[:]); err != nil {
			return "", nil, fmt.Errorf("random temporary name: %w", err)
		}
		path := filepath.Join(dirPath, "."+name+".tmp-"+hex.EncodeToString(random[:]))
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, fmt.Errorf("create private temporary file: %w", err)
		}
		return path, file, nil
	}
	return "", nil, errors.New("create private temporary file: too many name collisions")
}

func replaceFile(source, destination string) error {
	sourcePointer, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPointer, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	const moveFileReplaceExisting = 0x00000001
	const moveFileWriteThrough = 0x00000008
	ok, _, callErr := moveFileEx.Call(uintptr(unsafe.Pointer(sourcePointer)), uintptr(unsafe.Pointer(destinationPointer)), moveFileReplaceExisting|moveFileWriteThrough)
	if ok == 0 {
		return callErr
	}
	return nil
}

func WithLock(ctx context.Context, path string, fn func() error) error {
	parent, name, err := splitAbsoluteEntryPath(path)
	if err != nil {
		return err
	}
	dir, err := OpenPrivateDir(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	return WithLockAt(ctx, dir, name, fn)
}

func WithLockAt(ctx context.Context, dir *Dir, name string, fn func() error) error {
	path, err := dir.entryPath(name)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open credential lock: %w", err)
	}
	defer file.Close()
	if err := validateRegular(path); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := tryExclusiveLock(file)
		if err == nil {
			defer releaseExclusiveLock(file)
			return fn()
		}
		if !errors.Is(err, errorLockViolation) {
			return fmt.Errorf("lock credential file: %w", err)
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func tryExclusiveLock(file *os.File) error {
	overlapped := syscall.Overlapped{}
	ok, _, err := lockFileEx.Call(file.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if ok == 0 {
		return err
	}
	return nil
}

func releaseExclusiveLock(file *os.File) {
	overlapped := syscall.Overlapped{}
	_, _, _ = unlockFileEx.Call(file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
}

func protectForCurrentUser(plain []byte) ([]byte, error) {
	in := bytesToBlob(plain)
	var out dataBlob
	ok, _, err := cryptProtectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, cryptProtectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if ok == 0 {
		return nil, fmt.Errorf("protect private data: %w", err)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.Data)))
	return blobBytes(out), nil
}

func unprotectForCurrentUser(protected []byte) ([]byte, error) {
	in := bytesToBlob(protected)
	var out dataBlob
	ok, _, err := cryptUnprotectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, cryptProtectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if ok == 0 {
		return nil, fmt.Errorf("unprotect private data: %w", err)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.Data)))
	return blobBytes(out), nil
}

func bytesToBlob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{Length: uint32(len(data)), Data: &data[0]}
}

func blobBytes(blob dataBlob) []byte {
	if blob.Length == 0 || blob.Data == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice(blob.Data, int(blob.Length))...)
}

func bytesHavePrefix(data, prefix []byte) bool {
	return len(data) >= len(prefix) && string(data[:len(prefix)]) == string(prefix)
}

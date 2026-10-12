//go:build windows

// Package upload provides safe local primitives used by upload workflows.
package upload

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"

	"github.com/Findddx/pku-drive-cli/internal/apperr"
)

const (
	sliceSize             = 200 * 1024
	fingerprintBufferSize = 4 * 1024 * 1024
)

// Fingerprint retains the portable metadata used by the upload state store.
// Windows upload itself is not exposed by the MVP runtime yet; keeping this
// implementation conservative lets shared CLI types compile without claiming
// the Linux descriptor-binding guarantees on a different filesystem model.
type Fingerprint struct {
	Device   uint64 `json:"device"`
	Inode    uint64 `json:"inode"`
	Size     int64  `json:"size"`
	MtimeNS  int64  `json:"mtime_ns"`
	MD5      string `json:"md5"`
	SliceMD5 string `json:"slice_md5"`
	CRC32    string `json:"crc32"`
}

func OpenFingerprint(path string) (*os.File, Fingerprint, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, Fingerprint{}, localError("inspect upload file", err)
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, Fingerprint{}, localError("inspect upload file", fmt.Errorf("not a regular file"))
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Fingerprint{}, localError("open upload file", err)
	}
	fail := func(op string, cause error) (*os.File, Fingerprint, error) {
		_ = file.Close()
		return nil, Fingerprint{}, localError(op, cause)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(before, info) {
		if err == nil {
			err = fmt.Errorf("upload file changed while opening")
		}
		return fail("inspect upload file", err)
	}
	fullMD5 := md5.New()
	sliceMD5 := md5.New()
	crc := crc32.NewIEEE()
	prefix := &prefixHashWriter{hash: sliceMD5, remaining: sliceSize}
	reader := struct{ io.Reader }{Reader: file}
	if _, err := io.CopyBuffer(io.MultiWriter(fullMD5, crc, prefix), reader, make([]byte, fingerprintBufferSize)); err != nil {
		return fail("fingerprint upload file", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fail("rewind upload file", err)
	}
	return file, Fingerprint{Size: info.Size(), MtimeNS: info.ModTime().UnixNano(), MD5: upperHex(fullMD5.Sum(nil)), SliceMD5: upperHex(sliceMD5.Sum(nil)), CRC32: upperHex(crc.Sum(nil))}, nil
}

func (f Fingerprint) Unchanged(file *os.File) (bool, error) {
	if file == nil {
		return false, localError("inspect upload file", fmt.Errorf("missing file descriptor"))
	}
	info, err := file.Stat()
	if err != nil {
		return false, localError("inspect upload file", err)
	}
	return info.Mode().IsRegular() && info.Size() == f.Size && info.ModTime().UnixNano() == f.MtimeNS, nil
}

type prefixHashWriter struct {
	hash      hash.Hash
	remaining int
}

func (w *prefixHashWriter) Write(data []byte) (int, error) {
	if len(data) <= w.remaining {
		_, _ = w.hash.Write(data)
		w.remaining -= len(data)
	} else if w.remaining > 0 {
		_, _ = w.hash.Write(data[:w.remaining])
		w.remaining = 0
	}
	return len(data), nil
}

func upperHex(value []byte) string {
	encoded := make([]byte, hex.EncodedLen(len(value)))
	hex.Encode(encoded, value)
	for index, b := range encoded {
		if b >= 'a' && b <= 'f' {
			encoded[index] -= 'a' - 'A'
		}
	}
	return string(encoded)
}

func localError(operation string, cause error) error {
	return apperr.Wrap(apperr.Local, operation, "local file operation failed", cause)
}

//go:build windows

package oauth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
)

var errCallbackInputRestore = errors.New("callback input restore failed")

type callbackLineReader struct {
	input    io.Reader
	buffered *bufio.Reader
}

func newCallbackLineReader(input io.Reader) *callbackLineReader {
	if input == nil {
		return nil
	}
	return &callbackLineReader{input: input, buffered: bufio.NewReader(input)}
}

// ReadLine intentionally uses a visible prompt on Windows. Users paste a
// localhost callback URL, which is not a credential and should remain
// inspectable in terminals that do not expose a portable no-echo mode.
func (r *callbackLineReader) ReadLine(ctx context.Context, notice io.Writer, _, visiblePrompt string) (string, error) {
	if r == nil || r.input == nil {
		return "", io.EOF
	}
	if _, err := fmt.Fprint(notice, visiblePrompt); err != nil {
		return "", err
	}
	type result struct {
		line string
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		line, err := readBoundedBufferedLine(r.buffered)
		resultCh <- result{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case got := <-resultCh:
		if _, err := fmt.Fprintln(notice); err != nil && got.err == nil {
			got.err = err
		}
		return got.line, got.err
	}
}

func readBoundedBufferedLine(reader *bufio.Reader) (string, error) {
	line := make([]byte, 0, 512)
	for {
		one, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && len(line) > 0 {
				if line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				return string(line), nil
			}
			return "", err
		}
		if one == '\n' {
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			return string(line), nil
		}
		line = append(line, one)
		if len(line) > maximumPastedCallbackBytes {
			return "", invalidPastedCallback()
		}
	}
}

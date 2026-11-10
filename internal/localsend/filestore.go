package localsend

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// SafeRelativePath turns the name a sender offered into a path below a
// receive directory: no absolute path, no "." or ".." segment, and no
// character that a platform refuses in a file name. Folder sends keep
// their folders, as slashes.
func SafeRelativePath(name string) (string, error) {
	name = norm.NFC.String(strings.ReplaceAll(name, `\`, "/"))
	var parts []string
	for _, segment := range strings.Split(name, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." {
			continue
		}
		if segment == ".." {
			segment = "_"
		}
		segment = safeSegment(segment)
		if segment == "" {
			continue
		}
		parts = append(parts, segment)
	}
	if len(parts) == 0 {
		return "", errors.New("localsend: the file has no usable name")
	}
	return strings.Join(parts, "/"), nil
}

// safeSegment keeps segment a file system on any target accepts.
func safeSegment(segment string) string {
	var b strings.Builder
	for _, r := range segment {
		switch {
		case r == '/' || r == '\\' || r == '<' || r == '>' || r == ':' || r == '"' || r == '|' || r == '?' || r == '*':
			b.WriteRune('_')
		case unicode.IsControl(r):
			// Dropped, not replaced, so a name of controls becomes empty
			// and its caller replaces it.
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimRight(b.String(), ". ")
	out = strings.TrimSpace(out)
	if out == "" {
		return "_"
	}
	const max = 120
	if utf8.RuneCountInString(out) > max {
		runes := []rune(out)
		out = string(runes[:max])
	}
	if reservedName(out) {
		out = "_" + out
	}
	return out
}

// reservedName reports a Windows device name, which a file cannot be.
func reservedName(name string) bool {
	stem := name
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	switch strings.ToUpper(strings.TrimSpace(stem)) {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(stem) == 4 {
		upper := strings.ToUpper(stem)
		if (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
			return true
		}
	}
	return false
}

// ReceiveTarget is where one received file goes.
type ReceiveTarget struct {
	// Relative is the file's path below the receive directory, slashes
	// separated.
	Relative string
	// Path is the absolute target.
	Path string
}

// PlanReceive assigns accepted files destinations under dir, avoiding
// names that exist and names another file of the session took. reserved
// carries the absolute paths already planned and is updated.
func PlanReceive(dir string, files []File, reserved map[string]bool) ([]ReceiveTarget, error) {
	if dir == "" {
		return nil, errors.New("localsend: no receive directory")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if reserved == nil {
		reserved = map[string]bool{}
	}
	out := make([]ReceiveTarget, 0, len(files))
	for _, f := range files {
		rel, err := SafeRelativePath(f.FileName)
		if err != nil {
			rel = "file-" + f.ID
		}
		full, err := uniqueBelow(dir, rel, reserved)
		if err != nil {
			return nil, err
		}
		rel, err = filepath.Rel(dir, full)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		reserved[pathKey(full)] = true
		out = append(out, ReceiveTarget{Relative: rel, Path: full})
	}
	return out, nil
}

// uniqueBelow returns the absolute path below dir for rel that neither
// exists nor is reserved, adding " (1)", " (2)" and so on.
func uniqueBelow(dir, rel string, reserved map[string]bool) (string, error) {
	rel = filepath.FromSlash(rel)
	full := filepath.Join(dir, rel)
	if !within(dir, full) {
		return "", errors.New("localsend: receive path leaves its directory")
	}
	ext := filepath.Ext(full)
	stem := strings.TrimSuffix(full, ext)
	for i := 0; i < 10000; i++ {
		candidate := full
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		if reserved[pathKey(candidate)] {
			continue
		}
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("localsend: too many files with the same name")
}

// within reports whether path is dir or below it.
func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// pathKey is path as a reserved set compares it.
func pathKey(path string) string {
	path = filepath.Clean(path)
	if filepath.Separator == '\\' {
		return strings.ToLower(path)
	}
	return path
}

// WriteFile streams r to path, making its folders, and returns the
// SHA-256 and the bytes written. It writes beside the target and renames
// on success, so a failure leaves no partial file. progress receives the
// bytes written so far, at most about every 100 ms when it is set.
func WriteFile(path string, r io.Reader, size int64, progress func(written int64)) (written int64, hash string, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, "", err
	}
	tmp := path + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, "", err
	}
	ok := false
	defer func() {
		out.Close()
		if !ok {
			os.Remove(tmp)
		}
	}()
	h := sha256.New()
	w := &writeProgress{writer: io.MultiWriter(out, h), report: progress}
	if _, err := io.CopyBuffer(w, r, make([]byte, 256<<10)); err != nil {
		return w.n, "", err
	}
	if size >= 0 && w.n != size {
		return w.n, "", fmt.Errorf("localsend: received %d bytes of %d", w.n, size)
	}
	if err := out.Sync(); err != nil {
		return w.n, "", err
	}
	if err := out.Close(); err != nil {
		return w.n, "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return w.n, "", err
	}
	ok = true
	return w.n, hex.EncodeToString(h.Sum(nil)), nil
}

// writeProgress reports bytes written, throttled to about ten a second.
type writeProgress struct {
	writer io.Writer
	n      int64
	report func(int64)
	last   time.Time
}

func (w *writeProgress) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.n += int64(n)
	if w.report != nil {
		now := time.Now()
		if now.Sub(w.last) >= 100*time.Millisecond {
			w.last = now
			w.report(w.n)
		}
	}
	return n, err
}

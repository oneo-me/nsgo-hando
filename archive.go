package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// archiveEntry is one received file and its path in an archive.
type archiveEntry struct {
	path string
	name string
}

// hasReceivedFolders reports whether a session received a folder, which a
// sender names as paths below one.
func hasReceivedFolders(files []localsend.ReceivedFile) bool {
	for _, f := range files {
		if strings.Contains(strings.ReplaceAll(f.FileName, `\`, "/"), "/") {
			return true
		}
	}
	return false
}

// archiveReceivedFolders packs the files a session received below dir into
// one zip beside them and removes the files it packed. The zip takes the
// name of the folder they share, Hando otherwise. It returns the archive's
// path, or none when the session received no folder.
func archiveReceivedFolders(dir string, files []transferFile) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	entries, name := archiveEntries(dir, files)
	if len(entries) == 0 {
		return "", nil
	}
	targets, err := localsend.PlanReceive(dir, []localsend.File{{ID: "archive", FileName: name + ".zip"}}, nil)
	if err != nil {
		return "", err
	}
	path := targets[0].Path
	if err := writeArchive(path, entries); err != nil {
		return "", err
	}
	for _, entry := range entries {
		_ = os.Remove(entry.path)
		pruneEmptyDirs(filepath.Dir(entry.path), dir)
	}
	return path, nil
}

// archiveEntries lists the files to pack and the archive's name: the folder
// every file sits in when they share one, Hando otherwise.
func archiveEntries(dir string, files []transferFile) ([]archiveEntry, string) {
	var entries []archiveEntry
	roots := map[string]bool{}
	nested, allNested := false, true
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		rel, err := filepath.Rel(dir, f.Path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
			continue
		}
		entries = append(entries, archiveEntry{path: f.Path, name: rel})
		parts := strings.SplitN(rel, "/", 2)
		if len(parts) > 1 {
			nested = true
			roots[parts[0]] = true
		} else {
			allNested = false
		}
	}
	if !nested || len(entries) == 0 {
		return nil, ""
	}
	name := "Hando"
	if allNested && len(roots) == 1 {
		for root := range roots {
			name = root
		}
	}
	return entries, name
}

// writeArchive writes entries into a zip at path, beside it as a part file
// until it is complete, so a failure leaves no archive.
func writeArchive(path string, entries []archiveEntry) error {
	part := path + ".part"
	out, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		out.Close()
		if !ok {
			os.Remove(part)
		}
	}()
	zw := zip.NewWriter(out)
	for _, entry := range entries {
		in, err := os.Open(entry.path)
		if err != nil {
			zw.Close()
			return err
		}
		info, statErr := in.Stat()
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(0o644)
		if statErr == nil {
			header.Modified = info.ModTime()
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			in.Close()
			zw.Close()
			return err
		}
		if _, err := io.Copy(w, in); err != nil {
			in.Close()
			zw.Close()
			return fmt.Errorf("archiving %s: %w", entry.name, err)
		}
		in.Close()
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(part, path); err != nil {
		return err
	}
	ok = true
	return nil
}

// pruneEmptyDirs removes start and its parents while they are empty,
// stopping at dir.
func pruneEmptyDirs(start, dir string) {
	for current := start; current != dir && strings.HasPrefix(current, dir+string(filepath.Separator)); current = filepath.Dir(current) {
		if err := os.Remove(current); err != nil {
			return
		}
	}
}

// receiveArchived ends a receive session whose folder was packed as a zip,
// which the platform can open where it cannot open the folder.
func (a *app) receiveArchived(id, path string, err error) {
	l := locales.For(a.cx)
	t := a.updateTransfer(id, func(t *transfer) {
		t.Status = statusCompleted
		if err == nil && path != "" {
			t.Archive = path
			for i := range t.Files {
				t.Files[i].Path = ""
			}
		}
	})
	if t == nil {
		return
	}
	a.rememberTransfer(t)
	if err != nil {
		a.toastError(l.ArchiveFailed(), err.Error())
		return
	}
	detail := t.SavedDir
	if t.Archive != "" {
		detail = t.Archive
	}
	if detail == "" {
		detail = formatBytes(t.Total)
	}
	a.toastSuccess(l.TransferReceived(t.Peer.Alias), l.SavedTo(detail))
}

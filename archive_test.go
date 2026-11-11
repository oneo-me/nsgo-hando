package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// receivedFolderFiles writes rel's content below dir and returns a transfer
// file pointing at it.
func receivedFolderFile(t *testing.T, dir, rel, content string) transferFile {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return transferFile{ID: rel, Name: rel, Size: int64(len(content)), Done: int64(len(content)), Status: "done", Path: path}
}

// readArchive returns a zip's entries and their contents.
func readArchive(t *testing.T, path string) ([]string, map[string]string) {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	contents := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, f.Name)
		contents[f.Name] = string(data)
	}
	return names, contents
}

func TestArchiveReceivedFolders(t *testing.T) {
	dir := t.TempDir()
	files := []transferFile{
		receivedFolderFile(t, dir, "photos/a.jpg", "a"),
		receivedFolderFile(t, dir, "photos/sub/b.jpg", "bb"),
	}
	path, err := archiveReceivedFolders(dir, files)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "photos.zip"); path != want {
		t.Fatalf("archive at %q, want %q", path, want)
	}
	names, contents := readArchive(t, path)
	slices.Sort(names)
	if want := []string{"photos/a.jpg", "photos/sub/b.jpg"}; !slices.Equal(names, want) {
		t.Fatalf("archive holds %v, want %v", names, want)
	}
	if contents["photos/sub/b.jpg"] != "bb" {
		t.Fatalf("archive holds %q", contents["photos/sub/b.jpg"])
	}
	for _, f := range files {
		if _, err := os.Stat(f.Path); !os.IsNotExist(err) {
			t.Fatalf("%s is still there: %v", f.Path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "photos")); !os.IsNotExist(err) {
		t.Fatalf("the received folder is still there: %v", err)
	}
}

func TestArchiveReceivedFoldersWithoutOneRoot(t *testing.T) {
	dir := t.TempDir()
	files := []transferFile{
		receivedFolderFile(t, dir, "photos/a.jpg", "a"),
		receivedFolderFile(t, dir, "notes.txt", "notes"),
	}
	path, err := archiveReceivedFolders(dir, files)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Hando.zip"); path != want {
		t.Fatalf("archive at %q, want %q", path, want)
	}
	names, _ := readArchive(t, path)
	slices.Sort(names)
	if want := []string{"notes.txt", "photos/a.jpg"}; !slices.Equal(names, want) {
		t.Fatalf("archive holds %v, want %v", names, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); !os.IsNotExist(err) {
		t.Fatalf("the packed file is still there: %v", err)
	}
}

func TestArchiveLeavesLooseFiles(t *testing.T) {
	dir := t.TempDir()
	file := receivedFolderFile(t, dir, "notes.txt", "notes")
	path, err := archiveReceivedFolders(dir, []transferFile{file})
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Fatalf("archived a session with no folder at %q", path)
	}
	if _, err := os.Stat(file.Path); err != nil {
		t.Fatalf("the received file is gone: %v", err)
	}
}

func TestArchiveAvoidsAnExistingName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photos.zip"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []transferFile{receivedFolderFile(t, dir, "photos/a.jpg", "a")}
	path, err := archiveReceivedFolders(dir, files)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "photos (1).zip"); path != want {
		t.Fatalf("archive at %q, want %q", path, want)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "photos.zip")); err != nil || string(data) != "old" {
		t.Fatalf("the earlier archive changed: %q %v", data, err)
	}
}

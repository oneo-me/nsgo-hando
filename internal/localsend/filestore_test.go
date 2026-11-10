package localsend

import (
	"bytes"
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSafeRelativePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"photo.png", "photo.png"},
		{"Folder/Sub/photo.png", "Folder/Sub/photo.png"},
		{"../evil.txt", "_/evil.txt"},
		{"..", "_"},
		{"a/../../b", "a/_/_/b"},
		{`C:\Users\me\file.txt`, "C_/Users/me/file.txt"},
		{"  spaced  .txt  ", "spaced  .txt"},
		{"con", "_con"},
		{"a:b?.txt", "a_b_.txt"},
	}
	for _, c := range cases {
		got, err := SafeRelativePath(c.in)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPlanReceiveAvoidsExistingAndDuplicates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "same.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []File{{ID: "1", FileName: "same.txt"}, {ID: "2", FileName: "same.txt"}, {ID: "3", FileName: "folder/same.txt"}}
	targets, err := PlanReceive(dir, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"same (1).txt", "same (2).txt", "folder/same.txt"}
	for i, target := range targets {
		if target.Relative != want[i] {
			t.Errorf("target %d = %q, want %q", i, target.Relative, want[i])
		}
		if !within(dir, target.Path) {
			t.Errorf("target %d = %q leaves %q", i, target.Path, dir)
		}
	}
}

func TestWriteFileHonorsSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.bin")
	written, _, err := WriteFile(path, bytes.NewReader([]byte("1234")), 8, nil)
	if err == nil {
		t.Fatal("short file accepted")
	}
	if written != 4 {
		t.Fatalf("written = %d", written)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file kept: %v", err)
	}
}

func TestIdentityPersists(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		t.Logf("before second: %s", f.Name())
	}
	second, err := LoadIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint != second.Fingerprint {
		t.Fatalf("fingerprint changed: %q then %q", first.Fingerprint, second.Fingerprint)
	}
	if len(first.Fingerprint) != 64 || strings.ToUpper(first.Fingerprint) != first.Fingerprint {
		t.Fatalf("fingerprint %q is not 64 uppercase hex", first.Fingerprint)
	}
	if first.HTTPFingerprint != second.HTTPFingerprint || first.HTTPFingerprint == "" {
		t.Fatalf("HTTP fingerprint changed: %q then %q", first.HTTPFingerprint, second.HTTPFingerprint)
	}
}

func TestNodeStartsAndStops(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	node, err := StartNode(NodeConfig{
		Identity: id,
		Alias:    func() string { return "Test Node" },
		Port:     0,
		Addr:     "127.0.0.1:0",
		Protocol: ProtocolHTTP,
	})
	if err != nil {
		t.Fatal(err)
	}
	if node.Port() == 0 || node.Self().Alias != "Test Node" {
		t.Fatalf("node = %+v", node.Self())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	node.Stop(ctx)
}

func TestClientSendsItsCertificateDespiteCAs(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	cfg := id.ClientTLSConfig("")
	got, err := cfg.GetClientCertificate(&tls.CertificateRequestInfo{AcceptableCAs: [][]byte{{1, 2, 3}}})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got.Certificate) == 0 || !bytes.Equal(got.Certificate[0], id.Certificate.Certificate[0]) {
		t.Fatal("the client did not present its own certificate")
	}
}

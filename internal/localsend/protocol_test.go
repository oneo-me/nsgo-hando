package localsend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testFile(id, name string, content []byte) File {
	sum := sha256.Sum256(content)
	return File{ID: id, FileName: name, Size: int64(len(content)), FileType: MIMEForName(name),
		SHA256: strings.ToUpper(hex.EncodeToString(sum[:]))}
}

// startReceiver starts a server that accepts every request into dir.
func startReceiver(t *testing.T, id *Identity, protocol Protocol, dir string, pin string) (*Server, <-chan *IncomingRequest, <-chan SessionResult) {
	t.Helper()
	requests := make(chan *IncomingRequest, 4)
	results := make(chan SessionResult, 4)
	var srv *Server
	srv = NewServer(ServerConfig{
		Identity: id,
		Addr:     "127.0.0.1:0",
		Info: func() Device {
			port := 0
			if srv != nil {
				port = srv.Port()
			}
			return Device{Alias: "Receiver", Version: Version, DeviceType: DeviceDesktop, DeviceModel: "Test",
				Fingerprint: id.ClaimedFingerprint(protocol), Port: port, Protocol: protocol}
		},
		PIN:             func() string { return pin },
		VerifyChecksums: true,
		OnPrepare:       func(req *IncomingRequest) { requests <- req },
		OnSessionEnd:    func(result SessionResult) { results <- result },
	})
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})
	return srv, requests, results
}

func receiverDevice(srv *Server, id *Identity, protocol Protocol) Device {
	return Device{Alias: "Receiver", IP: "127.0.0.1", Port: srv.Port(), Protocol: protocol,
		Fingerprint: id.ClaimedFingerprint(protocol)}
}

func senderDevice(id *Identity, protocol Protocol) Device {
	return Device{Alias: "Sender", Version: Version, DeviceModel: "Test", DeviceType: DeviceDesktop,
		Fingerprint: id.ClaimedFingerprint(protocol), Port: 44444, Protocol: protocol}
}

func TestReceiveTLS(t *testing.T) {
	senderID, receiverID := mustIdentity(t), mustIdentity(t)
	dir := t.TempDir()
	srv, requests, results := startReceiver(t, receiverID, ProtocolHTTPS, dir, "")
	client := NewClient(senderID)
	defer client.Close()
	self := senderDevice(senderID, ProtocolHTTPS)
	at := receiverDevice(srv, receiverID, ProtocolHTTPS)

	content := []byte("hello over mutual TLS\n")
	file := testFile("f1", "notes/hello.txt", content)

	type reply struct {
		answer PrepareUploadResponse
		err    error
	}
	replied := make(chan reply, 1)
	go func() {
		answer, err := client.PrepareUpload(context.Background(), at, self, []File{file}, "")
		replied <- reply{answer, err}
	}()
	var req *IncomingRequest
	select {
	case req = <-requests:
	case r := <-replied:
		t.Fatalf("prepare: %v", r.err)
	case <-time.After(5 * time.Second):
		t.Fatal("prepare-upload never reached the receiver")
	}
	if req.Sender.Alias != "Sender" || len(req.Files) != 1 || req.Total != int64(len(content)) {
		t.Fatalf("request = %+v", req)
	}
	if !req.Accept(dir, nil) {
		t.Fatal("accept refused")
	}
	answer := <-replied
	if answer.err != nil {
		t.Fatalf("prepare: %v", answer.err)
	}
	token := answer.answer.Files["f1"]
	if token == "" {
		t.Fatal("no token for f1")
	}
	if err := client.Upload(context.Background(), at, answer.answer.SessionID, "f1", token,
		io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil); err != nil {
		t.Fatalf("upload: %v", err)
	}
	result := <-results
	if result.Status != StatusCompleted {
		t.Fatalf("session status %q, files %+v", result.Status, result.Files)
	}
	path := filepath.Join(dir, "notes", "hello.txt")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("saved %q", got)
	}
}

func TestReceiveHTTP(t *testing.T) {
	senderID, receiverID := mustIdentity(t), mustIdentity(t)
	dir := t.TempDir()
	srv, requests, results := startReceiver(t, receiverID, ProtocolHTTP, dir, "")
	client := NewClient(senderID)
	defer client.Close()
	self := senderDevice(senderID, ProtocolHTTP)
	at := receiverDevice(srv, receiverID, ProtocolHTTP)

	content := []byte("plain http")
	file := testFile("f1", "plain.txt", content)
	replied := make(chan PrepareUploadResponse, 1)
	failed := make(chan error, 1)
	go func() {
		answer, err := client.PrepareUpload(context.Background(), at, self, []File{file}, "")
		if err != nil {
			failed <- err
			return
		}
		replied <- answer
	}()
	req := <-requests
	if req.Sender.Fingerprint != senderID.HTTPFingerprint {
		t.Fatalf("sender fingerprint %q", req.Sender.Fingerprint)
	}
	req.Accept(dir, nil)
	select {
	case err := <-failed:
		t.Fatal(err)
	case answer := <-replied:
		if err := client.Upload(context.Background(), at, answer.SessionID, "f1", answer.Files["f1"],
			io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil); err != nil {
			t.Fatalf("upload: %v", err)
		}
	}
	result := <-results
	if result.Status != StatusCompleted {
		t.Fatalf("session status %q", result.Status)
	}
}

func TestReceivePIN(t *testing.T) {
	senderID, receiverID := mustIdentity(t), mustIdentity(t)
	srv, requests, _ := startReceiver(t, receiverID, ProtocolHTTP, t.TempDir(), "1234")
	client := NewClient(senderID)
	defer client.Close()
	self := senderDevice(senderID, ProtocolHTTP)
	at := receiverDevice(srv, receiverID, ProtocolHTTP)
	files := []File{testFile("f1", "a.txt", []byte("a"))}

	if _, err := client.PrepareUpload(context.Background(), at, self, files, ""); !errors.Is(err, ErrPINRequired) {
		t.Fatalf("without PIN: %v", err)
	}
	if _, err := client.PrepareUpload(context.Background(), at, self, files, "0000"); !errors.Is(err, ErrPINRequired) {
		t.Fatalf("wrong PIN: %v", err)
	}
	replied := make(chan error, 1)
	go func() {
		_, err := client.PrepareUpload(context.Background(), at, self, files, "1234")
		replied <- err
	}()
	req := <-requests
	if req.Sender.Alias != "Sender" {
		t.Fatalf("request sender %q", req.Sender.Alias)
	}
	req.Accept(t.TempDir(), nil)
	if err := <-replied; err != nil {
		t.Fatalf("right PIN: %v", err)
	}
}

func TestChecksumMismatch(t *testing.T) {
	senderID, receiverID := mustIdentity(t), mustIdentity(t)
	dir := t.TempDir()
	srv, requests, results := startReceiver(t, receiverID, ProtocolHTTP, dir, "")
	client := NewClient(senderID)
	defer client.Close()
	self := senderDevice(senderID, ProtocolHTTP)
	at := receiverDevice(srv, receiverID, ProtocolHTTP)
	content := []byte("actual")
	file := testFile("f1", "bad.txt", content)
	file.SHA256 = strings.Repeat("0", 64)
	replied := make(chan PrepareUploadResponse, 1)
	go func() {
		answer, err := client.PrepareUpload(context.Background(), at, self, []File{file}, "")
		if err != nil {
			t.Error(err)
			close(replied)
			return
		}
		replied <- answer
	}()
	req := <-requests
	req.Accept(dir, nil)
	answer := <-replied
	err := client.Upload(context.Background(), at, answer.SessionID, "f1", answer.Files["f1"],
		io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("upload error %v", err)
	}
	result := <-results
	if result.Status != StatusFailed {
		t.Fatalf("status %q", result.Status)
	}
	if _, err := os.Stat(filepath.Join(dir, "bad.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial file kept: %v", err)
	}
}

func TestSenderCancel(t *testing.T) {
	senderID, receiverID := mustIdentity(t), mustIdentity(t)
	srv, requests, results := startReceiver(t, receiverID, ProtocolHTTP, t.TempDir(), "")
	client := NewClient(senderID)
	defer client.Close()
	self := senderDevice(senderID, ProtocolHTTP)
	at := receiverDevice(srv, receiverID, ProtocolHTTP)
	replied := make(chan PrepareUploadResponse, 1)
	go func() {
		answer, err := client.PrepareUpload(context.Background(), at, self, []File{testFile("f1", "a.txt", []byte("a"))}, "")
		if err != nil {
			t.Error(err)
			return
		}
		replied <- answer
	}()
	req := <-requests
	req.Accept(t.TempDir(), nil)
	answer := <-replied
	if err := client.Cancel(context.Background(), at, answer.SessionID); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-results:
		if result.Status != StatusCancelled {
			t.Fatalf("status %q", result.Status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("session did not end")
	}
}

func mustIdentity(t *testing.T) *Identity {
	t.Helper()
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

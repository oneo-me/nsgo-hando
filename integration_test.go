package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oneo-me/nsgo/nsgotest"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
)

// peer is a stand-in LocalSend device on the other end of the wire.
type peer struct {
	server      *httptest.Server
	fingerprint string
	alias       string
	port        int

	mu      sync.Mutex
	files   map[string][]byte
	offered map[string]localsend.File
	prepare chan localsend.PrepareUploadRequest
}

func newPeer(t *testing.T) *peer {
	t.Helper()
	p := &peer{fingerprint: strings.Repeat("AB", 32), alias: "Phone", files: map[string][]byte{},
		offered: map[string]localsend.File{}, prepare: make(chan localsend.PrepareUploadRequest, 4)}
	mux := http.NewServeMux()
	mux.HandleFunc(localsend.APIPrefix+"/register", func(w http.ResponseWriter, r *http.Request) {
		var body localsend.Device
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeTestJSON(w, p.self())
	})
	mux.HandleFunc(localsend.APIPrefix+"/info", func(w http.ResponseWriter, r *http.Request) { writeTestJSON(w, p.self()) })
	mux.HandleFunc(localsend.APIPrefix+"/prepare-upload", func(w http.ResponseWriter, r *http.Request) {
		var body localsend.PrepareUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p.mu.Lock()
		for id, f := range body.Files {
			p.offered[id] = f
		}
		p.mu.Unlock()
		tokens := map[string]string{}
		for id := range body.Files {
			tokens[id] = "token-" + id
		}
		writeTestJSON(w, localsend.PrepareUploadResponse{SessionID: "session-from-app", Files: tokens})
	})
	mux.HandleFunc(localsend.APIPrefix+"/upload", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		id := r.URL.Query().Get("fileId")
		p.mu.Lock()
		p.files[id] = data
		p.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc(localsend.APIPrefix+"/cancel", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(p.server.URL, "http://"))
	fmt.Sscanf(port, "%d", &p.port)
	return p
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (p *peer) self() localsend.Device {
	return localsend.Device{Alias: p.alias, Version: localsend.Version, DeviceModel: "Pixel", DeviceType: localsend.DeviceMobile,
		Fingerprint: p.fingerprint, Port: p.port, Protocol: localsend.ProtocolHTTP}
}

// registerWith announces the peer to an app at port.
func (p *peer) registerWith(t *testing.T, port int) {
	t.Helper()
	data, _ := json.Marshal(p.self())
	url := fmt.Sprintf("http://127.0.0.1:%d%s/register", port, localsend.APIPrefix)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		res, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the app's receiver did not come up")
}

// waitServer waits for the app's receiver to answer.
func waitServer(t *testing.T, port int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d%s/info", port, localsend.APIPrefix)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		res, err := http.Get(url)
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the app's receiver did not come up")
}

// freePort picks a TCP port nothing listens on.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// seededStorage keeps settings the app loads at start.
func seededStorage(port int, dir string) map[string]string {
	s := settings{Alias: "Test Mac", Port: port, Protocol: localsend.ProtocolHTTP, ReceiveDir: dir,
		VerifyChecksums: true, Locale: "en"}
	data, _ := json.Marshal(s)
	return map[string]string{"settings": string(data)}
}

func TestSendMessageToPeer(t *testing.T) {
	port := freePort(t)
	dir := t.TempDir()
	p := newPeer(t)
	app := nsgotest.Start(t, Root, nsgotest.Size(1000, 720), nsgotest.Storage(seededStorage(port, dir)))
	p.registerWith(t, port)
	app.WaitFor("text:Phone", 5*time.Second)
	app.Press("id:favorite-device")
	if app.Exists("id:send-dialog") {
		t.Fatal("pressing the favorite star opened the send dialog")
	}
	if app.Exists("id:nav-settings") {
		app.WaitFor("id:sidebar-favorites", 2*time.Second)
	}

	app.Press("id:device-" + shortFingerprint(p.fingerprint))
	app.WaitFor("id:send-dialog", 2*time.Second)
	app.Type("id:send-message", "Hello from the test")
	app.Press("id:send-submit")

	deadline := time.Now().Add(5 * time.Second)
	for {
		p.mu.Lock()
		var got []byte
		for _, data := range p.files {
			got = data
		}
		p.mu.Unlock()
		if strings.Contains(string(got), "Hello from the test") {
			break
		}
		if time.Now().After(deadline) {
			p.mu.Lock()
			t.Fatalf("peer received %q from %v", got, p.offered)
			p.mu.Unlock()
		}
		time.Sleep(20 * time.Millisecond)
	}
	if app.Exists("id:nav-transfers") {
		app.Press("id:nav-transfers")
		app.WaitFor("text:Completed", 5*time.Second)
	}
}

func TestReceiveFileFromPeer(t *testing.T) {
	port := freePort(t)
	dir := t.TempDir()
	p := newPeer(t)
	app := nsgotest.Start(t, Root, nsgotest.Size(1000, 720), nsgotest.Storage(seededStorage(port, dir)))
	waitServer(t, port)

	content := []byte("sent from the phone")
	file := localsend.File{ID: "r1", FileName: "from-phone.txt", Size: int64(len(content)), FileType: "text/plain"}
	body, _ := json.Marshal(localsend.PrepareUploadRequest{Info: p.self(), Files: map[string]localsend.File{file.ID: file}})
	type answer struct {
		response localsend.PrepareUploadResponse
		err      error
	}
	answered := make(chan answer, 1)
	go func() {
		res, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d%s/prepare-upload", port, localsend.APIPrefix), "application/json", bytes.NewReader(body))
		if err != nil {
			answered <- answer{err: err}
			return
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(res.Body)
			answered <- answer{err: fmt.Errorf("status %d: %s", res.StatusCode, data)}
			return
		}
		var out localsend.PrepareUploadResponse
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			answered <- answer{err: err}
			return
		}
		answered <- answer{response: out}
	}()

	app.WaitFor("id:incoming-dialog", 5*time.Second)
	app.Press("id:incoming-accept")
	var a answer
	select {
	case a = <-answered:
	case <-time.After(5 * time.Second):
		t.Fatal("prepare-upload did not answer")
	}
	if a.err != nil {
		t.Fatal(a.err)
	}
	token := a.response.Files["r1"]
	if token == "" {
		t.Fatalf("no token: %+v", a.response)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s/upload?sessionId=%s&fileId=r1&token=%s", port, localsend.APIPrefix,
		a.response.SessionID, token)
	res, err := http.Post(url, "application/octet-stream", bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("upload answered %d", res.StatusCode)
	}
	saved := filepath.Join(dir, "from-phone.txt")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if data, err := os.ReadFile(saved); err == nil {
			if !bytes.Equal(data, content) {
				t.Fatalf("saved %q", data)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("file was not saved in %s", dir)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if app.Exists("id:nav-transfers") {
		app.Press("id:nav-transfers")
		app.WaitFor("text:Completed", 5*time.Second)
	}
}

func TestDropFilesOnPeer(t *testing.T) {
	port := freePort(t)
	p := newPeer(t)
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "dropped a.txt"), []byte("dropped file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "album"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "album", "b.txt"), []byte("inside a folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	fileURI := func(p string) string { return (&url.URL{Scheme: "file", Path: filepath.ToSlash(p)}).String() }
	app := nsgotest.Start(t, Root, nsgotest.Size(1000, 720), nsgotest.Storage(seededStorage(port, t.TempDir())))
	p.registerWith(t, port)
	app.WaitFor("text:Phone", 5*time.Second)

	app.DropFiles("id:device-"+shortFingerprint(p.fingerprint), fileURI(filepath.Join(src, "dropped a.txt")),
		fileURI(filepath.Join(src, "album")), "blob:http://localhost/1#ignored.txt")
	app.WaitFor("id:send-dialog", 2*time.Second)
	app.WaitFor("text:dropped a.txt", 2*time.Second)
	app.WaitFor("text:album/b.txt", 2*time.Second)
	app.DropFiles("id:send-form", fileURI(filepath.Join(src, "album")))
	app.WaitFor("text:album/b.txt >> nth=1", 2*time.Second)
	app.Press("id:send-submit")

	deadline := time.Now().Add(5 * time.Second)
	for {
		p.mu.Lock()
		got := map[string]string{}
		for id, data := range p.files {
			got[p.offered[id].FileName] = string(data)
		}
		n := len(p.files)
		p.mu.Unlock()
		if got["dropped a.txt"] == "dropped file" && got["album/b.txt"] == "inside a folder" && n == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("peer received %v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestDroppedPath(t *testing.T) {
	for uri, want := range map[string]string{
		"file:///tmp/a%20b.txt":        "/tmp/a b.txt",
		"file://localhost/tmp/x":       "/tmp/x",
		"file:///C:/nsgo%20dir/a.txt":  "C:/nsgo dir/a.txt",
		"file://server/share/y%20z":    "//server/share/y z",
		"blob:http://localhost/1#a.md": "",
	} {
		got, ok := droppedPath(uri)
		if want == "" {
			if ok {
				t.Errorf("droppedPath(%q) = %q, want none", uri, got)
			}
			continue
		}
		if filepath.ToSlash(got) != want {
			t.Errorf("droppedPath(%q) = %q, want %q", uri, got, want)
		}
	}
}

package localsend

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ServerConfig configures a receiver.
type ServerConfig struct {
	Identity *Identity
	// Addr is the TCP address to listen on, such as ":53317". Empty
	// listens on the port LocalSend uses.
	Addr string
	// Info returns this device as peers see it. Alias, device model and
	// type may change while the server runs.
	Info func() Device
	// PIN, when it returns a nonempty string, is required in
	// prepare-upload's pin query parameter.
	PIN func() string
	// VerifyChecksums checks the SHA-256 a sender offered.
	VerifyChecksums bool

	// OnRegister hears a device that discovered this one.
	OnRegister func(Device)
	// OnPrepare hears an upload request; it must call Accept or Decline
	// on the request, possibly from another goroutine.
	OnPrepare func(*IncomingRequest)
	// OnProgress hears bytes as files arrive.
	OnProgress func(Progress)
	// OnSessionEnd hears the session's result once.
	OnSessionEnd func(SessionResult)
	// OnError hears server failures that no request caused.
	OnError func(error)
}

// Server receives LocalSend uploads.
type Server struct {
	cfg ServerConfig

	mu      sync.Mutex
	http    *http.Server
	ln      net.Listener
	port    int
	session *receiveSession
	pending map[string]*IncomingRequest
	closed  bool
	pinFail map[string]int
}

// NewServer makes a server; Start begins listening.
func NewServer(cfg ServerConfig) *Server {
	return &Server{cfg: cfg, pending: map[string]*IncomingRequest{}, pinFail: map[string]int{}}
}

// Start begins serving. It reports where it listens through Port.
func (s *Server) Start() error {
	addr := s.cfg.Addr
	if addr == "" {
		addr = ":" + strconv.Itoa(DefaultPort)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("localsend: listening on %s: %w", addr, err)
	}
	s.mu.Lock()
	s.ln = ln
	if tcp, ok := ln.Addr().(*net.TCPAddr); ok {
		s.port = tcp.Port
	}
	s.mu.Unlock()

	if s.identity().Fingerprint != "" {
		// HTTPS peers present a client certificate; the plain listener is
		// used only for protocol http.
		if s.cfg.Info != nil && s.cfg.Info().Protocol == ProtocolHTTPS {
			ln = tls.NewListener(ln, s.identity().ServerTLSConfig())
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc(APIPrefix+"/register", s.register)
	mux.HandleFunc(APIPrefix+"/info", s.info)
	mux.HandleFunc(APIPrefix+"/prepare-upload", s.prepareUpload)
	mux.HandleFunc(APIPrefix+"/upload", s.upload)
	mux.HandleFunc(APIPrefix+"/cancel", s.cancel)
	mux.HandleFunc("/", s.hello)
	hs := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	s.mu.Lock()
	s.http = hs
	s.mu.Unlock()
	go func() {
		if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) && s.cfg.OnError != nil {
			s.cfg.OnError(err)
		}
	}()
	return nil
}

// Port is the port the server listens on, 0 before Start.
func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// Addr is the listener's address, "" before Start.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

// Stop ends the server and its session, declining pending requests.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	hs := s.http
	for _, req := range s.pending {
		req.Decline()
	}
	session := s.session
	s.mu.Unlock()
	if session != nil {
		s.endSession(session, StatusCancelled)
		session.cancel()
	}
	if hs == nil {
		return nil
	}
	if err := hs.Shutdown(ctx); err != nil {
		hs.Close()
		return err
	}
	return nil
}

// identity is the configured identity.
func (s *Server) identity() *Identity {
	if s.cfg.Identity == nil {
		return &Identity{}
	}
	return s.cfg.Identity
}

// self is this device as peers see it.
func (s *Server) self() Device {
	var d Device
	if s.cfg.Info != nil {
		d = s.cfg.Info()
	}
	if d.Alias == "" {
		d.Alias = "LocalSend"
	}
	if d.Version == "" {
		d.Version = Version
	}
	if d.Fingerprint == "" {
		protocol := d.Protocol
		if protocol == "" {
			protocol = ProtocolHTTPS
		}
		d.Fingerprint = s.identity().ClaimedFingerprint(protocol)
	}
	if d.Protocol == "" {
		d.Protocol = ProtocolHTTPS
	}
	if d.Port == 0 {
		d.Port = s.Port()
	}
	return d
}

// hello answers a browser a human-readable line.
func (s *Server) hello(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s\nLocalSend protocol %s\n", s.self().Alias, Version)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body Device
	if err := decodeJSON(w, r, &body, 1<<20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	peer := body
	peer.IP = clientIP(r)
	if peer.Protocol != ProtocolHTTP && peer.Protocol != ProtocolHTTPS {
		peer.Protocol = ProtocolHTTPS
	}
	if peer.Port < 1 || peer.Port > 65535 {
		peer.Port = DefaultPort
	}
	if fp := clientFingerprint(r); fp != "" {
		peer.Fingerprint = fp
	}
	peer.LastSeen = time.Now()
	if peer.Fingerprint != "" && s.cfg.OnRegister != nil {
		s.cfg.OnRegister(peer)
	}
	writeJSON(w, http.StatusOK, s.self().Response())
}

func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.self().Response())
}

func (s *Server) prepareUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if pin := s.pin(); pin != "" {
		if s.pinFailures(clientIP(r)) >= 10 {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			return
		}
		if r.URL.Query().Get("pin") != pin {
			s.notePinFailure(clientIP(r))
			http.Error(w, "PIN required or invalid", http.StatusUnauthorized)
			return
		}
		s.clearPinFailures(clientIP(r))
	}
	var body PrepareUploadRequest
	if err := decodeJSON(w, r, &body, 32<<20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.Files) == 0 {
		http.Error(w, "no files", http.StatusBadRequest)
		return
	}
	sender := body.Info
	sender.IP = clientIP(r)
	if fp := clientFingerprint(r); fp != "" {
		sender.Fingerprint = fp
	}
	if sender.Fingerprint == "" {
		http.Error(w, "missing fingerprint", http.StatusBadRequest)
		return
	}
	if sender.Port < 1 || sender.Port > 65535 {
		sender.Port = DefaultPort
	}
	if sender.Protocol != ProtocolHTTP && sender.Protocol != ProtocolHTTPS {
		sender.Protocol = ProtocolHTTPS
	}
	if sender.Alias == "" {
		sender.Alias = "Unknown device"
	}
	sender.LastSeen = time.Now()
	files := SortedFiles(body.Files)
	req := &IncomingRequest{SessionID: NewID(16), Sender: sender, Files: files, Total: TotalSize(files), decide: make(chan decision, 1)}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		http.Error(w, "server closed", http.StatusInternalServerError)
		return
	}
	if s.session != nil || len(s.pending) > 0 {
		s.mu.Unlock()
		http.Error(w, "blocked by another session", http.StatusConflict)
		return
	}
	s.pending[req.SessionID] = req
	s.mu.Unlock()
	if s.cfg.OnPrepare == nil {
		req.Decline()
	} else {
		s.cfg.OnPrepare(req)
	}

	var chosen decision
	select {
	case chosen = <-req.decide:
	case <-r.Context().Done():
		s.dropPending(req.SessionID)
		http.Error(w, "cancelled", http.StatusForbidden)
		return
	case <-time.After(10 * time.Minute):
		s.dropPending(req.SessionID)
		http.Error(w, "timed out", http.StatusForbidden)
		return
	}
	s.dropPending(req.SessionID)
	if !chosen.accept {
		http.Error(w, "rejected", http.StatusForbidden)
		return
	}
	accepted := make([]File, 0, len(files))
	for _, f := range files {
		if chosen.files[f.ID] {
			accepted = append(accepted, f)
		}
	}
	if len(accepted) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	targets, err := PlanReceive(chosen.dir, accepted, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session := newReceiveSession(req.SessionID, sender, accepted, targets)
	s.mu.Lock()
	if s.session != nil || s.closed {
		s.mu.Unlock()
		http.Error(w, "blocked by another session", http.StatusConflict)
		return
	}
	s.session = session
	s.mu.Unlock()
	tokens := make(map[string]string, len(accepted))
	for _, f := range accepted {
		tokens[f.ID] = session.files[f.ID].token
	}
	writeJSON(w, http.StatusOK, PrepareUploadResponse{SessionID: session.id, Files: tokens})
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	sessionID, fileID, token := q.Get("sessionId"), q.Get("fileId"), q.Get("token")
	if sessionID == "" || fileID == "" || token == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if session == nil || session.id != sessionID || !sameIP(session.sender.IP, clientIP(r)) {
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}
	file, ok := session.begin(fileID, token)
	if !ok {
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}
	if file.dto.Size > 0 && r.ContentLength >= 0 && r.ContentLength != file.dto.Size {
		session.failed(file, "size mismatch")
		http.Error(w, "size mismatch", http.StatusBadRequest)
		s.maybeFinish(session)
		return
	}
	var body io.ReadCloser = r.Body
	if file.dto.Size >= 0 {
		body = http.MaxBytesReader(w, body, file.dto.Size+1)
	}
	body = &contextReader{ctx: session.ctx, r: body}
	written, hash, err := WriteFile(file.target.Path, body, file.dto.Size, func(n int64) {
		session.progress(file, n)
		if s.cfg.OnProgress != nil {
			s.cfg.OnProgress(Progress{SessionID: session.id, SenderAlias: session.sender.Alias,
				FileID: file.dto.ID, FileName: file.dto.FileName, Received: n, Size: file.dto.Size})
		}
	})
	if err != nil {
		session.failed(file, err.Error())
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrPermission) {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		s.maybeFinish(session)
		return
	}
	if s.cfg.VerifyChecksums && file.dto.SHA256 != "" && !strings.EqualFold(file.dto.SHA256, hash) {
		_ = os.Remove(file.target.Path)
		session.failed(file, "checksum mismatch")
		http.Error(w, "checksum mismatch", http.StatusUnprocessableEntity)
		s.maybeFinish(session)
		return
	}
	_ = applyModified(file.target.Path, file.dto.Metadata)
	session.done(file, written)
	if s.cfg.OnProgress != nil {
		s.cfg.OnProgress(Progress{SessionID: session.id, SenderAlias: session.sender.Alias,
			FileID: file.dto.ID, FileName: file.dto.FileName, Received: written, Size: file.dto.Size, Done: true})
	}
	s.maybeFinish(session)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := r.URL.Query().Get("sessionId")
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if session != nil && session.id == sessionID && sameIP(session.sender.IP, clientIP(r)) {
		s.endSession(session, StatusCancelled)
		session.cancel()
	}
	w.WriteHeader(http.StatusOK)
}

// dropPending forgets a request by session.
func (s *Server) dropPending(sessionID string) {
	s.mu.Lock()
	delete(s.pending, sessionID)
	s.mu.Unlock()
}

// endSession removes session once and reports its result.
func (s *Server) endSession(session *receiveSession, status string) {
	s.mu.Lock()
	if s.session == session {
		s.session = nil
	}
	s.mu.Unlock()
	if !session.markEnded() {
		return
	}
	if s.cfg.OnSessionEnd != nil {
		s.cfg.OnSessionEnd(session.result(status))
	}
}

// maybeFinish ends session when every accepted file reached an end.
func (s *Server) maybeFinish(session *receiveSession) {
	if session.remaining() > 0 {
		return
	}
	s.endSession(session, session.outcome())
}

// pin is the configured PIN, "" for none.
func (s *Server) pin() string {
	if s.cfg.PIN == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.PIN())
}

// notePinFailure counts a wrong PIN.
func (s *Server) notePinFailure(ip string) {
	s.mu.Lock()
	s.pinFail[ip]++
	s.mu.Unlock()
}

func (s *Server) pinFailures(ip string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pinFail[ip]
}

func (s *Server) clearPinFailures(ip string) {
	s.mu.Lock()
	delete(s.pinFail, ip)
	s.mu.Unlock()
}

// Progress is one file's incoming bytes.
type Progress struct {
	SessionID   string
	SenderAlias string
	FileID      string
	FileName    string
	Received    int64
	Size        int64
	Done        bool
}

// ReceivedFile is one file of a finished session.
type ReceivedFile struct {
	File
	Path   string
	Status string
	Error  string
}

// SessionResult is a receive session's result.
type SessionResult struct {
	SessionID string
	Sender    Device
	Status    string
	Files     []ReceivedFile
	Total     int64
	Started   time.Time
	Ended     time.Time
}

// Session statuses.
const (
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
	StatusFailed    = "failed"
)

// IncomingRequest is a sender's prepare-upload waiting for the app's
// answer. Call Accept or Decline once.
type IncomingRequest struct {
	SessionID string
	Sender    Device
	Files     []File
	Total     int64

	once   sync.Once
	decide chan decision
}

type decision struct {
	accept bool
	dir    string
	files  map[string]bool
}

// Accept saves the chosen files (all when ids is empty) under dir and
// answers the sender. It reports false when the request was answered
// already.
func (r *IncomingRequest) Accept(dir string, ids []string) bool {
	if dir == "" {
		return false
	}
	chosen := map[string]bool{}
	for _, f := range r.Files {
		chosen[f.ID] = true
	}
	if len(ids) > 0 {
		chosen = map[string]bool{}
		for _, id := range ids {
			chosen[id] = true
		}
	}
	answered := false
	r.once.Do(func() {
		r.decide <- decision{accept: true, dir: dir, files: chosen}
		answered = true
	})
	return answered
}

// Decline rejects the request. It reports false when it was answered
// already.
func (r *IncomingRequest) Decline() bool {
	answered := false
	r.once.Do(func() {
		r.decide <- decision{accept: false}
		answered = true
	})
	return answered
}

// receiveSession is one accepted upload session.
type receiveSession struct {
	id      string
	sender  Device
	files   map[string]*receiveFile
	order   []string
	dir     string
	started time.Time

	mu      sync.Mutex
	ended   bool
	cancel_ context.CancelFunc
	ctx     context.Context
}

// receiveFile is one accepted file's state.
type receiveFile struct {
	dto    File
	token  string
	target ReceiveTarget
	status string
	got    int64
	err    string
}

func newReceiveSession(id string, sender Device, files []File, targets []ReceiveTarget) *receiveSession {
	ctx, cancel := context.WithCancel(context.Background())
	s := &receiveSession{id: id, sender: sender, files: map[string]*receiveFile{}, started: time.Now(),
		ctx: ctx, cancel_: cancel}
	for i, f := range files {
		s.files[f.ID] = &receiveFile{dto: f, token: NewID(16), target: targets[i], status: "pending"}
		s.order = append(s.order, f.ID)
	}
	return s
}

func (s *receiveSession) cancel() { s.cancel_() }

// begin marks a file uploading and returns it.
func (s *receiveSession) begin(fileID, token string) (*receiveFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return nil, false
	}
	f := s.files[fileID]
	if f == nil || f.token != token || f.status != "pending" {
		return nil, false
	}
	f.status = "uploading"
	return f, true
}

func (s *receiveSession) progress(f *receiveFile, n int64) {
	s.mu.Lock()
	f.got = n
	s.mu.Unlock()
}

func (s *receiveSession) done(f *receiveFile, n int64) {
	s.mu.Lock()
	f.got, f.status = n, "done"
	s.mu.Unlock()
}

func (s *receiveSession) failed(f *receiveFile, reason string) {
	s.mu.Lock()
	f.status, f.err = "failed", reason
	s.mu.Unlock()
}

func (s *receiveSession) remaining() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, id := range s.order {
		if f := s.files[id]; f != nil && (f.status == "pending" || f.status == "uploading") {
			n++
		}
	}
	return n
}

func (s *receiveSession) outcome() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	failed := false
	for _, f := range s.files {
		if f.status != "done" {
			failed = true
		}
	}
	if failed {
		return StatusFailed
	}
	return StatusCompleted
}

func (s *receiveSession) markEnded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return false
	}
	s.ended = true
	return true
}

func (s *receiveSession) result(status string) SessionResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := SessionResult{SessionID: s.id, Sender: s.sender, Status: status, Started: s.started, Ended: time.Now()}
	for _, id := range s.order {
		f := s.files[id]
		if f == nil {
			continue
		}
		path := ""
		if f.status == "done" {
			path = f.target.Path
		}
		result.Files = append(result.Files, ReceivedFile{File: f.dto, Path: path, Status: f.status, Error: f.err})
		result.Total += f.got
	}
	return result
}

// contextReader stops a body read once its session is cancelled.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(b []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(b)
}

func (c *contextReader) Close() error {
	if closer, ok := c.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// decodeJSON reads a JSON request body no larger than limit.
func decodeJSON(w http.ResponseWriter, r *http.Request, into any, limit int64) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	body := http.MaxBytesReader(w, r.Body, limit)
	if err := json.NewDecoder(body).Decode(into); err != nil {
		return fmt.Errorf("invalid body: %w", err)
	}
	return nil
}

// writeJSON answers status with v as JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// clientIP is the request's peer address without its port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// clientFingerprint is the SHA-256 of the client certificate TLS
// verified, "" on a plain connection.
func clientFingerprint(r *http.Request) string {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return ""
	}
	sum := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// sameIP compares addresses as hosts, including IPv4-mapped IPv6.
func sameIP(a, b string) bool {
	ipA, ipB := net.ParseIP(a), net.ParseIP(b)
	if ipA != nil && ipB != nil {
		return ipA.Equal(ipB)
	}
	return a == b
}

// applyModified sets a received file's modification time when the sender
// gave one.
func applyModified(path string, metadata *Metadata) error {
	if metadata == nil || metadata.Modified == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, metadata.Modified)
	if err != nil {
		return err
	}
	atime := t
	if metadata.Accessed != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, metadata.Accessed); err == nil {
			atime = parsed
		}
	}
	return os.Chtimes(path, atime, t)
}

// CancelSession ends the receive session sessionID, as the app's cancel
// button does, and reports whether it was the active one.
func (s *Server) CancelSession(sessionID string) bool {
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if session == nil || session.id != sessionID {
		return false
	}
	s.endSession(session, StatusCancelled)
	session.cancel()
	return true
}

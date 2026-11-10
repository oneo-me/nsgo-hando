package localsend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrPINRequired is prepare-upload's 401: the receiver asks for a PIN.
	ErrPINRequired = errors.New("localsend: a PIN is required")
	// ErrRejected is prepare-upload's 403: the receiver declined.
	ErrRejected = errors.New("localsend: transfer rejected")
	// ErrBusy is prepare-upload's 409: the receiver is in another session.
	ErrBusy = errors.New("localsend: receiver has another session")
	// ErrNoFiles is prepare-upload's 204: the receiver needs none of the files.
	ErrNoFiles = errors.New("localsend: receiver needs none of the files")
	// ErrChecksum is upload's 422: the received file did not match its hash.
	ErrChecksum = errors.New("localsend: checksum mismatch")
	// ErrInvalidToken is upload's 403: the file or its token is not valid.
	ErrInvalidToken = errors.New("localsend: invalid upload token")
)

// Error is a peer's HTTP answer other than success.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("localsend: peer answered %d", e.StatusCode)
	}
	return fmt.Sprintf("localsend: peer answered %d: %s", e.StatusCode, e.Message)
}

// Status reports whether err is a peer's answer with code.
func Status(err error, code int) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == code
}

// Client talks to LocalSend peers as the device id identifies itself.
type Client struct {
	id     *Identity
	plain  *http.Client
	mu     sync.Mutex
	secure map[string]*http.Client
}

// NewClient makes a client.
func NewClient(id *Identity) *Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	c := &Client{id: id, secure: map[string]*http.Client{}}
	c.plain = &http.Client{Transport: &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    true,
	}}
	return c
}

// Close ends idle connections.
func (c *Client) Close() {
	c.plain.CloseIdleConnections()
	c.mu.Lock()
	for _, hc := range c.secure {
		hc.CloseIdleConnections()
	}
	c.mu.Unlock()
}

// HTTP is the client for at: HTTPS pinned to its fingerprint when at is
// HTTPS, and the plain client otherwise. An empty fingerprint is only for
// discovery, which learns the peer's fingerprint from its certificate.
func (c *Client) HTTP(at Device) *http.Client {
	if at.Protocol != ProtocolHTTPS {
		return c.plain
	}
	key := strings.ToUpper(strings.TrimSpace(at.Fingerprint))
	c.mu.Lock()
	defer c.mu.Unlock()
	if hc := c.secure[key]; hc != nil {
		return hc
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	hc := &http.Client{Transport: &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       c.id.ClientTLSConfig(key),
		ForceAttemptHTTP2:     false,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    true,
	}}
	c.secure[key] = hc
	return hc
}

// Register announces self to at and returns what answered, with the
// fingerprint of the certificate it answered with over HTTPS.
func (c *Client) Register(ctx context.Context, at, self Device) (Device, error) {
	if at.Port == 0 {
		at.Port = DefaultPort
	}
	body, err := json.Marshal(self)
	if err != nil {
		return Device{}, err
	}
	res, err := c.request(ctx, at, http.MethodPost, "/register", nil, bytes.NewReader(body), "application/json")
	if err != nil {
		return Device{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Device{}, responseError(res)
	}
	var peer Device
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&peer); err != nil {
		return Device{}, fmt.Errorf("localsend: reading register answer: %w", err)
	}
	if at.Protocol == ProtocolHTTPS {
		if tlsState := res.TLS; tlsState != nil && len(tlsState.PeerCertificates) > 0 {
			peer.Fingerprint = FingerprintOf(tlsState.PeerCertificates[0].Raw)
		}
	}
	peer.IP, peer.Port, peer.Protocol = at.IP, at.Port, at.Protocol
	peer.LastSeen = time.Now()
	return peer, nil
}

// PrepareUpload asks at to accept files and returns the session, or one
// of the errors above.
func (c *Client) PrepareUpload(ctx context.Context, at, self Device, files []File, pin string) (PrepareUploadResponse, error) {
	if at.Port == 0 {
		at.Port = DefaultPort
	}
	byID := make(map[string]File, len(files))
	for _, f := range files {
		byID[f.ID] = f
	}
	body, err := json.Marshal(PrepareUploadRequest{Info: self, Files: byID})
	if err != nil {
		return PrepareUploadResponse{}, err
	}
	var query url.Values
	if pin != "" {
		query = url.Values{"pin": {pin}}
	}
	res, err := c.request(ctx, at, http.MethodPost, "/prepare-upload", query, bytes.NewReader(body), "application/json")
	if err != nil {
		return PrepareUploadResponse{}, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNoContent:
		return PrepareUploadResponse{}, ErrNoFiles
	case http.StatusUnauthorized:
		return PrepareUploadResponse{}, ErrPINRequired
	case http.StatusForbidden:
		return PrepareUploadResponse{}, ErrRejected
	case http.StatusConflict:
		return PrepareUploadResponse{}, ErrBusy
	case http.StatusTooManyRequests:
		return PrepareUploadResponse{}, &Error{StatusCode: res.StatusCode, Message: "too many requests"}
	default:
		return PrepareUploadResponse{}, responseError(res)
	}
	var answer PrepareUploadResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&answer); err != nil {
		return PrepareUploadResponse{}, fmt.Errorf("localsend: reading prepare-upload answer: %w", err)
	}
	if answer.SessionID == "" {
		return PrepareUploadResponse{}, errors.New("localsend: prepare-upload answer has no session")
	}
	return answer, nil
}

// Upload sends one accepted file's content. The reader is closed when
// done. progress receives the bytes sent so far, as the HTTP client
// consumes the body.
func (c *Client) Upload(ctx context.Context, at Device, sessionID, fileID, token string, content io.ReadCloser, size int64, progress func(sent int64)) error {
	if at.Port == 0 {
		at.Port = DefaultPort
	}
	defer content.Close()
	query := url.Values{"sessionId": {sessionID}, "fileId": {fileID}, "token": {token}}
	var body io.Reader = content
	if progress != nil {
		body = &progressReader{r: content, total: size, report: progress}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, at.URLBase()+APIPrefix+"/upload?"+query.Encode(), body)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := c.HTTP(at).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusForbidden:
		return ErrInvalidToken
	case http.StatusUnprocessableEntity:
		return ErrChecksum
	default:
		return responseError(res)
	}
}

// Cancel tells at to end the session.
func (c *Client) Cancel(ctx context.Context, at Device, sessionID string) error {
	if at.Port == 0 {
		at.Port = DefaultPort
	}
	query := url.Values{"sessionId": {sessionID}}
	res, err := c.request(ctx, at, http.MethodPost, "/cancel", query, nil, "")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return responseError(res)
	}
	return nil
}

// request sends one API request to at.
func (c *Client) request(ctx context.Context, at Device, method, route string, query url.Values, body io.Reader, contentType string) (*http.Response, error) {
	if at.Port == 0 {
		at.Port = DefaultPort
	}
	addr := at.URLBase() + APIPrefix + route
	if len(query) > 0 {
		addr += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, addr, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", "NSGo-LocalSend/"+Version)
	return c.HTTP(at).Do(req)
}

// responseError is res as an Error carrying a short body.
func responseError(res *http.Response) error {
	message, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
	return &Error{StatusCode: res.StatusCode, Message: strings.TrimSpace(string(message))}
}

// progressReader reports how much of a body an HTTP client read.
type progressReader struct {
	r      io.Reader
	sent   int64
	total  int64
	report func(int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.sent += int64(n)
		if p.sent > p.total && p.total > 0 {
			p.sent = p.total
		}
		p.report(p.sent)
	}
	return n, err
}

// ParsePort reads a port, or DefaultPort when s names none or an
// invalid one.
func ParsePort(s string) int {
	port, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || port < 1 || port > 65535 {
		return DefaultPort
	}
	return port
}

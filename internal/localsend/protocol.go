// Package localsend is the LocalSend protocol: HTTP discovery and file
// transfer, with the HTTPS identity LocalSend peers use.
//
// It implements protocol v2.2 of https://github.com/localsend/protocol,
// the version current LocalSend apps speak: mutual TLS where each device
// presents a self-signed certificate and peers are identified by the
// certificate's SHA-256 fingerprint, HTTP where encryption is off, and
// IPv4 multicast discovery with the HTTP register fallback.
package localsend

import (
	"crypto/rand"
	"encoding/hex"
	"mime"
	"net"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Version is the protocol version announced to peers.
	Version = "2.2"
	// DefaultPort is LocalSend's HTTP and multicast port.
	DefaultPort = 53317
	// DefaultMulticastGroup is LocalSend's IPv4 multicast group.
	DefaultMulticastGroup = "224.0.0.167"
	// APIPrefix is the path every protocol v2 route starts with.
	APIPrefix = "/api/localsend/v2"
)

// Protocol is how a device serves its API.
type Protocol string

const (
	ProtocolHTTP  Protocol = "http"
	ProtocolHTTPS Protocol = "https"
)

// DeviceType is what a device is, for choosing an icon.
type DeviceType string

const (
	DeviceMobile   DeviceType = "mobile"
	DeviceDesktop  DeviceType = "desktop"
	DeviceWeb      DeviceType = "web"
	DeviceHeadless DeviceType = "headless"
	DeviceServer   DeviceType = "server"
)

// Device is a LocalSend peer, as discovery and register exchange it. IP
// and LastSeen are this app's bookkeeping and never go on the wire.
type Device struct {
	Alias       string     `json:"alias"`
	Version     string     `json:"version,omitempty"`
	DeviceModel string     `json:"deviceModel,omitempty"`
	DeviceType  DeviceType `json:"deviceType,omitempty"`
	Fingerprint string     `json:"fingerprint"`
	Port        int        `json:"port,omitempty"`
	Protocol    Protocol   `json:"protocol,omitempty"`
	Download    bool       `json:"download,omitempty"`

	IP       string    `json:"-"`
	LastSeen time.Time `json:"-"`
	// Favorite is this app's pin on the device, never on the wire.
	Favorite bool `json:"favorite,omitempty"`
}

// Response is d as register and info answer it: without where to reach
// the device, which the request already knows.
func (d Device) Response() Device {
	d.Port, d.Protocol, d.IP, d.LastSeen = 0, "", "", time.Time{}
	return d
}

// Addr is where the device's API is, host:port.
func (d Device) Addr() string {
	port := d.Port
	if port == 0 {
		port = DefaultPort
	}
	return net.JoinHostPort(d.IP, strconv.Itoa(port))
}

// URLBase is the device's API root, such as https://192.168.1.4:53317.
func (d Device) URLBase() string {
	scheme := string(d.Protocol)
	if scheme == "" {
		scheme = string(ProtocolHTTP)
	}
	return scheme + "://" + d.Addr()
}

// Metadata is a file's timestamps, as RFC 3339 strings.
type Metadata struct {
	Modified string `json:"modified,omitempty"`
	Accessed string `json:"accessed,omitempty"`
}

// File is a file offered or received in a session.
type File struct {
	ID       string    `json:"id"`
	FileName string    `json:"fileName"`
	Size     int64     `json:"size"`
	FileType string    `json:"fileType"`
	SHA256   string    `json:"sha256,omitempty"`
	Preview  string    `json:"preview,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

// PrepareUploadRequest asks a receiver to accept files.
type PrepareUploadRequest struct {
	Info  Device          `json:"info"`
	Files map[string]File `json:"files"`
}

// PrepareUploadResponse is a receiver's decision: the session's ID and,
// for each accepted file, the token its upload needs.
type PrepareUploadResponse struct {
	SessionID string            `json:"sessionId"`
	Files     map[string]string `json:"files"`
}

// RegisterRequest is a register request body; it is a Device without the
// fields a request does not carry.
type RegisterRequest Device

// SortedFiles returns files by the order their names put them in, so a
// map from the wire shows in a stable order.
func SortedFiles(files map[string]File) []File {
	out := make([]File, 0, len(files))
	for _, f := range files {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FileName != out[j].FileName {
			return out[i].FileName < out[j].FileName
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// TotalSize adds the files' sizes.
func TotalSize(files []File) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

// NewID returns a random lowercase hex identifier.
func NewID(bytes int) string {
	if bytes <= 0 {
		bytes = 16
	}
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand never fails on the platforms LocalSend runs on.
		panic(err)
	}
	return hex.EncodeToString(b)
}

// MIMEForName guesses a file's media type from its name, as LocalSend
// shows its icon.
func MIMEForName(name string) string {
	if t := mime.TypeByExtension(strings.ToLower(path.Ext(name))); t != "" {
		if i := strings.IndexByte(t, ';'); i >= 0 {
			t = strings.TrimSpace(t[:i])
		}
		return t
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".md":
		return "text/markdown"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".toml":
		return "text/toml"
	case ".apk":
		return "application/vnd.android.package-archive"
	}
	return "application/octet-stream"
}

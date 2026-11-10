package localsend

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Identity is this device's LocalSend identity: the self-signed
// certificate and key HTTPS peers present, its fingerprint, and the
// random token HTTP discovery uses instead.
type Identity struct {
	Certificate tls.Certificate
	CertPEM     []byte
	KeyPEM      []byte
	// Fingerprint is the certificate's SHA-256 as uppercase hex, the
	// identity current LocalSend uses over HTTPS.
	Fingerprint string
	// HTTPFingerprint identifies the device when HTTPS is off.
	HTTPFingerprint string
}

const (
	certFile  = "identity.crt"
	keyFile   = "identity.key"
	tokenFile = "identity.http"
)

// LoadIdentity reads the identity in dir, making one on first use. A
// damaged certificate or key is replaced, as a peer would treat it as
// another device anyway.
func LoadIdentity(dir string) (*Identity, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("localsend: making identity directory: %w", err)
	}
	id := &Identity{}
	certPEM, certErr := os.ReadFile(filepath.Join(dir, certFile))
	keyPEM, keyErr := os.ReadFile(filepath.Join(dir, keyFile))
	if certErr == nil && keyErr == nil {
		pair, err := tls.X509KeyPair(certPEM, keyPEM)
		if err == nil && len(pair.Certificate) > 0 {
			id.Certificate, id.CertPEM, id.KeyPEM = pair, certPEM, keyPEM
			id.Fingerprint = FingerprintOf(pair.Certificate[0])
		}
	}
	if id.Fingerprint == "" {
		next, err := NewIdentity()
		if err != nil {
			return nil, err
		}
		*id = *next
		if err := os.WriteFile(filepath.Join(dir, certFile), id.CertPEM, 0o600); err != nil {
			return nil, fmt.Errorf("localsend: saving certificate: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, keyFile), id.KeyPEM, 0o600); err != nil {
			return nil, fmt.Errorf("localsend: saving key: %w", err)
		}
	}
	token, err := os.ReadFile(filepath.Join(dir, tokenFile))
	if saved := strings.TrimSpace(string(token)); err == nil && saved != "" {
		id.HTTPFingerprint = saved
	} else {
		if id.HTTPFingerprint == "" {
			id.HTTPFingerprint = NewID(32)
		}
		if err := os.WriteFile(filepath.Join(dir, tokenFile), []byte(id.HTTPFingerprint), 0o600); err != nil {
			return nil, fmt.Errorf("localsend: saving HTTP fingerprint: %w", err)
		}
	}
	return id, nil
}

// NewIdentity makes a fresh self-signed identity, as LocalSend's Dart
// app does: RSA-2048 with no subject name peers can rely on, since the
// certificate's fingerprint is the identity.
func NewIdentity() (*Identity, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("localsend: making key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("localsend: making serial number: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "LocalSend User"},
		NotBefore:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2126, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("localsend: making certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("localsend: encoding key: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("localsend: loading key pair: %w", err)
	}
	return &Identity{Certificate: pair, CertPEM: certPEM, KeyPEM: keyPEM,
		Fingerprint: FingerprintOf(der), HTTPFingerprint: NewID(32)}, nil
}

// FingerprintOf is a certificate's SHA-256 in DER, as uppercase hex.
func FingerprintOf(der []byte) string {
	sum := sha256.Sum256(der)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// ClaimedFingerprint is the fingerprint the device puts in its
// announcements and register requests: the certificate's over HTTPS, a
// random token over HTTP.
func (id *Identity) ClaimedFingerprint(protocol Protocol) string {
	if protocol == ProtocolHTTPS {
		return id.Fingerprint
	}
	return id.HTTPFingerprint
}

// ServerTLSConfig serves HTTPS with the identity's certificate and asks
// every client for one too, as LocalSend's server does.
func (id *Identity) ServerTLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{id.Certificate},
		ClientAuth:   tls.RequireAnyClientCert,
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 {
				return errors.New("localsend: client presented no certificate")
			}
			return verifyPeerCertificate(raw[0], "")
		},
	}
}

// ClientTLSConfig calls HTTPS peers with the identity's certificate.
// Discovery accepts any valid self-signed peer; every request after it
// passes the peer's fingerprint, so nothing goes to another device
// answering on that address.
func (id *Identity) ClientTLSConfig(pinnedFingerprint string) *tls.Config {
	return &tls.Config{
		Certificates:       []tls.Certificate{id.Certificate},
		InsecureSkipVerify: true, // identity is the fingerprint, not a chain
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"http/1.1"},
		// LocalSend peers ask for a client certificate naming their own
		// acceptable CAs; a self-signed identity matches none, and Go
		// would send none. Send ours whatever they name.
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &id.Certificate, nil
		},
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 {
				return errors.New("localsend: peer presented no certificate")
			}
			return verifyPeerCertificate(raw[0], pinnedFingerprint)
		},
	}
}

// verifyPeerCertificate checks a LocalSend peer's certificate: its time
// validity, and either the fingerprint the caller pinned or that it
// signs itself.
func verifyPeerCertificate(der []byte, pinned string) error {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return fmt.Errorf("localsend: bad peer certificate: %w", err)
	}
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("localsend: peer certificate is valid %s to %s", cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339))
	}
	if pinned != "" {
		if !strings.EqualFold(FingerprintOf(der), strings.TrimSpace(pinned)) {
			return fmt.Errorf("localsend: peer certificate fingerprint %s does not match %s", FingerprintOf(der), pinned)
		}
		return nil
	}
	if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
		return fmt.Errorf("localsend: peer certificate is not self-signed: %w", err)
	}
	return nil
}

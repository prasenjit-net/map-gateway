package tlsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prasenjit-net/mcp-gateway/tlsutil"
)

func TestGenerateSelfSigned(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")

	if err := tlsutil.GenerateSelfSigned(certPath, keyPath); err != nil {
		t.Fatalf("GenerateSelfSigned: %v", err)
	}

	// Both files must exist.
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("cert file missing: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("key file missing: %v", err)
	}
}

func TestGenerateSelfSignedIdempotent(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")

	// First call — creates files.
	if err := tlsutil.GenerateSelfSigned(certPath, keyPath); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — files already exist, should be a no-op (no error).
	if err := tlsutil.GenerateSelfSigned(certPath, keyPath); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestLoadTLSCertificate(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")

	if err := tlsutil.GenerateSelfSigned(certPath, keyPath); err != nil {
		t.Fatalf("GenerateSelfSigned: %v", err)
	}

	cert, err := tlsutil.LoadTLSCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadTLSCertificate: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Error("loaded certificate has no DER blocks")
	}
}

func TestLoadTLSCertificateNotFound(t *testing.T) {
	_, err := tlsutil.LoadTLSCertificate("/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Error("expected error loading nonexistent cert, got nil")
	}
}

func TestLoadCA(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	if err := tlsutil.GenerateSelfSigned(certPath, keyPath); err != nil {
		t.Fatalf("GenerateSelfSigned: %v", err)
	}

	pool, err := tlsutil.LoadCA(certPath)
	if err != nil {
		t.Fatalf("LoadCA: %v", err)
	}
	if pool == nil {
		t.Error("LoadCA returned nil pool")
	}
}

func TestLoadCANotFound(t *testing.T) {
	_, err := tlsutil.LoadCA("/nonexistent/ca.pem")
	if err == nil {
		t.Error("expected error loading nonexistent CA, got nil")
	}
}

func TestNewMTLSClientConfig(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")

	if err := tlsutil.GenerateSelfSigned(certPath, keyPath); err != nil {
		t.Fatalf("GenerateSelfSigned: %v", err)
	}

	cfg, err := tlsutil.NewMTLSClientConfig(certPath, keyPath, certPath)
	if err != nil {
		t.Fatalf("NewMTLSClientConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("returned nil tls.Config")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("want 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("RootCAs should not be nil")
	}
}

package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestMapCurveNamesToIDs(t *testing.T) {
	tests := []struct {
		name       string
		input      []string
		want       []tls.CurveID
		errContain string
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name: "all supported names (IANA and Go constant)",
			input: []string{
				// X25519 and the ML-KEM hybrids: IANA name == Go constant name.
				"X25519",
				"X25519MLKEM768",
				"SecP256r1MLKEM768",
				"SecP384r1MLKEM1024",
				// Classic EC curves: IANA name and Go constant name.
				"secp256r1", "CurveP256",
				"secp384r1", "CurveP384",
				"secp521r1", "CurveP521",
			},
			want: []tls.CurveID{
				tls.X25519,
				tls.X25519MLKEM768,
				tls.SecP256r1MLKEM768,
				tls.SecP384r1MLKEM1024,
				tls.CurveP256, tls.CurveP256,
				tls.CurveP384, tls.CurveP384,
				tls.CurveP521, tls.CurveP521,
			},
		},
		{
			name:  "mixed IANA and Go constant names",
			input: []string{"secp256r1", "CurveP384"},
			want:  []tls.CurveID{tls.CurveP256, tls.CurveP384},
		},
		{
			name:       "unknown curve name",
			input:      []string{"CurveUnknown"},
			errContain: "unknown TLSCurve: CurveUnknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mapCurveNamesToIDs(tc.input)
			if tc.errContain != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContain)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestNewServerConfigCurvePreferences(t *testing.T) {
	certPath, keyPath := newSelfSignedCert(t)

	l := log.NewNopLogger()

	tests := []struct {
		name        string
		curves      []string
		want        []tls.CurveID
		errContains []string
	}{
		{
			name:   "omitted uses default curves",
			curves: nil,
			want:   nil,
		},
		{
			name: "configured curve preferences",
			curves: []string{
				"X25519MLKEM768",
				"X25519",
				"secp256r1",
				"CurveP384",
				"secp521r1",
				"SecP256r1MLKEM768",
				"SecP384r1MLKEM1024",
			},
			want: []tls.CurveID{
				tls.X25519MLKEM768,
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
				tls.SecP256r1MLKEM768,
				tls.SecP384r1MLKEM1024,
			},
		},
		{
			name:   "invalid curve preferences",
			curves: []string{"CurveUnknown"},
			errContains: []string{
				"TLS curve preference name to ID conversion",
				"unknown TLSCurve: CurveUnknown",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewServerConfig(l, certPath, keyPath, "VersionTLS13", nil, tc.curves)
			if len(tc.errContains) > 0 {
				require.Error(t, err)
				for _, msg := range tc.errContains {
					require.ErrorContains(t, err, msg)
				}
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.CurvePreferences)
		})
	}
}

// newSelfSignedCert writes a self-signed cert/key pair to temp files and
// returns their paths.
func newSelfSignedCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600))

	return certPath, keyPath
}

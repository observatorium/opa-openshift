package config

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"k8s.io/component-base/cli/flag"
)

var errUnknownTLSCurve = errors.New("unknown TLSCurve")

// curveIDs maps supported key-exchange group names to crypto/tls CurveIDs.
// Both IANA names and Go crypto/tls constant names are accepted.
var curveIDs = map[string]tls.CurveID{
	// X25519 and the ML-KEM hybrids: IANA name == Go constant name.
	"X25519":             tls.X25519,
	"X25519MLKEM768":     tls.X25519MLKEM768,
	"SecP256r1MLKEM768":  tls.SecP256r1MLKEM768,
	"SecP384r1MLKEM1024": tls.SecP384r1MLKEM1024,

	// Classic EC curves: IANA name (preferred) and Go constant name (alias).
	"secp256r1": tls.CurveP256,
	"secp384r1": tls.CurveP384,
	"secp521r1": tls.CurveP521,
	"CurveP256": tls.CurveP256,
	"CurveP384": tls.CurveP384,
	"CurveP521": tls.CurveP521,
}

func mapCurveNamesToIDs(rawTLSCurvePreferences []string) ([]tls.CurveID, error) {
	if rawTLSCurvePreferences == nil {
		return nil, nil
	}

	curvePreferences := []tls.CurveID{}

	for _, name := range rawTLSCurvePreferences {
		id, ok := curveIDs[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errUnknownTLSCurve, name)
		}

		curvePreferences = append(curvePreferences, id)
	}

	return curvePreferences, nil
}

// NewServerConfig provides new server TLS configuration.
func NewServerConfig(logger log.Logger, certFile, keyFile, minVersion string, cipherSuites, curvePreferences []string) (*tls.Config, error) {
	if certFile == "" && keyFile == "" {
		level.Info(logger).Log("msg", "TLS disabled; key and cert must be set to enable") //nolint:errcheck

		return nil, nil
	}

	level.Info(logger).Log("msg", "enabling server side TLS") //nolint:errcheck

	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("server credentials: %w", err)
	}

	version, err := flag.TLSVersion(minVersion)
	if err != nil {
		return nil, fmt.Errorf("TLS version invalid: %w", err)
	}

	cipherSuiteIDs, err := flag.TLSCipherSuites(cipherSuites)
	if err != nil {
		return nil, fmt.Errorf("TLS cipher suite name to ID conversion: %w", err)
	}

	curvePreferenceIDs, err := mapCurveNamesToIDs(curvePreferences)
	if err != nil {
		return nil, fmt.Errorf("TLS curve preference name to ID conversion: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		// A list of supported cipher suites for TLS versions up to TLS 1.2.
		// If CipherSuites is nil, a default list of secure cipher suites is used.
		// Note that TLS 1.3 ciphersuites are not configurable.
		CipherSuites: cipherSuiteIDs,
		// If CurvePreferences is nil, a default list of secure curves is used.
		CurvePreferences: curvePreferenceIDs,
		ClientAuth:       tls.RequestClientCert,
		MinVersion:       version,
	}

	return tlsCfg, nil
}

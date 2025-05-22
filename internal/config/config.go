package config

import (
	"fmt"
	stdlog "log"
	"regexp"
	"strings"

	"github.com/go-kit/log/level"
	flag "github.com/spf13/pflag"
)

var (
	validRule    = regexp.MustCompile(`^[_A-Za-z][\w]*$`)
	validPackage = regexp.MustCompile(`^[_A-Za-z][\w]*(\.[_A-Za-z][\w]*)*$`)
)

type Config struct {
	KubeconfigPath string
	DebugToken     string
	Name           string
	Mappings       map[string]string

	LogFormat string
	LogLevel  level.Option

	Opa       OPAConfig
	Server    ServerConfig
	TLS       TLSConfig
	Memcached MemcachedConfig
}

type OPAConfig struct {
	Pkg                 string
	Rule                string
	Matcher             string
	MatcherOp           string
	MatcherSkipTenants  string
	MatcherAdminGroups  string
	SSAR                bool
	ViaQToOTELMigration bool
}

type ServerConfig struct {
	Listen         string
	ListenInternal string
	HealthcheckURL string
}

type TLSConfig struct {
	MinVersion   string
	CipherSuites []string

	ServerCertFile string
	ServerKeyFile  string

	InternalServerCertFile string
	InternalServerKeyFile  string
	InternalServerCAFile   string
}

type MemcachedConfig struct {
	Expire   int32
	Interval int32
	Servers  []string
}

//nolint:cyclop
func ParseFlags() (*Config, error) {
	var rawTLSCipherSuites string

	cfg := &Config{}
	// Logger flags
	flag.StringVar(&cfg.Name, "debug.name", "opa-openshift", "A name to add as a prefix to log lines.")
	logLevelRaw := flag.String("log.level", "info", "The log filtering level. Options: 'error', 'warn', 'info', 'debug'.")
	flag.StringVar(&cfg.LogFormat, "log.format", "logfmt", "The log format to use. Options: 'logfmt', 'json'.")

	// Server flags
	flag.StringVar(&cfg.Server.Listen, "web.listen", ":8080", "The address on which the public server listens.")
	flag.StringVar(&cfg.Server.ListenInternal, "web.internal.listen", ":8081", "The address on which the internal server listens.")
	flag.StringVar(&cfg.Server.HealthcheckURL, "web.healthchecks.url", "http://localhost:8080", "The URL against which to run healthchecks.")

	flag.StringVar(&cfg.TLS.MinVersion, "tls.min-version", "VersionTLS13",
		"Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants.")
	flag.StringVar(&rawTLSCipherSuites, "tls.cipher-suites", "",
		"Comma-separated list of cipher suites for the server."+
			" Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants)."+
			" If omitted, the default Go cipher suites will be used."+
			" Note that TLS 1.3 ciphersuites are not configurable.")
	flag.StringVar(&cfg.TLS.ServerCertFile, "tls.server.cert-file", "",
		"File containing the default x509 Certificate for HTTPS. Leave blank to disable TLS.")
	flag.StringVar(&cfg.TLS.ServerKeyFile, "tls.server.key-file", "",
		"File containing the default x509 private key matching --tls.server.cert-file. Leave blank to disable TLS.")
	flag.StringVar(&cfg.TLS.InternalServerCertFile, "tls.internal.server.cert-file", "",
		"File containing the default x509 Certificate for internal HTTPS. Leave blank to disable TLS.")
	flag.StringVar(&cfg.TLS.InternalServerKeyFile, "tls.internal.server.key-file", "",
		"File containing the default x509 private key matching --tls.internal.server.cert-file. Leave blank to disable TLS.")
	flag.StringVar(&cfg.TLS.InternalServerCAFile, "tls.internal.server.ca-file", "",
		"File containing the TLS CA against which to verify servers."+
			" If no server CA is specified, the client will use the system certificates.")

	// OpenShift API flags
	flag.StringVar(&cfg.KubeconfigPath, "openshift.kubeconfig", "", "A path to the kubeconfig against to use for authorizing client requests.")
	mappingsRaw := flag.StringSlice("openshift.mappings", nil, "A map of tenantIDs to resource api groups to check to apply a given role to a user, e.g. tenant-a=observatorium.openshift.io") //nolint:lll

	// OPA flags
	flag.StringVar(&cfg.Opa.Pkg, "opa.package", "", "The name of the OPA package that opa-openshift should implement, see https://www.openpolicyagent.org/docs/latest/policy-language/#packages.")                              //nolint:lll
	flag.StringVar(&cfg.Opa.Rule, "opa.rule", "allow", "The name of the OPA rule for which opa-openshift should provide a result, see https://www.openpolicyagent.org/docs/latest/policy-language/#rules.")                     //nolint:lll
	flag.StringVar(&cfg.Opa.Matcher, "opa.matcher", "", "The label key of the OPA label matcher returned to the requesting client. When opa.matcher-op is provided alongside, multiple coma-separated values can be provided.") //nolint:lll
	flag.StringVar(&cfg.Opa.MatcherOp, "opa.matcher-op", "", "When several matchers are supplied (coma-separated string), this is the logical operation to perform. Allowed values: 'and', 'or'.")                              //nolint:lll
	flag.StringVar(&cfg.Opa.MatcherSkipTenants, "opa.skip-tenants", "", "Tenants for which the label matcher should not be set as comma-separated values.")
	flag.StringVar(&cfg.Opa.MatcherAdminGroups, "opa.admin-groups", "", "Groups which should be treated as admins and cause the matcher to be omitted.")
	flag.BoolVar(&cfg.Opa.SSAR, "opa.ssar", false, "Use SelftSubjectAccessReview instead of SubjectAccessReview.")
	flag.BoolVar(&cfg.Opa.ViaQToOTELMigration, "opa.viaq-to-otel-migration", false, "Enable the ViaQ to OTel migration.")

	// Memcached flags
	flag.StringSliceVar(&cfg.Memcached.Servers, "memcached", nil, "One or more Memcached server addresses.")
	flag.Int32Var(&cfg.Memcached.Expire, "memcached.expire", 60, "Time after which keys stored in Memcached should expire, given in seconds.")                 //nolint:lll,gomnd
	flag.Int32Var(&cfg.Memcached.Interval, "memcached.interval", 10, "The interval at which to update the Memcached DNS, given in seconds; use 0 to disable.") //nolint:lll,gomnd

	// Integration testing flags
	flag.StringVar(&cfg.DebugToken, "debug.token", "", "Debug bearer token used for integration tests.")

	err := flag.CommandLine.MarkHidden("debug.token")
	if err != nil {
		stdlog.Fatal("failed to mark flag hidden")
	}

	flag.Parse()

	ll, err := parseLogLevel(logLevelRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	cfg.LogLevel = ll

	if rawTLSCipherSuites != "" {
		cfg.TLS.CipherSuites = strings.Split(rawTLSCipherSuites, ",")
	}

	if len(cfg.Opa.Pkg) > 0 && !validPackage.Match([]byte(cfg.Opa.Pkg)) {
		return nil, fmt.Errorf("invalid OPA package name: %s", cfg.Opa.Pkg) //nolint:goerr113
	}

	if len(cfg.Opa.Rule) > 0 && !validRule.Match([]byte(cfg.Opa.Rule)) {
		return nil, fmt.Errorf("invalid OPA rule name: %s", cfg.Opa.Rule) //nolint:goerr113
	}

	if *mappingsRaw == nil {
		stdlog.Fatal("missing tenant mappings")
	}

	cfg.Mappings = make(map[string]string)

	for _, m := range *mappingsRaw {
		parts := strings.Split(m, "=")
		if len(parts) != 2 { //nolint:gomnd
			return nil, fmt.Errorf("invalid mapping: %q", m) //nolint:goerr113
		}

		cfg.Mappings[parts[0]] = parts[1]
	}

	if cfg.Opa.ViaQToOTELMigration {
		if !(strings.Contains(cfg.Opa.Matcher, "kubernetes_namespace_name") && strings.Contains(cfg.Opa.Matcher, "k8s_namespace_name")) {
			return nil, fmt.Errorf("OPA matcher must contain both 'kubernetes_namespace_name' and 'k8s_namespace_name' when ViaQ to OTel migration is enabled")
		}
	}

	return cfg, nil
}

func parseLogLevel(logLevelRaw *string) (level.Option, error) {
	switch *logLevelRaw {
	case "error":
		return level.AllowError(), nil
	case "warn":
		return level.AllowWarn(), nil
	case "info":
		return level.AllowInfo(), nil
	case "debug":
		return level.AllowDebug(), nil
	default:
		return nil, fmt.Errorf("unexpected log level: %s", *logLevelRaw) //nolint:goerr113
	}
}

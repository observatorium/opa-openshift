package main

import (
	"context"
	stdtls "crypto/tls"
	"crypto/x509"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/metalmatze/signal/healthcheck"
	"github.com/metalmatze/signal/internalserver"
	"github.com/metalmatze/signal/server/signalhttp"
	"github.com/observatorium/opa-openshift/internal/cache"
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/observatorium/opa-openshift/internal/handler"
	"github.com/observatorium/opa-openshift/internal/instrumentation"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/component-base/cli/flag"
)

const (
	dataEndpoint = "/v1/data"
)

//nolint:funlen,cyclop
func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		stdlog.Fatal(err)
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	if cfg.LogFormat == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	}

	logger = level.NewFilter(logger, cfg.LogLevel)
	if cfg.Name != "" {
		logger = log.With(logger, "name", cfg.Name)
	}

	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	defer level.Info(logger).Log("msg", "exiting")

	reg := prometheus.NewRegistry()
	hi := signalhttp.NewHandlerInstrumenter(reg, []string{"handler"})
	rti := instrumentation.NewRoundTripperInstrumenter(reg)
	healthchecks := healthcheck.NewMetricsHandler(healthcheck.NewHandler(), reg)

	var mc cache.Cacher

	if len(cfg.Memcached.Servers) > 0 {
		mc = cache.NewMemached(context.Background(), cfg.Memcached.Interval, cfg.Memcached.Expire, cfg.Memcached.Servers...)
	} else {
		mc = cache.NewInMemoryCache(cfg.Memcached.Expire)
	}

	if cacheMetrics, ok := mc.(cache.CacherWithMetrics); ok {
		reg.MustRegister(cacheMetrics)
	}

	wt := func(rt http.RoundTripper) http.RoundTripper {
		return rti.NewRoundTripper("openshift", rt)
	}

	p := path.Join(dataEndpoint, strings.ReplaceAll(cfg.Opa.Pkg, ".", "/"), cfg.Opa.Rule)
	level.Info(logger).Log("msg", "configuring the OPA endpoint", "path", p)

	l := log.With(logger, "component", "authorizer")
	m := http.NewServeMux()
	m.HandleFunc(p, hi.NewHandler(prometheus.Labels{"handler": "data"}, handler.New(l, mc, wt, cfg)))

	if cfg.Server.HealthcheckURL != "" {
		minVer, err := flag.TLSVersion(cfg.TLS.MinVersion)
		if err != nil {
			stdlog.Fatalf("failed to read TLS min version: %v", err)
		}

		t := (http.DefaultTransport).(*http.Transport).Clone()
		t.TLSClientConfig = &stdtls.Config{ //nolint:gosec
			MinVersion: minVer,
		}

		if cfg.TLS.InternalServerCAFile != "" {
			caCert, err := os.ReadFile(cfg.TLS.InternalServerCAFile)
			if err != nil {
				stdlog.Fatalf("failed to initialize healthcheck server TLS CA: %v", err)
			}

			t.TLSClientConfig.RootCAs = x509.NewCertPool()
			t.TLSClientConfig.RootCAs.AppendCertsFromPEM(caCert)
		}

		// checks if server is up
		healthchecks.AddLivenessCheck("http",
			healthcheck.HTTPCheckClient(
				&http.Client{Transport: t}, //nolint:exhaustivestruct
				cfg.Server.HealthcheckURL,
				http.MethodGet,
				http.StatusNotFound,
				time.Second,
			),
		)
	}

	level.Info(logger).Log("msg", "starting opa-openshift")

	var g run.Group
	{
		// Signal channels must be buffered.
		sig := make(chan os.Signal, 1)
		g.Add(func() error {
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			<-sig
			level.Info(logger).Log("msg", "caught interrupt")

			return nil
		}, func(_ error) {
			close(sig)
		})
	}
	{
		tlsConfig, err := config.NewServerConfig(
			log.With(logger, "protocol", "HTTP"),
			cfg.TLS.ServerCertFile,
			cfg.TLS.ServerKeyFile,
			cfg.TLS.MinVersion,
			cfg.TLS.CipherSuites,
		)
		if err != nil {
			stdlog.Fatal(err)

			return
		}

		//nolint:exhaustivestruct
		s := http.Server{
			Addr:      cfg.Server.Listen,
			Handler:   m,
			TLSConfig: tlsConfig,
		}

		g.Add(func() error {
			level.Info(logger).Log("msg", "starting the HTTP server", "address", cfg.Server.Listen)

			if tlsConfig != nil {
				// serverCertFile and serverKeyFile passed in TLSConfig at initialization.
				return s.ListenAndServeTLS("", "") //nolint:wrapcheck
			}

			return s.ListenAndServe() //nolint:wrapcheck
		}, func(_ error) {
			level.Info(logger).Log("msg", "shutting down the HTTP server")
			_ = s.Shutdown(context.Background())
		})
	}

	{
		tlsConfig, err := config.NewServerConfig(
			log.With(logger, "protocol", "HTTP"),
			cfg.TLS.InternalServerCertFile,
			cfg.TLS.InternalServerKeyFile,
			cfg.TLS.MinVersion,
			cfg.TLS.CipherSuites,
		)
		if err != nil {
			stdlog.Fatal(err)

			return
		}

		h := internalserver.NewHandler(
			internalserver.WithName("Internal - opa-openshift API"),
			internalserver.WithHealthchecks(healthchecks),
			internalserver.WithPrometheusRegistry(reg),
			internalserver.WithPProf(),
		)

		//nolint:exhaustivestruct
		s := http.Server{
			Addr:      cfg.Server.ListenInternal,
			Handler:   h,
			TLSConfig: tlsConfig,
		}

		g.Add(func() error {
			level.Info(logger).Log("msg", "starting internal HTTP server", "address", s.Addr)

			if tlsConfig != nil {
				// serverCertFile and serverKeyFile passed in TLSConfig at initialization.
				return s.ListenAndServeTLS("", "") //nolint:wrapcheck
			}

			return s.ListenAndServe() //nolint:wrapcheck
		}, func(_ error) {
			_ = s.Shutdown(context.Background())
		})
	}

	if err := g.Run(); err != nil {
		stdlog.Fatal(err)
	}
}

package main

import (
	"context"
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
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/observatorium/opa-openshift/internal/handler"
	"github.com/observatorium/opa-openshift/internal/instrumentation"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	dataEndpoint = "/v1/data"
)

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

	p := path.Join(dataEndpoint, strings.ReplaceAll(cfg.Opa.Pkg, ".", "/"), cfg.Opa.Rule)
	level.Info(logger).Log("msg", "configuring the OPA endpoint", "path", p)

	l := log.With(logger, "component", "authorizer")
	m := http.NewServeMux()
	m.HandleFunc(p,
		hi.NewHandler(
			prometheus.Labels{"handler": "data"},
			handler.New(rti, l, cfg),
		),
	)

	if cfg.Server.HealthcheckURL != "" {
		// checks if server is up
		healthchecks.AddLivenessCheck("http",
			healthcheck.HTTPCheckClient(
				&http.Client{}, //nolint:exhaustivestruct
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
		//nolint:exhaustivestruct
		s := http.Server{
			Addr:    cfg.Server.Listen,
			Handler: m,
		}

		g.Add(func() error {
			level.Info(logger).Log("msg", "starting the HTTP server", "address", cfg.Server.Listen)

			return s.ListenAndServe() //nolint:wrapcheck
		}, func(_ error) {
			level.Info(logger).Log("msg", "shutting down the HTTP server")
			_ = s.Shutdown(context.Background())
		})
	}
	{
		h := internalserver.NewHandler(
			internalserver.WithName("Internal - opa-openshift API"),
			internalserver.WithHealthchecks(healthchecks),
			internalserver.WithPrometheusRegistry(reg),
			internalserver.WithPProf(),
		)

		//nolint:exhaustivestruct
		s := http.Server{
			Addr:    cfg.Server.ListenInternal,
			Handler: h,
		}

		g.Add(func() error {
			level.Info(logger).Log("msg", "starting internal HTTP server", "address", s.Addr)

			return s.ListenAndServe() //nolint:wrapcheck
		}, func(_ error) {
			_ = s.Shutdown(context.Background())
		})
	}

	if err := g.Run(); err != nil {
		stdlog.Fatal(err)
	}
}

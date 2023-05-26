package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/observatorium/opa-openshift/internal/authorizer"
	"github.com/observatorium/opa-openshift/internal/cache"
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/observatorium/opa-openshift/internal/openshift"
	"k8s.io/client-go/transport"
)

const (
	xForwardedAccessTokenHeader = "X-Forwarded-Access-Token" //nolint:gosec
)

// Permission is an Observatorium RBAC permission.
type Permission string

const (
	// Write gives access to write data to a tenant.
	Write Permission = "write"
	// Read gives access to read data from a tenant.
	Read Permission = "read"
)

type Input struct {
	Groups     []string   `json:"groups"`
	Permission Permission `json:"permission"`
	Resource   string     `json:"resource"`
	Subject    string     `json:"subject"`
	Tenant     string     `json:"tenant"`
	Namespaces []string   `json:"namespaces"`
	Path       string     `json:"path"`
}

type dataRequestV1 struct {
	Input Input `json:"input"`
}

//nolint:cyclop,gocognit
func New(l log.Logger, c cache.Cacher, wt transport.WrapperFunc, cfg *config.Config) http.HandlerFunc {
	kubeconfigPath := cfg.KubeconfigPath
	tenantAPIGroups := cfg.Mappings
	debugToken := cfg.DebugToken
	matcher := cfg.Opa.ToMatcher()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "request must be a POST", http.StatusBadRequest)
			return //nolint:nlreturn
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return //nolint:nlreturn
		}
		defer r.Body.Close()

		var req dataRequestV1

		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, "failed to unmarshal JSON", http.StatusInternalServerError)
			return //nolint:nlreturn
		}

		apiGroup, ok := tenantAPIGroups[req.Input.Tenant]
		if !ok {
			http.Error(w, "unknown tenant", http.StatusInternalServerError)
			return //nolint:nlreturn
		}

		if req.Input.Resource == "" {
			http.Error(w, "unknown resource", http.StatusBadRequest)
			return //nolint:nlreturn
		}

		var verb string

		switch req.Input.Permission {
		case Read:
			verb = authorizer.GetVerb
		case Write:
			verb = authorizer.CreateVerb
		default:
			http.Error(w, "unknown permission", http.StatusBadRequest)
			return //nolint:nlreturn
		}

		token := r.Header.Get(xForwardedAccessTokenHeader)
		if token == "" {
			if debugToken == "" {
				http.Error(w, "missing forwarded access token", http.StatusBadRequest)

				return
			}

			token = debugToken

			level.Warn(l).Log("msg", "using debug.token in production environments is not recommended.")
		}

		oc, err := openshift.NewClient(wt, kubeconfigPath, token)
		if err != nil {
			http.Error(w, "failed to create openshift client", http.StatusInternalServerError)

			return
		}

		matcherForRequest := matcher.ForRequest(req.Input.Tenant, req.Input.Groups)

		a := authorizer.New(oc, l, c, matcherForRequest)

		res, err := a.Authorize(token, req.Input.Subject, req.Input.Groups, verb, req.Input.Tenant, req.Input.Resource, apiGroup, req.Input.Namespaces, req.Input.Path)
		if err != nil {
			statusCode := http.StatusInternalServerError
			//nolint:errorlint
			if sce, ok := err.(authorizer.StatusCoder); ok {
				statusCode = sce.StatusCode()
			}

			http.Error(w, err.Error(), statusCode)

			return
		}

		out, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return //nolint:nlreturn
		}

		_, err = w.Write(out)
		if err != nil {
			statusCode := http.StatusInternalServerError
			//nolint:errorlint
			if sce, ok := err.(authorizer.StatusCoder); ok {
				statusCode = sce.StatusCode()
			}

			http.Error(w, err.Error(), statusCode)

			return
		}
	}
}

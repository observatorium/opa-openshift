package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/observatorium/api/opa"
	"github.com/observatorium/api/rbac"
	"github.com/observatorium/opa-openshift/internal/authorizer"
	"github.com/observatorium/opa-openshift/internal/cache"
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/observatorium/opa-openshift/internal/openshift"
	"k8s.io/client-go/transport"
)

const (
	getVerb    = "get"
	createVerb = "create"

	xForwardedAccessTokenHeader = "X-Forwarded-Access-Token" //nolint:gosec
)

type dataRequestV1 struct {
	Input opa.Input `json:"input"`
}

//nolint:cyclop,gocognit
func New(l log.Logger, c cache.Cacher, wt transport.WrapperFunc, cfg *config.Config) http.HandlerFunc {
	kubeconfigPath := cfg.KubeconfigPath
	tenantAPIGroups := cfg.Mappings
	matcher := cfg.Opa.Matcher
	debugToken := cfg.DebugToken

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "request must be a POST", http.StatusBadRequest)
			return //nolint:nlreturn
		}

		body, err := ioutil.ReadAll(r.Body)
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
		case rbac.Read:
			verb = getVerb
		case rbac.Write:
			verb = createVerb
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

		a := authorizer.New(oc, l, c, matcher)

		res, err := a.Authorize(token, req.Input.Subject, req.Input.Groups, verb, req.Input.Tenant, req.Input.Resource, apiGroup)
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

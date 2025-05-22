package authorizer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/observatorium/opa-openshift/internal/cache"
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/observatorium/opa-openshift/internal/openshift"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/prometheus/prometheus/pkg/labels"
)

const (
	GetVerb    = "get"
	CreateVerb = "create"
)

type Authorizer struct {
	client  openshift.Client
	logger  log.Logger
	cache   cache.Cacher
	matcher *config.Matcher
}

type AuthzResponseData struct {
	Matchers  []*labels.Matcher `json:"matchers,omitempty"`
	MatcherOp config.MatcherOp  `json:"matcherOp,omitempty"`
}

type StatusCoder interface {
	StatusCode() int
}

type StatusCodeError struct {
	error
	SC int
}

func (s *StatusCodeError) StatusCode() int {
	return s.SC
}

func New(c openshift.Client, l log.Logger, cc cache.Cacher, matcher *config.Matcher) *Authorizer {
	return &Authorizer{client: c, logger: l, cache: cc, matcher: matcher}
}

func (a *Authorizer) Authorize(
	token,
	user string, groups []string,
	verb, resource, resourceName, apiGroup string,
	namespaces []string, metadataOnly bool,
) (types.DataResponseV1, error) {
	switch verb {
	case CreateVerb, GetVerb:
	default:
		return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("unexpected verb: %s", verb), http.StatusBadRequest}
	}

	cacheKey := generateCacheKey(token, user, groups, verb, resource, resourceName, apiGroup, namespaces, metadataOnly)

	level.Debug(a.logger).Log("msg", "looking up in cache", "cachekey", cacheKey) //nolint:errcheck
	res, ok, err := a.cache.Get(cacheKey)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to fetch authorization response from cache: %w", err), http.StatusInternalServerError}
	}

	if ok {
		level.Debug(a.logger).Log("msg", "cache hit", "cachekey", cacheKey) //nolint:errcheck
		return res, nil
	}

	res, err = a.authorizeInner(user, groups, verb, resource, resourceName, apiGroup, namespaces, metadataOnly)
	if err != nil {
		return types.DataResponseV1{}, err
	}

	if err := a.cache.Set(cacheKey, res); err != nil {
		// Only emit a warning when saving to cache fails, request still proceeds normally
		level.Warn(a.logger).Log("msg", fmt.Sprintf("failed to save cached response: %s", err), "cachekey", cacheKey) //nolint:errcheck
	}

	return res, nil
}

func (a *Authorizer) authorizeInner(user string, groups []string, verb, resource, resourceName, apiGroup string, namespaces []string, metadataOnly bool) (types.DataResponseV1, error) {
	// check if user has cluster-wide access
	clusterAllow, err := a.client.AccessReview(user, groups, verb, resource, resourceName, apiGroup, "")
	if err != nil {
		return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("cluster-wide SAR failed: %w", err), http.StatusUnauthorized}
	}

	//nolint:errcheck
	level.Debug(a.logger).Log(
		"msg", "cluster-scoped AccessReview",
		"user", user, "groups", fmt.Sprintf("%s", groups),
		"res", resource, "name", resourceName, "api", apiGroup,
		"allowed", clusterAllow,
	)

	if verb == CreateVerb {
		// No namespaced checks for log collection -> allow based on cluster-wide check
		return minimalDataResponseV1(clusterAllow), nil
	}

	if clusterAllow {
		// user has cluster-wide access -> per-namespace check is not meaningful (always successful)
		return a.authorizeClusterWide(namespaces)
	}

	if metadataOnly && len(namespaces) == 0 {
		// Only a metadata request and no namespaces provided -> populate with API list
		nsList, err := a.client.ListNamespaces()
		if err != nil {
			return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("failed to access api server: %w", err), http.StatusUnauthorized}
		}
		//nolint:errcheck
		level.Debug(a.logger).Log("msg", "list namespaces for meta request",
			"namespaces", fmt.Sprintf("%s", nsList),
		)

		if len(nsList) == 0 {
			// list of namespaces is empty -> deny
			return minimalDataResponseV1(false), nil
		}

		namespaces = nsList
	}

	allowed := []string{}
	for _, ns := range namespaces {
		nsAllowed, err := a.client.AccessReview(user, groups, verb, resource, resourceName, apiGroup, ns)
		if err != nil {
			return types.DataResponseV1{},
				&StatusCodeError{fmt.Errorf("namespaced SAR failed: %w", err), http.StatusUnauthorized}
		}
		//nolint:errcheck
		level.Debug(a.logger).Log(
			"msg", "namespace-scoped AccessReview",
			"user", user, "groups", fmt.Sprintf("%s", groups),
			"res", resource, "name", resourceName, "api", apiGroup,
			"allowed", nsAllowed, "namespace", ns,
		)

		if nsAllowed {
			allowed = append(allowed, ns)
		}
	}

	if len(allowed) == 0 {
		// all SARs were unsuccessful -> deny
		return minimalDataResponseV1(false), nil
	}

	// allow access for the namespaces where the SAR was successful
	res, err := newDataResponseV1(allowed, a.matcher)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to create auth response: %w", err), http.StatusInternalServerError}
	}

	return res, nil
}

func (a *Authorizer) authorizeClusterWide(namespaces []string) (types.DataResponseV1, error) {
	if a.matcher.IsEmpty() {
		// user has cluster-wide access and does not need matcher -> allow
		return minimalDataResponseV1(true), nil
	}

	// user has cluster-wide access but needs a matcher -> populate namespaces from API list
	nsList, err := a.client.ListNamespaces()
	if err != nil {
		return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("failed to access api server: %w", err), http.StatusUnauthorized}
	}

	if len(namespaces) == 0 {
		// request was cluster-scoped, return matcher with all accessible namespaces
		return newDataResponseV1(nsList, a.matcher)
	}

	nsMap := map[string]bool{}
	for _, ns := range nsList {
		nsMap[ns] = true
	}

	filtered := []string{}
	for _, ns := range namespaces {
		if nsMap[ns] {
			filtered = append(filtered, ns)
		}
	}

	// cluster-scoped SAR was successful, so namespaced SARs will be successful as well -> return matcher
	return newDataResponseV1(filtered, a.matcher)
}

func minimalDataResponseV1(allowed bool) types.DataResponseV1 {
	var res interface{} = allowed
	//nolint:exhaustivestruct
	return types.DataResponseV1{Result: &res}
}

func newDataResponseV1(ns []string, matcher *config.Matcher) (types.DataResponseV1, error) {
	if matcher.IsEmpty() && len(ns) > 0 {
		return minimalDataResponseV1(true), nil
	}

	matchers := []*labels.Matcher{}
	for _, key := range matcher.Keys {
		lm, err := labels.NewMatcher(labels.MatchRegexp, key, strings.Join(ns, "|"))
		if err != nil {
			return types.DataResponseV1{}, fmt.Errorf("failed to create new matcher: %w", err)
		}
		matchers = append(matchers, lm)
	}

	data, err := json.Marshal(&AuthzResponseData{
		Matchers:  matchers,
		MatcherOp: matcher.MatcherOp,
	})
	if err != nil {
		return types.DataResponseV1{}, fmt.Errorf("failed to marshal matcher to json: %w", err)
	}

	allowed := "true"
	if len(ns) == 0 {
		// Disallow request if no namespaces are allowed
		allowed = "false"
	}

	var res interface{} = map[string]string{
		"allowed": allowed,
		"data":    string(data),
	}

	//nolint:exhaustivestruct
	return types.DataResponseV1{Result: &res}, nil
}

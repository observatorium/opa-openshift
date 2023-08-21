package authorizer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
) (types.DataResponseV1, error) {
	cacheKey := generateCacheKey(token, user, groups, verb, resource, resourceName, apiGroup)

	res, ok, err := a.cache.Get(cacheKey)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to fetch authorization response from cache: %w", err), http.StatusInternalServerError}
	}

	if ok {
		return res, nil
	}

	allowed, err := a.client.SubjectAccessReview(user, groups, verb, resource, resourceName, apiGroup)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to authorize subject for auth backend role: %w", err), http.StatusUnauthorized}
	}

	level.Debug(a.logger).Log(
		"msg", "executed SubjectAccessReview",
		"user", user, "groups", fmt.Sprintf("%s", groups),
		"res", resource, "name", resourceName, "api", apiGroup,
		"allowed", allowed,
	)

	switch verb {
	case GetVerb:
		ns, err := a.client.ListNamespaces()
		if err != nil {
			return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("failed to access api server: %w", err), http.StatusUnauthorized}
		}
		if len(ns) == 0 {
			// Explicitly disallow a user query with no allowed namespaces
			allowed = false
		}
		level.Debug(a.logger).Log("msg", "executed ListNamespaces", "allowed", allowed)
		res, err = newDataResponseV1(allowed, ns, a.matcher)
		if err != nil {
			return types.DataResponseV1{},
				&StatusCodeError{fmt.Errorf("failed to create a new authorization response: %w", err), http.StatusInternalServerError}
		}
	case CreateVerb:
		// No namespace check needed as there won't be a query injection
		res = minimalDataResponseV1(allowed)
	default:
		// Verb was already validated in handler; at this step, an unexpected verb is a bug
		return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("unexpected verb: %s", verb), http.StatusInternalServerError}
	}

	err = a.cache.Set(cacheKey, res)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to store authorization response into cache: %w", err), http.StatusInternalServerError}
	}

	return res, nil
}

func minimalDataResponseV1(allowed bool) types.DataResponseV1 {
	var res interface{} = allowed
	//nolint:exhaustivestruct
	return types.DataResponseV1{Result: &res}
}

func newDataResponseV1(allowed bool, ns []string, matcher *config.Matcher) (types.DataResponseV1, error) {
	if matcher.IsEmpty() {
		return minimalDataResponseV1(allowed), nil
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

	var res interface{} = map[string]string{
		"allowed": strconv.FormatBool(allowed),
		"data":    string(data),
	}

	//nolint:exhaustivestruct
	return types.DataResponseV1{Result: &res}, nil
}

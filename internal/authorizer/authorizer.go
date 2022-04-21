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
	"github.com/observatorium/opa-openshift/internal/openshift"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/prometheus/prometheus/pkg/labels"
)

const clusterAdminGroup = "system:cluster-admins"

type Authorizer struct {
	client  openshift.Client
	logger  log.Logger
	cache   cache.Cacher
	matcher string
}

type AuthzResponse struct {
	Result bool   `json:"result"`
	Data   string `json:"data,omitempty"`
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

func New(c openshift.Client, l log.Logger, cc cache.Cacher, matcher string) *Authorizer {
	return &Authorizer{client: c, logger: l, cache: cc, matcher: matcher}
}

func (a *Authorizer) Authorize(
	token,
	user string, groups []string,
	verb, resource, resourceName, apiGroup string,
) (types.DataResponseV1, error) {
	res, ok, err := a.cache.Get(token)
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

	ns, err := a.client.ListNamespaces()
	if err != nil {
		return types.DataResponseV1{}, &StatusCodeError{fmt.Errorf("failed to access api server: %w", err), http.StatusUnauthorized}
	}

	isAdmin := groupsContainsClusterAdmin(groups)

	level.Debug(a.logger).Log(
		"msg", "result of admin check",
		"user", user, "admin", isAdmin,
	)

	res, err = newDataResponseV1(allowed, ns, a.matcher, isAdmin)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to create a new authorization response: %w", err), http.StatusInternalServerError}
	}

	err = a.cache.Set(token, res)
	if err != nil {
		return types.DataResponseV1{},
			&StatusCodeError{fmt.Errorf("failed to store authorization response into cache: %w", err), http.StatusInternalServerError}
	}

	return res, nil
}

func newDataResponseV1(allowed bool, ns []string, matcher string, isAdmin bool) (types.DataResponseV1, error) {
	var res interface{}
	if isAdmin || matcher == "" {
		res = allowed

		//nolint:exhaustivestruct
		return types.DataResponseV1{Result: &res}, nil
	}

	lm, err := labels.NewMatcher(labels.MatchRegexp, matcher, strings.Join(ns, "|"))
	if err != nil {
		return types.DataResponseV1{}, fmt.Errorf("failed to create new matcher: %w", err)
	}

	matchers := []*labels.Matcher{lm}

	data, err := json.Marshal(&matchers)
	if err != nil {
		return types.DataResponseV1{}, fmt.Errorf("failed to marshal matcher to json: %w", err)
	}

	res = map[string]string{
		"allowed": strconv.FormatBool(allowed),
		"data":    string(data),
	}

	//nolint:exhaustivestruct
	return types.DataResponseV1{Result: &res}, nil
}

func groupsContainsClusterAdmin(groups []string) bool {
	for _, g := range groups {
		if g == clusterAdminGroup {
			return true
		}
	}

	return false
}

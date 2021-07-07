package authorizer

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/observatorium/opa-openshift/internal/openshift"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/prometheus/prometheus/pkg/labels"
)

type Authorizer struct {
	client openshift.Client
	logger log.Logger
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

func New(c openshift.Client, l log.Logger) *Authorizer {
	return &Authorizer{client: c, logger: l}
}

func (a *Authorizer) Authorize(verb, resource, resourceName, apiGroup string) (bool, []string, error) {
	allowed, err := a.client.SelfSubjectAccessReview(verb, resource, resourceName, apiGroup)
	if err != nil {
		return false, nil, &StatusCodeError{fmt.Errorf("failed to authorize subject for auth backend role: %w", err), http.StatusUnauthorized}
	}

	ns, err := a.client.ListNamespaces()
	if err != nil {
		return false, nil, &StatusCodeError{fmt.Errorf("failed to access api server: %w", err), http.StatusUnauthorized}
	}

	return allowed, ns, nil
}

func NewDataResponseV1(allowed bool, ns []string, matcher string) (types.DataResponseV1, error) {
	var res interface{}
	if matcher == "" {
		res = allowed

		//nolint:exhaustivestruct
		return types.DataResponseV1{Result: &res}, nil
	}

	lm, err := labels.NewMatcher(labels.MatchRegexp, matcher, strings.Join(ns, "|"))
	if err != nil {
		return types.DataResponseV1{}, fmt.Errorf("failed to create new matcher: %w", err)
	}

	res = map[string]string{
		"allowed": strconv.FormatBool(allowed),
		"data":    lm.String(),
	}

	//nolint:exhaustivestruct
	return types.DataResponseV1{Result: &res}, nil
}

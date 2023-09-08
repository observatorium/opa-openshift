package authorizer

import (
	"errors"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/stretchr/testify/require"
)

type sarFunc func(user string, groups []string, verb, resource, resourceName, apiGroup, namespace string) (bool, error)

func simpleSARFunc(allowed bool, err error) sarFunc {
	return func(_ string, _ []string, _, _, _, _, _ string) (bool, error) {
		return allowed, err
	}
}

var (
	allowSAR = simpleSARFunc(true, nil)
	denySAR  = simpleSARFunc(false, nil)
)

type fakeClient struct {
	sarFunc sarFunc
	nsList  []string
	nsErr   error
}

func (f *fakeClient) SubjectAccessReview(user string, groups []string, verb, resource, resourceName, apiGroup, namespace string) (bool, error) {
	return f.sarFunc(user, groups, verb, resource, resourceName, apiGroup, namespace)
}

func (f *fakeClient) ListNamespaces() ([]string, error) {
	return f.nsList, f.nsErr
}

type fakeCache struct {
	getResponse types.DataResponseV1
	getFound    bool
	getErr      error
	setErr      error
}

func (f *fakeCache) Get(_ string) (types.DataResponseV1, bool, error) {
	return f.getResponse, f.getFound, f.getErr
}

func (f *fakeCache) Set(_ string, _ types.DataResponseV1) error {
	return f.setErr
}

func TestAuthorize(t *testing.T) {
	namespaceMatcher := &config.Matcher{
		Keys:      []string{"kubernetes_namespace_name"},
		MatcherOp: config.MatcherOr,
	}
	var namespaceResult interface{} = map[string]string{
		"allowed": "true",
		"data":    `{"matchers":[{"Type":2,"Name":"kubernetes_namespace_name","Value":"test-namespace-1"}],"matcherOp":"or"}`,
	}
	namespaceResponse := types.DataResponseV1{
		Result: &namespaceResult,
	}

	var namespaceResultDeny interface{} = map[string]string{
		"allowed": "false",
		"data":    `{"matchers":[{"Type":2,"Name":"kubernetes_namespace_name","Value":""}],"matcherOp":"or"}`,
	}
	namespaceResponseDeny := types.DataResponseV1{
		Result: &namespaceResultDeny,
	}
	tt := []struct {
		desc          string
		matcher       *config.Matcher
		cacheResponse types.DataResponseV1
		cacheFound    bool
		cacheGetErr   error
		cacheSetErr   error
		sarFunc       sarFunc
		nsList        []string
		nsErr         error
		verb          string
		namespaces    []string
		metadataOnly  bool
		wantAuthorize types.DataResponseV1
		wantErr       error
	}{
		{
			desc:    "allow - get, no matcher",
			matcher: config.EmptyMatcher(),
			sarFunc: allowSAR,
			nsList: []string{
				"test-namespace-1",
			},
			verb:          GetVerb,
			wantAuthorize: minimalDataResponseV1(true),
			wantErr:       nil,
		},
		{
			desc:    "allow - get, with matcher",
			matcher: namespaceMatcher,
			sarFunc: allowSAR,
			nsList: []string{
				"test-namespace-1",
			},
			verb:          GetVerb,
			wantAuthorize: namespaceResponse,
			wantErr:       nil,
		},
		{
			desc:          "deny - get, with matcher, no namespaces",
			matcher:       namespaceMatcher,
			sarFunc:       allowSAR,
			nsList:        []string{},
			verb:          GetVerb,
			wantAuthorize: namespaceResponseDeny,
			wantErr:       nil,
		},
		{
			desc:          "allow - create",
			matcher:       config.EmptyMatcher(),
			sarFunc:       allowSAR,
			nsList:        []string{},
			verb:          CreateVerb,
			wantAuthorize: minimalDataResponseV1(true),
			wantErr:       nil,
		},
		{
			desc:        "fail - cache get error",
			cacheGetErr: errors.New("get-cache error"),
			verb:        GetVerb,
			wantErr:     errors.New("failed to fetch authorization response from cache: get-cache error"),
		},
		{
			desc:    "fail - wrong verb",
			matcher: config.EmptyMatcher(),
			sarFunc: allowSAR,
			nsList:  []string{},
			verb:    "invalid",
			wantErr: errors.New("unexpected verb: invalid"),
		},
		{
			desc:    "fail - SAR error",
			matcher: config.EmptyMatcher(),
			sarFunc: simpleSARFunc(false, errors.New("test SAR error")),
			nsList: []string{
				"test-namespace-1",
			},
			verb:    GetVerb,
			wantErr: errors.New("cluster-wide SAR failed: test SAR error"),
		},
		{
			desc:    "fail - list namespace error",
			matcher: namespaceMatcher,
			sarFunc: allowSAR,
			nsErr:   errors.New("test list namespace error"),
			verb:    GetVerb,
			wantErr: errors.New("failed to access api server: test list namespace error"),
		},
		{
			desc:          "allow - cached",
			cacheResponse: namespaceResponse,
			cacheFound:    true,
			verb:          GetVerb,
			wantAuthorize: namespaceResponse,
		},
		{
			desc:    "allow - get, with matcher, namespaced",
			matcher: namespaceMatcher,
			sarFunc: func(_ string, _ []string, _, _, _, _, namespace string) (bool, error) {
				if namespace == "test-namespace-1" {
					return true, nil
				}

				return false, nil
			},
			nsList:        []string{"test-namespace-0", "test-namespace-1"},
			verb:          GetVerb,
			namespaces:    []string{"test-namespace-0", "test-namespace-1"},
			wantAuthorize: namespaceResponse,
		},
		{
			desc:    "allow - get, with matcher, namespaced, cluster-wide SAR",
			matcher: namespaceMatcher,
			sarFunc: func(_ string, _ []string, _, _, _, _, namespace string) (bool, error) {
				if namespace == "" || namespace == "test-namespace-1" {
					return true, nil
				}

				return false, nil
			},
			nsList:        []string{"test-namespace-1"},
			verb:          GetVerb,
			namespaces:    []string{"test-namespace-0", "test-namespace-1"},
			wantAuthorize: namespaceResponse,
		},
		{
			desc:    "allow - get, with matcher, no cluster-wide access, meta request",
			matcher: namespaceMatcher,
			sarFunc: func(_ string, _ []string, _, _, _, _, namespace string) (bool, error) {
				if namespace == "test-namespace-1" {
					return true, nil
				}

				return false, nil
			},
			nsList:        []string{"test-namespace-0", "test-namespace-1"},
			verb:          GetVerb,
			metadataOnly:  true,
			wantAuthorize: namespaceResponse,
		},
		{
			desc:          "deny - get, with matcher, namespaced, no namespaces",
			matcher:       namespaceMatcher,
			sarFunc:       denySAR,
			nsList:        []string{"test-namespace-0", "test-namespace-1"},
			verb:          GetVerb,
			namespaces:    []string{"test-namespace-0", "test-namespace-1"},
			wantAuthorize: minimalDataResponseV1(false),
		},
		{
			desc:    "fail - get, with matcher, namespaced SAR failure",
			matcher: namespaceMatcher,
			sarFunc: func(_ string, _ []string, _, _, _, _, namespace string) (bool, error) {
				if namespace == "" {
					return false, nil
				}

				return false, errors.New("namespaced SAR error")
			},
			nsList:     []string{"test-namespace-0", "test-namespace-1"},
			verb:       GetVerb,
			namespaces: []string{"test-namespace-1"},
			wantErr:    errors.New("namespaced SAR failed: namespaced SAR error"),
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			c := &fakeClient{
				sarFunc: tc.sarFunc,
				nsList:  tc.nsList,
				nsErr:   tc.nsErr,
			}
			l := log.NewNopLogger()
			cc := &fakeCache{
				getResponse: tc.cacheResponse,
				getFound:    tc.cacheFound,
				getErr:      tc.cacheGetErr,
				setErr:      tc.cacheSetErr,
			}

			a := New(c, l, cc, tc.matcher)
			authorize, err := a.Authorize(
				"test-token", "test-user", []string{"test-group-1"},
				tc.verb,
				"application", "logs", "loki.grafana.com",
				tc.namespaces, tc.metadataOnly,
			)

			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.wantErr.Error())
			}
			require.Equal(t, tc.wantAuthorize, authorize)
		})
	}
}

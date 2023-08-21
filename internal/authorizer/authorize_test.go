package authorizer

import (
	"errors"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	sarAllowed bool
	sarErr     error
	nsList     []string
	nsErr      error
}

func (f *fakeClient) SubjectAccessReview(_ string, _ []string, _, _, _, _ string) (bool, error) {
	return f.sarAllowed, f.sarErr
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
		sarAllowed    bool
		sarErr        error
		nsList        []string
		nsErr         error
		verb          string
		wantAuthorize types.DataResponseV1
		wantErr       error
	}{
		{
			desc:       "allow - get, no matcher",
			matcher:    config.EmptyMatcher(),
			sarAllowed: true,
			nsList: []string{
				"test-namespace-1",
			},
			verb:          GetVerb,
			wantAuthorize: minimalDataResponseV1(true),
			wantErr:       nil,
		},
		{
			desc:       "allow - get, with matcher",
			matcher:    namespaceMatcher,
			sarAllowed: true,
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
			sarAllowed:    true,
			nsList:        []string{},
			verb:          GetVerb,
			wantAuthorize: namespaceResponseDeny,
			wantErr:       nil,
		},
		{
			desc:          "allow - create",
			matcher:       config.EmptyMatcher(),
			sarAllowed:    true,
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
			desc:        "fail - cache set error",
			matcher:     config.EmptyMatcher(),
			cacheSetErr: errors.New("set-cache error"),
			sarAllowed:  true,
			nsList: []string{
				"test-namespace-1",
			},
			verb:    GetVerb,
			wantErr: errors.New("failed to store authorization response into cache: set-cache error"),
		},
		{
			desc:       "fail - wrong verb",
			matcher:    config.EmptyMatcher(),
			sarAllowed: true,
			nsList:     []string{},
			verb:       "invalid",
			wantErr:    errors.New("unexpected verb: invalid"),
		},
		{
			desc:       "fail - SAR error",
			matcher:    config.EmptyMatcher(),
			sarAllowed: true,
			sarErr:     errors.New("test SAR error"),
			nsList: []string{
				"test-namespace-1",
			},
			verb:    GetVerb,
			wantErr: errors.New("failed to authorize subject for auth backend role: test SAR error"),
		},
		{
			desc:       "fail - list namespace error",
			matcher:    namespaceMatcher,
			sarAllowed: true,
			nsErr:      errors.New("test list namespace error"),
			verb:       GetVerb,
			wantErr:    errors.New("failed to access api server: test list namespace error"),
		},
		{
			desc:          "allow - cached",
			matcher:       config.EmptyMatcher(),
			cacheResponse: namespaceResponse,
			cacheFound:    true,
			verb:          GetVerb,
			wantAuthorize: namespaceResponse,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			c := &fakeClient{
				sarAllowed: tc.sarAllowed,
				sarErr:     tc.sarErr,
				nsList:     tc.nsList,
				nsErr:      tc.nsErr,
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

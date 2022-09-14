package handler

import (
	"testing"

	"github.com/observatorium/opa-openshift/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMatcherForRequest(t *testing.T) {
	tt := []struct {
		desc        string
		opaConfig   config.OPAConfig
		tenant      string
		groups      []string
		wantMatcher string
	}{
		{
			desc:        "empty matcher",
			opaConfig:   config.OPAConfig{},
			tenant:      "",
			groups:      []string{},
			wantMatcher: "",
		},
		{
			desc: "normal",
			opaConfig: config.OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{"authenticated"},
			wantMatcher: "test-matcher",
		},
		{
			desc: "skip empty group",
			opaConfig: config.OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{""},
			wantMatcher: "test-matcher",
		},
		{
			desc: "tenant skipped",
			opaConfig: config.OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantB",
			groups:      []string{"authenticated"},
			wantMatcher: "",
		},
		{
			desc: "user is admin",
			opaConfig: config.OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{"admin-group"},
			wantMatcher: "",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			matcherForRequest := createMatcherFunc(tc.opaConfig)

			matcher := matcherForRequest(tc.tenant, tc.groups)

			require.Equal(t, tc.wantMatcher, matcher)
		})
	}
}

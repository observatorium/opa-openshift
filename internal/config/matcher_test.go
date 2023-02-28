package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatcherForRequest(t *testing.T) {
	tt := []struct {
		desc        string
		opaConfig   OPAConfig
		tenant      string
		groups      []string
		wantMatcher []string
	}{
		{
			desc:        "empty matcher",
			opaConfig:   OPAConfig{},
			tenant:      "",
			groups:      []string{},
			wantMatcher: nil,
		},
		{
			desc: "normal",
			opaConfig: OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{"authenticated"},
			wantMatcher: []string{"test-matcher"},
		},
		{
			desc: "multi-keys",
			opaConfig: OPAConfig{
				Matcher:            "test-matcher1,test-matcher2",
				MatcherOp:          string(MatcherOr),
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{"authenticated"},
			wantMatcher: []string{"test-matcher1", "test-matcher2"},
		},
		{
			desc: "skip empty group",
			opaConfig: OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{""},
			wantMatcher: []string{"test-matcher"},
		},
		{
			desc: "tenant skipped",
			opaConfig: OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantB",
			groups:      []string{"authenticated"},
			wantMatcher: nil,
		},
		{
			desc: "user is admin",
			opaConfig: OPAConfig{
				Matcher:            "test-matcher",
				MatcherSkipTenants: "tenantB,tenantC",
				MatcherAdminGroups: "admin-group,other-admin-group",
			},
			tenant:      "tenantA",
			groups:      []string{"admin-group"},
			wantMatcher: nil,
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			matcher := tc.opaConfig.ToMatcher()

			matcherForRequest := matcher.ForRequest(tc.tenant, tc.groups)

			require.Equal(t, tc.wantMatcher, matcherForRequest.Keys)
		})
	}
}

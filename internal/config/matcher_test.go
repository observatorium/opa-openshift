package config

import (
	"fmt"
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

			matchers, err := tc.opaConfig.ToMatchers()
			require.NoError(t, err)

			matcherForRequest := matchers.ForRequest(tc.tenant, tc.groups)

			require.Equal(t, tc.wantMatcher, matcherForRequest.Keys)
		})
	}
}

func TestMatchersConfigForRequest(t *testing.T) {
	matchersConfigStr := `
	{
		"byGroup": {
			"admin-group": {},
			"other-admin-group": {}
		},
		"byTenant": {
			"tenantA": {
				"keys": [
					"a-test-matcher"
				],
				"op": "or"
			},
			"tenantB": {
				"keys": [
					"b-test-matcher"
				],
				"op": "and"
			},
			"tenantC": {
				"keys": [
					"another-test-matcher"
				]
			}
		},
		"default": {
			"keys": [
				"test-matcher"
			]
		}
	}`

	tt := []struct {
		desc        string
		opaConfig   OPAConfig
		tenant      string
		groups      []string
		wantMatcher []string
		err         error
	}{
		{
			desc: "Matchers config from JSON string",
			opaConfig: OPAConfig{
				MatchersConfig: matchersConfigStr,
			},
			tenant:      "defaultTenant",
			groups:      []string{"authenticated"},
			wantMatcher: []string{"test-matcher"},
		},
		{
			desc: "Matchers config from JSON string for tenantA",
			opaConfig: OPAConfig{
				MatchersConfig: matchersConfigStr,
			},
			tenant:      "tenantA",
			groups:      []string{"authenticated"},
			wantMatcher: []string{"a-test-matcher"},
		},
		{
			desc: "Matchers config from JSON string for tenantB",
			opaConfig: OPAConfig{
				MatchersConfig: matchersConfigStr,
			},
			tenant:      "tenantB",
			groups:      []string{"authenticated"},
			wantMatcher: []string{"b-test-matcher"},
		},
		{
			desc: "Matchers config from JSON string for admin group",
			opaConfig: OPAConfig{
				MatchersConfig: matchersConfigStr,
			},
			tenant:      "anyTenant",
			groups:      []string{"admin-group"},
			wantMatcher: nil,
		},
		{
			desc: "Invalid config",
			opaConfig: OPAConfig{
				MatchersConfig: "{ invalid json }",
			},
			err: fmt.Errorf("invalid character 'i' looking for beginning of object key string"),
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			matchers, err := tc.opaConfig.ToMatchers()
			if tc.err != nil {
				require.Equal(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)

				matcherForRequest := matchers.ForRequest(tc.tenant, tc.groups)

				require.Equal(t, tc.wantMatcher, matcherForRequest.Keys)
			}
		})
	}
}

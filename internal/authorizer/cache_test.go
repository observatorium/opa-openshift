package authorizer

import (
	"testing"
)

const (
	// maxCacheKeyLength is the longest length the cache key should have. The value is based on the memcached documentation.
	maxCacheKeyLength = 250
)

func TestGenerateCacheKey(t *testing.T) {
	tt := []struct {
		desc         string
		token        string
		user         string
		groups       []string
		verb         string
		resource     string
		resourceName string
		apiGroup     string
		wantKey      string
	}{
		{
			desc:  "kubeadmin",
			token: "sha256~tokentokentokentokentokentokentokentokentok",
			user:  "kube:admin",
			groups: []string{
				"system:cluster-admins",
				"system:authenticated",
			},
			verb:         GetVerb,
			resource:     "logs",
			resourceName: "application",
			apiGroup:     "loki.grafana.com",
			wantKey:      "get,loki.grafana.com,application,logs,kube:admin:82516c2c21f2cb869241ffee091dd6e07b6fa1f74595536802d72de88b4c2130",
		},
		{
			desc:  "logcollector",
			token: "eytokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentok.eytokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokentokent",
			user:  "system:serviceaccount:openshift-logging:logcollector",
			groups: []string{
				"system:serviceaccounts",
				"system:serviceaccounts:openshift-logging",
				"system:authenticated",
			},
			verb:         CreateVerb,
			resource:     "logs",
			resourceName: "infrastructure",
			apiGroup:     "loki.grafana.com",
			wantKey:      "create,loki.grafana.com,infrastructure,logs,system:serviceaccount:openshift-logging:logcollector:4209c35b9ede6e39245d0c141006cb523d44bf65f04fdf834e164de263842753",
		},
		{
			desc:  "test user",
			token: "sha256~tokentokentokentokentokentokentokentokentok",
			user:  "testuser-0",
			groups: []string{
				"system:authenticated:oauth",
				"system:authenticated",
			},
			verb:         GetVerb,
			resource:     "logs",
			resourceName: "application",
			apiGroup:     "loki.grafana.com",
			wantKey:      "get,loki.grafana.com,application,logs,testuser-0:0cda1618ea4d6358ea3fb7e5270b8a85695fd4114a72f994fe71dde69df8d54a",
		},
	}

	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			got := generateCacheKey(tc.token, tc.user, tc.groups, tc.verb, tc.resource, tc.resourceName, tc.apiGroup)

			if got != tc.wantKey {
				t.Errorf("got cache key %q, want %q", got, tc.wantKey)
			}

			if len(got) > maxCacheKeyLength {
				t.Errorf("cache key is longer than %v characters: %v", maxCacheKeyLength, len(got))
			}
		})
	}
}

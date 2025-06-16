package authorizer

import (
	"testing"

	"github.com/observatorium/opa-openshift/internal/config"
)

const (
	// maxCacheKeyLength is the longest length the cache key should have. The value is based on the memcached documentation.
	maxCacheKeyLength = 250
)

func TestGenerateCacheKey(t *testing.T) {
	// Create a test matcher for consistent testing
	testMatcher := &config.Matcher{
		Keys:      []string{"kubernetes_namespace_name"},
		MatcherOp: config.MatcherOr,
	}
	newMatcher := &config.Matcher{
		Keys:      []string{"k8s_namespace_name"},
		MatcherOp: config.MatcherOr,
	}

	tt := []struct {
		desc         string
		token        string
		user         string
		groups       []string
		verb         string
		resource     string
		resourceName string
		apiGroup     string
		namespaces   []string
		metadataOnly bool
		matcher      *config.Matcher
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
			namespaces: []string{
				"log-test-0",
			},
			matcher: testMatcher,
			wantKey: "get,false,loki.grafana.com,application,logs,log-test-0,kube:admin:82516c2c21f2cb869241ffee091dd6e07b6fa1f74595536802d72de88b4c2130,m:e87a64ecd681d9831b31f30f429773801d276cf23e4b112cce2f077a1a092060",
		},
		{
			desc:  "kubeadmin - new OTEL matcher",
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
			namespaces: []string{
				"log-test-0",
			},
			matcher: newMatcher,
			wantKey: "get,false,loki.grafana.com,application,logs,log-test-0,kube:admin:82516c2c21f2cb869241ffee091dd6e07b6fa1f74595536802d72de88b4c2130,m:63ef1e06752e96333e3ae17570b81df7b194ad421c2a8920708832815bf0a6b0",
		},
		{
			desc:  "kubeadmin - empty matcher",
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
			namespaces: []string{
				"log-test-0",
			},
			matcher: config.EmptyMatcher(),
			wantKey: "get,false,loki.grafana.com,application,logs,log-test-0,kube:admin:82516c2c21f2cb869241ffee091dd6e07b6fa1f74595536802d72de88b4c2130,m:empty",
		},
		{
			desc:  "kubeadmin - nil matcher",
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
			namespaces: []string{
				"log-test-0",
			},
			matcher: nil,
			wantKey: "get,false,loki.grafana.com,application,logs,log-test-0,kube:admin:82516c2c21f2cb869241ffee091dd6e07b6fa1f74595536802d72de88b4c2130,m:empty",
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
			namespaces:   []string{},
			matcher:      testMatcher,
			wantKey:      "create,false,loki.grafana.com,infrastructure,logs,,system:serviceaccount:openshift-logging:logcollector:4209c35b9ede6e39245d0c141006cb523d44bf65f04fdf834e164de263842753,m:e87a64ecd681d9831b31f30f429773801d276cf23e4b112cce2f077a1a092060",
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
			namespaces: []string{
				"log-test-0",
			},
			matcher: testMatcher,
			wantKey: "get,false,loki.grafana.com,application,logs,log-test-0,testuser-0:0cda1618ea4d6358ea3fb7e5270b8a85695fd4114a72f994fe71dde69df8d54a,m:e87a64ecd681d9831b31f30f429773801d276cf23e4b112cce2f077a1a092060",
		},
		{
			desc:  "test user - metadata request",
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
			namespaces: []string{
				"log-test-0",
			},
			metadataOnly: true,
			matcher:      testMatcher,
			wantKey:      "get,true,loki.grafana.com,application,logs,log-test-0,testuser-0:0cda1618ea4d6358ea3fb7e5270b8a85695fd4114a72f994fe71dde69df8d54a,m:e87a64ecd681d9831b31f30f429773801d276cf23e4b112cce2f077a1a092060",
		},
	}

	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			got := generateCacheKey(tc.token, tc.user, tc.groups, tc.verb, tc.resource, tc.resourceName, tc.apiGroup, tc.namespaces, tc.metadataOnly, tc.matcher)

			if got != tc.wantKey {
				t.Errorf("got cache key %q, want %q", got, tc.wantKey)
			}

			if len(got) > maxCacheKeyLength {
				t.Errorf("cache key is longer than %v characters: %v", maxCacheKeyLength, len(got))
			}
		})
	}
}

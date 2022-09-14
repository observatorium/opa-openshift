package handler

import (
	"strings"

	"github.com/observatorium/opa-openshift/internal/config"
)

func prepareMap(csvInput string) map[string]struct{} {
	if csvInput == "" {
		return nil
	}

	tokens := strings.Split(csvInput, ",")

	skipMap := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}

		skipMap[token] = struct{}{}
	}

	return skipMap
}

func createMatcherFunc(cfg config.OPAConfig) func(tenant string, groups []string) string {
	matcher := cfg.Matcher
	skipTenants := prepareMap(cfg.MatcherSkipTenants)
	adminGroups := prepareMap(cfg.MatcherAdminGroups)

	return func(tenant string, groups []string) string {
		if matcher == "" {
			return ""
		}

		if _, skip := skipTenants[tenant]; skip {
			return ""
		}

		for _, group := range groups {
			if _, admin := adminGroups[group]; admin {
				return ""
			}
		}

		return matcher
	}
}

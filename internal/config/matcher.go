package config

import (
	"maps"
	"slices"
	"strings"
)

type MatcherOp string

const (
	MatcherOr         = MatcherOp("or")
	MatcherAnd        = MatcherOp("and")
	matchersSeparator = ","
)

type Matcher struct {
	Keys        []string
	MatcherOp   MatcherOp
	skipTenants map[string]struct{}
	adminGroups map[string]struct{}
}

func (m *Matcher) Clone() *Matcher {
	return &Matcher{
		Keys:        slices.Clone(m.Keys),
		MatcherOp:   m.MatcherOp,
		skipTenants: maps.Clone(m.skipTenants),
		adminGroups: maps.Clone(m.adminGroups),
	}
}

func (c *OPAConfig) ToMatcher() Matcher {
	matcherKeys := c.Matcher
	matcherOp := MatcherOp(c.MatcherOp)
	skipTenants := prepareMap(c.MatcherSkipTenants)
	adminGroups := prepareMap(c.MatcherAdminGroups)

	matcher := Matcher{
		MatcherOp:   matcherOp,
		skipTenants: skipTenants,
		adminGroups: adminGroups,
	}

	if keys := strings.Split(matcherKeys, matchersSeparator); len(keys) > 0 && keys[0] != "" {
		matcher.Keys = keys
	}

	return matcher
}

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

func (m *Matcher) IsEmpty() bool {
	return len(m.Keys) == 0
}

func (m *Matcher) IsSingle() bool {
	return len(m.Keys) == 1
}

func EmptyMatcher() *Matcher {
	return &Matcher{}
}

func (m *Matcher) ForRequest(tenant string, groups []string) *Matcher {
	if m.IsEmpty() {
		return m
	}

	if _, skip := m.skipTenants[tenant]; skip {
		return EmptyMatcher()
	}

	for _, group := range groups {
		if _, admin := m.adminGroups[group]; admin {
			return EmptyMatcher()
		}
	}

	return m.Clone() // Return a clone for request-specific modifications
}

func (m *Matcher) ViaQToOTELMigration(selectors map[string][]string) {
	if vals, ok := selectors["k8s_namespace_name"]; ok && len(vals) > 0 {
		if i := slices.Index(m.Keys, "kubernetes_namespace_name"); i != -1 {
			m.Keys = slices.Delete(m.Keys, i, i+1)
		}
		return
	}
	// Here we always delete the key "k8s_namespace_name" from the keys
	// to cover both the cases where kubernetes_namespace_name is present or no
	// selectors were present
	if i := slices.Index(m.Keys, "k8s_namespace_name"); i != -1 {
		m.Keys = slices.Delete(m.Keys, i, i+1)
	}
}

package config

import "strings"

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

	return m
}

func (m *Matcher) ViaQToOTELMigration(selectors map[string][]string) {
	if vals, ok := selectors["k8s_namespace_name"]; ok && len(vals) > 0 {
		m.Keys = deleteKey(m.Keys, "kubernetes_namespace_name")
		return
	}
	// Here we always delete the key "k8s_namespace_name" from the keys
	// to cover both the cases where kubernetes_namespace_name is present or no
	// selectors were present
	m.Keys = deleteKey(m.Keys, "k8s_namespace_name")
}

func deleteKey(keys []string, keyToDelete string) []string {
	for i, key := range keys {
		if key == keyToDelete {
			return append(keys[:i], keys[i+1:]...)
		}
	}
	return keys
}

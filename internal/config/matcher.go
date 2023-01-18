package config

import "strings"

type MatchersLogicalOp string

const (
	LogicalOr         = MatchersLogicalOp("or")
	LogicalAnd        = MatchersLogicalOp("and")
	matchersSeparator = ","
)

type Matcher struct {
	Keys        []string
	LogicalOp   MatchersLogicalOp
	skipTenants map[string]struct{}
	adminGroups map[string]struct{}
}

func (c *OPAConfig) ToMatcher() Matcher {
	matcherKeys := c.Matcher
	logicalOp := MatchersLogicalOp(c.MatchersLogicalOp)
	skipTenants := prepareMap(c.MatcherSkipTenants)
	adminGroups := prepareMap(c.MatcherAdminGroups)

	matcher := Matcher{
		LogicalOp:   logicalOp,
		skipTenants: skipTenants,
		adminGroups: adminGroups,
	}

	if logicalOp != "" {
		matcher.Keys = strings.Split(matcherKeys, matchersSeparator)
	} else if matcherKeys != "" {
		matcher.Keys = []string{matcherKeys}
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

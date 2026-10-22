package config

import (
	"encoding/json"
	"slices"
	"strings"
)

type MatcherOp string

const (
	MatcherOr         = MatcherOp("or")
	MatcherAnd        = MatcherOp("and")
	matchersSeparator = ","
)

type Matchers struct {
	ByGroup  map[string]Matcher `json:"byGroup,omitempty"`
	ByTenant map[string]Matcher `json:"byTenant,omitempty"`
	Default  Matcher            `json:"default,omitempty"`
}

type Matcher struct {
	Keys      []string  `json:"keys,omitempty"`
	MatcherOp MatcherOp `json:"op,omitempty"`
}

// Return a clone for request-specific modifications
func (m *Matcher) Clone() *Matcher {
	return &Matcher{
		Keys:      slices.Clone(m.Keys),
		MatcherOp: m.MatcherOp,
	}
}

func (c *OPAConfig) ToMatchers() (*Matchers, error) {
	var matchers Matchers
	if c.MatchersConfig != "" {
		err := json.Unmarshal([]byte(c.MatchersConfig), &matchers)
		if err != nil {
			return nil, err
		}
		return &matchers, nil
	}

	var defaultKeys []string
	if keys := strings.Split(c.Matcher, matchersSeparator); len(keys) > 0 && keys[0] != "" {
		defaultKeys = keys
	}

	matchers = Matchers{
		ByGroup:  emptyMatchers(c.MatcherAdminGroups),
		ByTenant: emptyMatchers(c.MatcherSkipTenants),
		Default: Matcher{
			Keys:      defaultKeys,
			MatcherOp: MatcherOp(c.MatcherOp),
		},
	}

	return &matchers, nil
}

func emptyMatchers(csvInput string) map[string]Matcher {
	if csvInput == "" {
		return nil
	}

	tokens := strings.Split(csvInput, ",")

	matcherMap := make(map[string]Matcher, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}

		matcherMap[token] = *EmptyMatcher()
	}

	return matcherMap
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

func (m *Matchers) ForRequest(tenant string, groups []string) *Matcher {
	for _, group := range groups {
		if m, found := m.ByGroup[group]; found {
			return m.Clone()
		}
	}

	if m, found := m.ByTenant[tenant]; found {
		return m.Clone()
	}

	return m.Default.Clone()
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

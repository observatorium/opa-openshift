package authorizer

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/observatorium/opa-openshift/internal/config"
)

func generateCacheKey(
	token, user string, groups []string,
	verb, resource, resourceName, apiGroup string, namespaces []string,
	metadataOnly bool, matcher *config.Matcher,
) string {
	userHash := hashUserinfo(token, user, groups)
	matcherHash := hashMatcher(matcher)

	return strings.Join([]string{
		verb, fmt.Sprintf("%v", metadataOnly),
		apiGroup, resourceName, resource, strings.Join(namespaces, ":"),
		userHash, matcherHash,
	}, ",")
}

func hashUserinfo(token, user string, groups []string) string {
	hash := sha256.New()
	hash.Write([]byte(token))
	hash.Write([]byte(user))

	sort.Strings(groups)
	for _, g := range groups {
		hash.Write([]byte(g))
	}

	hashBytes := hash.Sum([]byte{})
	return fmt.Sprintf("%s:%x", user, hashBytes)
}

func hashMatcher(matcher *config.Matcher) string {
	if matcher == nil || len(matcher.Keys) == 0 {
		return "m:empty"
	}

	// Include the Keys slice (which can be modified by ViaQToOTELMigration)
	keysCopy := slices.Clone(matcher.Keys)
	sort.Strings(keysCopy) // Sort to ensure consistent hash regardless of order

	hash := sha256.New()
	for _, key := range keysCopy {
		hash.Write([]byte(key))
	}

	hashBytes := hash.Sum([]byte{})
	return fmt.Sprintf("m:%x", hashBytes)
}

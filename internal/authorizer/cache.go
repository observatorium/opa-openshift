package authorizer

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

func generateCacheKey(
	token, user string, groups []string,
	verb, resource, resourceName, apiGroup string,
) string {
	userHash := hashUserinfo(token, user, groups)

	return strings.Join([]string{
		verb,
		apiGroup, resourceName, resource,
		userHash,
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

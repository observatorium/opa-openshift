package authorizer

import "strings"

const (
	pathLabels = "/loki/api/v1/labels"
)

func isMetaRequest(path string) bool {
	return path == pathLabels ||
		(strings.HasPrefix(path, "/loki/api/v1/label/") && strings.HasSuffix(path, "/values"))
}

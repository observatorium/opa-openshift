package cache

import (
	"encoding/json"
	"fmt"

	"github.com/open-policy-agent/opa/v1/server/types"
	"github.com/prometheus/client_golang/prometheus"
)

// Cacher is able to get and set key value pairs.
type Cacher interface {
	Get(string) (types.DataResponseV1, bool, error)
	Set(string, types.DataResponseV1) error
}

type CacherWithMetrics interface {
	Cacher
	prometheus.Collector
}

func toJSON(res types.DataResponseV1) ([]byte, error) {
	val, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataResponseV1 to JSON: %w", err)
	}

	return val, nil
}

func fromJSON(b []byte) (types.DataResponseV1, error) {
	var res types.DataResponseV1

	if err := json.Unmarshal(b, &res); err != nil {
		return types.DataResponseV1{}, fmt.Errorf("failed to convert JSON to DataResponseV1: %w", err)
	}

	return res, nil
}

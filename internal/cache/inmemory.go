package cache

import (
	"errors"
	"fmt"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/prometheus/client_golang/prometheus"
)

var errNotSupportedType = errors.New("value type not supported")

type inmemory struct {
	tc *ttlcache.Cache
}

func NewInMemoryCache(expire int32) CacherWithMetrics {
	tc := ttlcache.NewCache()
	_ = tc.SetTTL(time.Duration(int64(expire) * int64(time.Second)))
	tc.SkipTTLExtensionOnHit(true)

	return &inmemory{tc: tc}
}

func (i *inmemory) Describe(descs chan<- *prometheus.Desc) {
	descs <- descCacheInserts
	descs <- descCacheRequests
	descs <- descCacheEvictions
	descs <- descCacheItems
}

func (i *inmemory) Collect(metricsCh chan<- prometheus.Metric) {
	count := i.tc.Count()
	metrics := i.tc.GetMetrics()

	metricsCh <- prometheus.MustNewConstMetric(descCacheInserts, prometheus.CounterValue, float64(metrics.Inserted))
	metricsCh <- prometheus.MustNewConstMetric(descCacheRequests,
		prometheus.CounterValue, float64(metrics.Retrievals), metricsRequestResultHit)
	metricsCh <- prometheus.MustNewConstMetric(descCacheRequests,
		prometheus.CounterValue, float64(metrics.Misses), metricsRequestResultMiss)
	metricsCh <- prometheus.MustNewConstMetric(descCacheEvictions, prometheus.CounterValue, float64(metrics.Evicted))
	metricsCh <- prometheus.MustNewConstMetric(descCacheItems, prometheus.GaugeValue, float64(count))
}

func (i *inmemory) Get(k string) (types.DataResponseV1, bool, error) {
	val, err := i.tc.Get(k)
	if err != nil {
		if errors.Is(err, ttlcache.ErrNotFound) {
			return types.DataResponseV1{}, false, nil
		}

		return types.DataResponseV1{}, false, fmt.Errorf("failed to fetch from in-memory cache: %w", err)
	}

	switch v := val.(type) {
	case []byte:
		res, err := fromJSON(v)
		if err != nil {
			return types.DataResponseV1{}, false, err
		}

		return res, true, nil
	default:
		return types.DataResponseV1{}, false, fmt.Errorf("failed to read in-memory cache entry: %w", errNotSupportedType)
	}
}

func (i *inmemory) Set(k string, res types.DataResponseV1) error {
	v, err := toJSON(res)
	if err != nil {
		return err
	}

	if err := i.tc.Set(k, v); err != nil {
		return fmt.Errorf("failed to store in in-memory cache: %w", err)
	}

	return nil
}

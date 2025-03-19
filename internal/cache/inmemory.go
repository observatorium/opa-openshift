package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/open-policy-agent/opa/v1/server/types"
	"github.com/prometheus/client_golang/prometheus"
)

type inmemory struct {
	tc *ttlcache.Cache[string, []byte]
}

func NewInMemoryCache(expire int32) CacherWithMetrics {
	expireDuration := time.Duration(int64(expire) * int64(time.Second))
	tc := ttlcache.New(
		ttlcache.WithTTL[string, []byte](expireDuration),
		ttlcache.WithDisableTouchOnHit[string, []byte](),
	)

	return &inmemory{tc: tc}
}

func (i *inmemory) Describe(descs chan<- *prometheus.Desc) {
	descs <- descCacheInserts
	descs <- descCacheRequests
	descs <- descCacheEvictions
}

func (i *inmemory) Collect(metricsCh chan<- prometheus.Metric) {
	metrics := i.tc.Metrics()

	metricsCh <- prometheus.MustNewConstMetric(descCacheInserts, prometheus.CounterValue, float64(metrics.Insertions))
	metricsCh <- prometheus.MustNewConstMetric(descCacheRequests,
		prometheus.CounterValue, float64(metrics.Hits), metricsRequestResultHit)
	metricsCh <- prometheus.MustNewConstMetric(descCacheRequests,
		prometheus.CounterValue, float64(metrics.Misses), metricsRequestResultMiss)
	metricsCh <- prometheus.MustNewConstMetric(descCacheEvictions, prometheus.CounterValue, float64(metrics.Evictions))
}

func (i *inmemory) Get(k string) (types.DataResponseV1, bool, error) {
	item := i.tc.Get(k)

	if item == nil {
		return types.DataResponseV1{}, false, nil
	}

	res, err := fromJSON(item.Value())
	if err != nil {
		return types.DataResponseV1{}, false, err
	}

	return res, true, nil
}

func (i *inmemory) Set(k string, res types.DataResponseV1) error {
	v, err := toJSON(res)
	if err != nil {
		return err
	}

	// Save entry to cache using globally-defined TTL
	i.tc.Set(k, v, ttlcache.DefaultTTL)
	return nil
}

package cache

import "github.com/prometheus/client_golang/prometheus"

const (
	metricsPrefix            = "opa_openshift_cache_"
	metricNameCacheRequests  = metricsPrefix + "requests_total"
	metricNameCacheInserts   = metricsPrefix + "inserts_total"
	metricNameCacheEvictions = metricsPrefix + "evictions_total"

	metricsRequestResultHit  = "hit"
	metricsRequestResultMiss = "miss"
)

var (
	metricsLabels = []string{"result"}

	descCacheRequests = prometheus.NewDesc(
		metricNameCacheRequests,
		"Counts the number of retrieval requests to the cache.",
		metricsLabels, nil)
	descCacheInserts = prometheus.NewDesc(
		metricNameCacheInserts,
		"Counts the number of inserts into the cache.",
		nil, nil)
	descCacheEvictions = prometheus.NewDesc(
		metricNameCacheEvictions,
		"Counts the number of cache evictions.",
		nil, nil)
)

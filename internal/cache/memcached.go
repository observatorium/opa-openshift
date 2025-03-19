package cache

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/server/types"
	tcache "github.com/openshift/telemeter/pkg/cache"
	"github.com/openshift/telemeter/pkg/cache/memcached"
)

type memcache struct {
	mc tcache.Cacher
}

func NewMemached(ctx context.Context, interval, expiration int32, servers ...string) Cacher {
	mc := memcached.New(ctx, interval, expiration, servers...)

	return &memcache{mc: mc}
}

func (m *memcache) Get(k string) (types.DataResponseV1, bool, error) {
	b, ok, err := m.mc.Get(k)
	if err != nil {
		return types.DataResponseV1{}, false, fmt.Errorf("failed to fetch from memcached: %w", err)
	}

	if ok {
		res, err := fromJSON(b)
		if err != nil {
			return types.DataResponseV1{}, false, err
		}

		return res, true, nil
	}

	return types.DataResponseV1{}, false, nil
}

func (m *memcache) Set(k string, res types.DataResponseV1) error {
	v, err := toJSON(res)
	if err != nil {
		return err
	}

	if err := m.mc.Set(k, v); err != nil {
		return fmt.Errorf("failed to store in memcached: %w", err)
	}

	return nil
}

package cache

import (
	"errors"
	"fmt"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/open-policy-agent/opa/server/types"
)

var errNotSupportedType = errors.New("value type not supported")

type inmemory struct {
	tc ttlcache.SimpleCache
}

func NewInMemoryCache(expire int32) Cacher {
	tc := ttlcache.NewCache()
	_ = tc.SetTTL(time.Duration(expire * int32(time.Second)))

	return &inmemory{tc: tc}
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

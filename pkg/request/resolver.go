package request

import (
	"os"
	"strconv"

	"github.com/hbagdi/hit/pkg/cache"
)

type resolver interface {
	Resolve(string) (interface{}, error)
}

func newCacheResolver(cache cache.Cache, args []string) cacheResolver {
	return cacheResolver{
		cache: cache,
		args:  args,
	}
}

type cacheResolver struct {
	args  []string
	cache cache.Cache
}

func (r cacheResolver) Resolve(key string) (interface{}, error) {
	key = key[1:]
	n, err := strconv.Atoi(key)
	if err == nil && n < len(os.Args) {
		v := r.args[n]
		if v[0] != '@' {
			return v, nil
		}
		key = v[1:]
	}
	return r.cache.Get(key)
}

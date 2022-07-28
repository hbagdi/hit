package request

import (
	"fmt"
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
	if len(key) == 0 {
		return nil, fmt.Errorf("invalid reference '@'")
	}
	n, err := strconv.Atoi(key)
	if err == nil {
		// referenced key is a number
		if n >= len(r.args) {
			return nil, fmt.Errorf(
				"cannot find command-line argument number '@%d'", n)
		}
		if n == 0 {
			return nil, fmt.Errorf("positional argument must be greater than 0")
		}
		v := r.args[n]
		if v[0] != '@' {
			return typedValue(v), nil
		}
		key = v[1:]
	}
	return r.cache.Get(key)
}

const floatBitSize = 64

func typedValue(v string) interface{} {
	n, err := strconv.Atoi(v)
	if err == nil {
		return n
	}
	f, err := strconv.ParseFloat(v, floatBitSize)
	if err == nil {
		return f
	}
	if v == "true" {
		return true
	} else if v == "false" {
		return false
	}
	return v
}

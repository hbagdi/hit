package cache

import (
	"context"
	"fmt"
	"strings"

	"github.com/hbagdi/hit/pkg/db"
	"github.com/hbagdi/hit/pkg/model"
	"github.com/tidwall/gjson"
)

type DBCache struct {
	store *db.Store
}

func GetDBCache(store *db.Store) *DBCache {
	c := &DBCache{
		store: store,
	}
	return c
}

func (c *DBCache) Get(key string) (interface{}, error) {
	const splitN = 2
	splits := strings.SplitN(key, ".", splitN)
	if len(splits) != splitN {
		return nil, fmt.Errorf("invalid reference: '@%s'", key)
	}
	id := splits[0]
	path := splits[1]
	hit, err := c.store.LoadLatestHitForID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	js := gjson.ParseBytes(hit.Response.Body)
	res := js.Get(path)
	switch res.Type {
	case gjson.Null:
		return nil, fmt.Errorf("key not found: '%v'", key)
	case gjson.JSON:
		return nil, fmt.Errorf("found json, expected a string, "+
			"number or boolean for key '%v'", key)
	case gjson.Number:
		return res.Num, nil
	case gjson.False:
		return false, nil
	case gjson.True:
		return true, nil
	case gjson.String:
		return res.Str, nil
	default:
		panic(fmt.Sprintf("unexpected JSON data-type: %v", res.Type))
	}
}

func (c *DBCache) Save(hit model.Hit) error {
	return c.store.Save(context.Background(), hit)
}

func (c *DBCache) Flush() error {
	return nil
}

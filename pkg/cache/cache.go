package cache

import "github.com/hbagdi/hit/pkg/model"

type Cache interface {
	Get(key string) (interface{}, error)
	Save(hit model.Hit) error
	Flush() error
}

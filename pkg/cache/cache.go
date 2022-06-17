package cache

import (
	"net/http"
)

type Cache interface {
	Get(key string) (interface{}, error)
	Save(req http.Request, resp *http.Response) error
	Flush() error
}

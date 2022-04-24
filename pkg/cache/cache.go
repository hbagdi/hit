package cache

import (
	"net/http"

	"github.com/hbagdi/hit/pkg/parser"
)

type Cache interface {
	Get(key string) (interface{}, error)
	Save(req parser.Request, resp *http.Response) error
	Flush() error
}

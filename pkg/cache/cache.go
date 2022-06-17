package cache

type Cache interface {
	Get(key string) (interface{}, error)
	Save(hit Hit) error
	Flush() error
}

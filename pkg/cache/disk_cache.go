package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/hbagdi/hit/pkg/parser"
)

type DiskCache struct {
	m map[string]interface{}
}

func Get() *DiskCache {
	c := &DiskCache{}
	if err := c.load(); err != nil {
		panic(err)
	}
	return c
}

func (c *DiskCache) Get(key string) (interface{}, error) {
	pathElements := strings.Split(key, ".")
	var r interface{} = c.m
	for _, element := range pathElements {
		m, ok := r.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to index key: %v", key)
		}
		r, ok = m[element]
		if !ok {
			return nil, fmt.Errorf("key not found: %v", key)
		}
	}
	return r, nil
}

func (c *DiskCache) load() error {
	content, err := ioutil.ReadFile(cacheFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(content) == 0 {
		return nil
	}
	var m map[string]interface{}
	err = json.Unmarshal(content, &m)
	if err != nil {
		return err
	}

	c.m = m
	return nil
}

func (c *DiskCache) Save(req parser.Request, resp *http.Response) error {
	contentType := resp.Header.Get("content-type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var i interface{}
	err = json.Unmarshal(res, &i)
	if err != nil {
		return err
	}
	c.m[req.ID] = i
	return nil
}

func (c *DiskCache) Flush() error {
	f, err := json.Marshal(c.m)
	if err != nil {
		return fmt.Errorf("flush cache: marshal json: %v", err)
	}
	err = ioutil.WriteFile(cacheFilePath, f, fileMode)
	if err != nil {
		return fmt.Errorf("flush cache: write cache file: %v", err)
	}
	return nil
}

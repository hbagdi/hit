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
	"github.com/tidwall/gjson"
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
	jsonCache, err := json.Marshal(c.m)
	if err != nil {
		return nil, err
	}
	js := gjson.ParseBytes(jsonCache)
	res := js.Get(key)
	switch res.Type {
	case gjson.Null:
		return nil, fmt.Errorf("key not found: '%v'", key)
	case gjson.JSON:
		return nil, fmt.Errorf("found json, expected a string, "+
			"number or boolean for key '%v'", key)
	case gjson.Number:
		fallthrough
	case gjson.False:
		fallthrough
	case gjson.True:
		fallthrough
	case gjson.String:
		return res.Raw, nil
	default:
		panic(fmt.Sprintf("unexpected JSON data-type: %v", res.Type))
	}
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

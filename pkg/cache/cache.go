package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hbagdi/hit/pkg/parser"
)

var cacheFilePath string

func init() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("failed to find cache directory: %v", err))
	}
	cacheFilePath = filepath.Join(cacheDir, cacheFileName)
	_, err = os.Stat(cacheDir + "/hit")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.Mkdir(cacheDir+"/hit", os.ModePerm)
			if err != nil {
				panic(err)
			} else {
				return
			}
		}
		panic(err)
	}
}

const cacheFileName = "hit/cache.json"

func Load() (map[string]interface{}, error) {
	content, err := ioutil.ReadFile(cacheFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if len(content) == 0 {
		return map[string]interface{}{}, nil
	}
	var m map[string]interface{}
	err = json.Unmarshal(content, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func Save(req parser.Request, resp *http.Response) error {
	m, err := Load()
	if err != nil {
		return err
	}
	if resp.Header.Get("content-type") != "application/json" {
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
	m[req.ID] = i

	f, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cacheFilePath, f, 0600)
}

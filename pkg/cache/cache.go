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

const (
	cacheDir  = "hit"
	cacheFile = "cache.json"
	fileMode  = 0o0600
)

var cacheFilePath string

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("failed to find cache directory: %v", err))
	}
	if err = ensureDir(filepath.Join(userCacheDir, cacheDir),
		os.ModePerm); err != nil {
		panic(err)
	}
	if err = ensureFile(filepath.Join(userCacheDir,
		cacheDir, cacheFile), fileMode); err != nil {
		panic(err)
	}
	cacheFilePath = filepath.Join(userCacheDir, cacheDir, cacheFile)
}

func ensureFile(path string, perm os.FileMode) error {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.WriteFile(path, []byte("{}"), perm)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func ensureDir(path string, perm os.FileMode) error {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.Mkdir(path, perm)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

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

	return ioutil.WriteFile(cacheFilePath, f, fileMode)
}

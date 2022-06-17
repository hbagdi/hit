package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hbagdi/hit/pkg/util"
)

const (
	cacheFile = "cache.json"
	fileMode  = 0o0600
)

var cacheFilePath string

// init ensures that the cache files are correctly setup.
func init() {
	userCacheDir, err := util.GetUserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("failed to find cache directory: %v", err))
	}
	if err = ensureDir(userCacheDir, os.ModePerm); err != nil {
		panic(err)
	}
	hitCacheDir, err := util.HitCacheDir()
	if err != nil {
		panic(err)
	}
	if err = ensureDir(hitCacheDir, os.ModePerm); err != nil {
		panic(err)
	}
	hitCacheFile := filepath.Join(hitCacheDir, cacheFile)
	if err = ensureFile(hitCacheFile, fileMode); err != nil {
		panic(err)
	}
	cacheFilePath = hitCacheFile
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

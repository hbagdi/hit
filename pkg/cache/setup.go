package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	cacheDir  = "hit"
	cacheFile = "cache.json"
	fileMode  = 0o0600
)

var cacheFilePath string

// init ensures that the cache files are correctly setup.
func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("failed to find cache directory: %v", err))
	}
	if err = ensureDir(userCacheDir, os.ModePerm); err != nil {
		panic(err)
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

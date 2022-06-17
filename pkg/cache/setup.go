package cache

import (
	"errors"
	"fmt"
	"os"

	"github.com/hbagdi/hit/pkg/util"
)

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

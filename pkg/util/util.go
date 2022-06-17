package util

import (
	"fmt"
	"os"
	"path/filepath"
)

const cacheDir = "hit"

func GetUserCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to find user's cache directory: %w", err)
	}
	return userCacheDir, nil
}

func HitCacheDir() (string, error) {
	userCacheDir, err := GetUserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCacheDir, cacheDir), nil
}

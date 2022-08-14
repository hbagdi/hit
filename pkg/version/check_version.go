package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hbagdi/hit/pkg/client"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/hbagdi/hit/pkg/util"
	"go.uber.org/zap"
)

func checkForUpdate() (string, error) {
	c, err := client.NewHitClient(client.HitClientOpts{
		Logger:        log.Logger,
		HitCLIVersion: Version,
	})
	if err != nil {
		return "", fmt.Errorf("fetch latest version: %w", err)
	}
	latestVersion, err := c.LatestHitCLIVersion(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest verion: %w", err)
	}
	return latestVersion, nil
}

type data struct {
	Errored bool   `json:"errored"`
	Version string `json:"version"`
}

func parseVersionFromResponseOrFile(js []byte) (string, error) {
	var d data
	if err := json.Unmarshal(js, &d); err != nil {
		return "", err
	}
	if d.Errored {
		return "", fmt.Errorf("no version in cache")
	}
	return d.Version, nil
}

var versionCacheFullPath string

func versionCacheFileName() (string, error) {
	if versionCacheFullPath != "" {
		return versionCacheFullPath, nil
	}
	const versionCacheFilename = "latest_version.json"
	hitCacheDir, err := util.HitCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(hitCacheDir, versionCacheFilename), nil
}

var errCacheMiss = fmt.Errorf("cache miss")

func loadVersionFromCache() (string, error) {
	filename, err := versionCacheFileName()
	if err != nil {
		return "", err
	}
	js, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errCacheMiss
		}
		return "", err
	}
	cacheInfo, err := os.Stat(filename)
	if err != nil {
		return "", err
	}
	lastUpdated := cacheInfo.ModTime()
	cutoff := time.Now().Add(-24 * time.Hour)
	if lastUpdated.Before(cutoff) {
		return "", errCacheMiss
	}
	return parseVersionFromResponseOrFile(js)
}

func refreshVersionCache() (string, error) {
	version, err := checkForUpdate()
	updateCacheErr := updateCache(version, err != nil)
	if updateCacheErr != nil {
		log.Logger.Debug("version-check: failed to update version cache",
			zap.Error(updateCacheErr))
	}
	return version, err
}

func updateCache(version string, errored bool) error {
	const fileMode = 0o0600
	js, err := json.Marshal(map[string]interface{}{
		"version": version,
		"errored": errored,
	})
	if err != nil {
		return err
	}
	filename, err := versionCacheFileName()
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, js, fileMode)
	if err != nil {
		return fmt.Errorf("update version cache: %w", err)
	}
	return nil
}

func LoadLatestVersion() (string, error) {
	version, err := loadVersionFromCache()
	if err != nil {
		if err == errCacheMiss {
			updatedVersion, err := refreshVersionCache()
			if err != nil {
				return "", err
			}
			return updatedVersion, nil
		}
		return "", err
	}
	return version, nil
}

package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/blang/semver/v4"
	"github.com/hbagdi/hit/pkg/cache"
	executorPkg "github.com/hbagdi/hit/pkg/executor"
	"github.com/hbagdi/hit/pkg/version"
)

const (
	minArgs = 2
)

var (
	versionLoadMutex sync.Mutex
	latestVersion    string
)

func init() {
	go func() {
		versionLoadMutex.Lock()
		defer versionLoadMutex.Unlock()
		version, err := version.LoadLatestVersion()
		if err != nil {
			// TODO(hbagdi): add logging
			return
		}
		latestVersion = version
	}()
}

func getLatestVersion() string {
	versionLoadMutex.Lock()
	defer versionLoadMutex.Unlock()
	return latestVersion
}

func Run(ctx context.Context, args ...string) (err error) {
	if len(args) < minArgs {
		return fmt.Errorf("need a request to execute")
	}
	id := args[1]

	switch {
	case id == "completion":
		return executeCompletion()
	case id == "c1":
		return completion()
	case id == "version":
		return executeVersion()
	case id[0] == '@':
	default:
		return fmt.Errorf("request must begin with '@' character")
	}
	id = id[1:]

	cache := cache.Get()
	defer func() {
		flushErr := cache.Flush()
		if flushErr != nil {
			if err != nil {
				err = flushErr
			} else {
				// two errors, log the flush error and move on
				log.Println("failed to flush cache:", err)
			}
		}
	}()

	executor, err := executorPkg.NewExecutor(&executorPkg.Opts{
		Cache: cache,
	})
	if err != nil {
		return fmt.Errorf("initialize executor: %v", err)
	}
	defer executor.Close()

	err = executor.LoadFiles()
	if err != nil {
		return fmt.Errorf("read hit files: %v", err)
	}

	req, err := executor.BuildRequest(id, &executorPkg.RequestOpts{
		Params: args,
	})
	if err != nil {
		return fmt.Errorf("build request: %v", err)
	}

	err = printRequest(req.HTTPRequest)
	if err != nil {
		return fmt.Errorf("print request: %v", err)
	}

	resp, err := executor.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("execute request: %v", err)
	}

	err = printResponse(resp)
	if err != nil {
		return err
	}

	printLatestVersion()
	return err
}

func printLatestVersion() {
	latestVersion := getLatestVersion()
	if latestVersion == "" {
		return
	}
	latest, err := semver.New(cleanVersionString(latestVersion))
	if err != nil {
		// TODO(hbagdi): log error
		return
	}

	current, err := semver.New(cleanVersionString(version.Version))
	if err != nil {
		// TODO(hbagdi): log error
		return
	}
	if latest.GT(*current) {
		fmt.Printf("New version(%s) available! Current installed version is"+
			" %s.\nCheckout https://hit.yolo42.com for details.\n",
			latestVersion, version.Version)
	}
}

func cleanVersionString(v string) string {
	if strings.HasPrefix(v, "v") {
		return v[1:]
	}
	return v
}

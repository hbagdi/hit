package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/blang/semver/v4"
	"github.com/hbagdi/hit/pkg/cache"
	executorPkg "github.com/hbagdi/hit/pkg/executor"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/hbagdi/hit/pkg/version"
	"go.uber.org/zap"
)

const (
	minArgs = 2
)

var (
	versionLoadMutex sync.Mutex
	latestVersion    string
)

func setupLogger() {
	c := zap.NewDevelopmentConfig()
	c.OutputPaths = []string{"stderr"}
	c.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	zapLogger, err := c.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	log.Logger = zapLogger
}

func init() {
	setupLogger()
	go func() {
		versionLoadMutex.Lock()
		defer versionLoadMutex.Unlock()
		version, err := version.LoadLatestVersion()
		if err != nil {
			log.Logger.Debug("failed to load latest version", zap.Error(err))
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
	defer func() {
		_ = log.Logger.Sync()
	}()
	log.Logger.Debug("starting run cmd")
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
				log.Logger.Error("failed to flush cache:", zap.Error(err))
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
		log.Logger.Debug("failed to parse latest semantic version",
			zap.Error(err), zap.String("version", latestVersion))
		return
	}

	current, err := semver.New(cleanVersionString(version.Version))
	if err != nil {
		log.Logger.Debug("failed to parse current semantic version",
			zap.Error(err), zap.String("version", latestVersion))
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

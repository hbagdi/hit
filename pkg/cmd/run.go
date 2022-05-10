package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/hbagdi/hit/pkg/cache"
	executorPkg "github.com/hbagdi/hit/pkg/executor"
)

const (
	minArgs = 2
)

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

	return err
}

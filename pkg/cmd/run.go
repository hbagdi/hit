package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
)

const (
	minArgs = 2
	timeout = 10 * time.Second
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

	files, err := loadFiles()
	if err != nil {
		return fmt.Errorf("read hit files: %v", err)
	}
	req, err := fetchRequest(id, files)
	if err != nil {
		return fmt.Errorf("request '@%s' not found", id)
	}
	global, err := fetchGlobal(files)
	if err != nil {
		return err
	}
	cache := cache.Get()
	defer func() {
		err = cache.Flush()
	}()

	err = executeRequest(ctx, req, request.Options{
		GlobalContext: global,
		Cache:         cache,
		Args:          args,
	})
	return err
}

func executeRequest(ctx context.Context, req parser.Request, opts request.Options) error {
	httpReq, err := request.Generate(ctx, req, opts)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}

	err = printRequest(httpReq)
	if err != nil {
		return err
	}

	// execute
	if err != nil {
		return fmt.Errorf("failed to dump request: %v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	err = printResponse(resp)
	if err != nil {
		return err
	}

	// save cached response
	err = opts.Cache.Save(req, resp)
	if err != nil {
		return fmt.Errorf("saving response: %v", err)
	}
	return nil
}

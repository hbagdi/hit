package cmd

import (
	"context"
	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/db"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"github.com/elazarl/goproxy"

	"github.com/hbagdi/hit/pkg/log"
)

func executeCURL(ctx context.Context) {
	store, err := db.NewStore(ctx, db.StoreOpts{Logger: log.Logger})
	if err != nil {
		panic(err)
	}
	defer func() {
		err := store.Close()
		if err != nil {
			log.Logger.Sugar().Errorf("failed to close store: %v", err)
		}
	}()

	dbCache := cache.GetDBCache(store)
	defer func() {
		flushErr := dbCache.Flush()
		if flushErr != nil {
			if err != nil {
				err = flushErr
			} else {
				// two errors, log the flush error and move on
				log.Logger.Error("failed to flush cache:", zap.Error(err))
			}
		}
	}()

	var proxy = goproxy.NewProxyHttpServer()
	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			resp, _ := http.Get("http://httpbin.org/status/404")
			return req, resp
		})
	go func() {
		err := http.ListenAndServe(":8080", proxy)
		if err != nil {
			panic(err)
		}
	}()

	//executor, err := executorPkg.NewExecutor(&executorPkg.Opts{
	//	Cache: dbCache,
	//})
	//if err != nil {
	//	panic(err)
	//}
	//defer executor.Close()

	log.Logger.Debug("executing cmd")
	args := os.Args
	args = args[1:]

	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = []string{
		"http_proxy=http://localhost:8080",
		"https_proxy=http://localhost:8080",
	}

	err = cmd.Run()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			c := e.ExitCode()
			os.Exit(c)
		}
		//return model.Hit{}, err
	}
	log.Logger.Debug("executed cmd with no error")
}

package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/db"
	"github.com/hbagdi/hit/pkg/executor"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

var c cache.Cache

func init() {
	store, err := db.NewStore(db.StoreOpts{Logger: log.Logger})
	if err != nil {
		panic(fmt.Errorf("init test db: %v", err))
	}
	c = cache.GetDBCache(store)
}

func TestMain(m *testing.M) {
	var code int
	defer func() {
		err := c.Flush()
		if err != nil {
			panic(fmt.Sprintf("failed to flush cache: %v", err))
		}
		os.Exit(code)
	}()
	code = m.Run()
}

func testExecutor(t *testing.T) *executor.Executor {
	t.Helper()
	e, err := executor.NewExecutor(&executor.Opts{Cache: c})
	require.Nil(t, err)
	return e
}

func TestBasic(t *testing.T) {
	e := testExecutor(t)
	defer e.Close()

	require.Nil(t, e.LoadFiles())

	t.Run("successfully performs a basic request", func(t *testing.T) {
		id := "get-headers"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/headers", req.URL())
		require.Equal(t, "yes!!!!", req.Header.Get("global-header"))

		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.NotNil(t, res)
		require.Equal(t, http.StatusOK, res.Response.Code)

		body := gjsonBody(t, res.Response.Body)
		require.Equal(t, "bar", body.Get("headers.Foo").String())
		require.Equal(t, "yes!!!!", body.Get("headers.Global-Header").String())
	})
}

func gjsonBody(t *testing.T, body []byte) gjson.Result {
	t.Helper()
	if !gjson.Valid(string(body)) {
		require.FailNow(t, "invalid JSON in the body")
	}

	js := gjson.ParseBytes(body)
	return js
}

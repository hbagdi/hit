package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/executor"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

var c = cache.Get()

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
		req, err := e.BuildRequest("get-headers", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/headers",
			req.HTTPRequest.URL.String())

		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.NotNil(t, res)
		require.Equal(t, http.StatusOK, res.StatusCode)

		headerValue := gjsonBody(t, res).
			Get("headers.Foo").
			String()
		require.Equal(t, "bar", headerValue)
	})
	t.Run("populate cache", func(t *testing.T) {
		req, err := e.BuildRequest("populate-cache", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	})
	t.Run("request with body referencing cache", func(t *testing.T) {
		req, err := e.BuildRequest("get-using-cache", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything",
			req.HTTPRequest.URL.String())

		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)

		js := gjsonBody(t, res)
		str := js.Get("json.string").
			String()
		require.Equal(t, "foobar", str)

		boolFalse := js.Get("json.bool-false").
			Bool()
		require.Equal(t, false, boolFalse)

		boolTrue := js.Get("json.bool-true").
			Bool()
		require.Equal(t, true, boolTrue)

		num := js.Get("json.num").Value()
		require.Equal(t, float64(42), num)

		numFloat := js.Get("json.num-float").
			Num
		require.Equal(t, 42.42, numFloat)
	})
}

func gjsonBody(t *testing.T, res *http.Response) gjson.Result {
	t.Helper()
	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)
	require.Nil(t, err)
	if !gjson.Valid(string(bodyBytes)) {
		require.FailNow(t, "invalid JSON in the body")
	}

	js := gjson.ParseBytes(bodyBytes)
	return js
}

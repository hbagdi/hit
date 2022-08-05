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

	t.Run("successfully performs a request with static yaml", func(t *testing.T) {
		id := "post-static-json"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)

		hit, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.NotNil(t, hit)
		require.Equal(t, http.StatusOK, hit.Response.Code)

		body := gjsonBody(t, hit.Response.Body)
		require.Equal(t, "foobar", body.Get("json.string").String())
		require.Equal(t, int64(42), body.Get("json.num").Int())
		require.Equal(t, 42.42, body.Get("json.num-float").Float())
		require.Equal(t, false, body.Get("json.bool-false").Bool())
		require.Equal(t, true, body.Get("json.bool-true").Bool())
	})
	t.Run("ensure that request with no body has no content-type header", func(t *testing.T) {
		req, err := e.BuildRequest("get-headers", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/headers", req.URL())
		require.Equal(t, "httpbin.org", req.Header.Get("host"))
		require.Empty(t, req.Header.Get("content-type"))
	})
	t.Run("successfully performs a basic request", func(t *testing.T) {
		id := "get-headers"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/headers", req.URL())

		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.NotNil(t, res)
		require.Equal(t, http.StatusOK, res.Response.Code)

		headerValue := gjsonBody(t, res.Response.Body).
			Get("headers.Foo").
			String()
		require.Equal(t, "bar", headerValue)
	})
	t.Run("content-type header is set with static body", func(t *testing.T) {
		id := "post-with-static-body"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		body := gjsonBody(t, res.Response.Body)
		contentTypeHeaderValue := body.Get("headers.Content-Type").String()
		require.Equal(t, "application/json", contentTypeHeaderValue)
	})
	t.Run("populate cache", func(t *testing.T) {
		id := "populate-cache"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		body := gjsonBody(t, res.Response.Body)
		contentTypeHeaderValue := body.Get("headers.Content-Type").String()
		require.Equal(t, "application/json", contentTypeHeaderValue)
	})
	t.Run("request with body referencing cache", func(t *testing.T) {
		id := "get-using-cache"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything", req.URL())

		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)

		js := gjsonBody(t, res.Response.Body)
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
	t.Run("request with path referencing cache", func(t *testing.T) {
		id := "get-cache-ref-in-path"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything/foobar", req.URL())

		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)

		url := js.Get("url").String()
		require.Equal(t, "https://httpbin.org/anything/foobar", url)
	})
	t.Run("request with path referencing cache in a path segment", func(t *testing.T) {
		id := "get-cache-ref-in-path-in-middle"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything/foobar/baz", req.URL())

		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)

		url := js.Get("url").String()
		require.Equal(t, "https://httpbin.org/anything/foobar/baz", url)
	})
	t.Run("request with query param referencing cache", func(t *testing.T) {
		id := "get-cache-ref-in-query-param"
		req, err := e.BuildRequest(id, nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything/qp?foo=42", req.URL())

		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)

		url := js.Get("url").String()
		require.Equal(t, "https://httpbin.org/anything/qp?foo=42", url)
	})
	t.Run("non existent request errors", func(t *testing.T) {
		req, err := e.BuildRequest("get-does-not-exist", nil)
		require.NotNil(t, err)
		require.Equal(t, "request 'get-does-not-exist' not found", err.Error())
		require.Empty(t, req)
	})
	t.Run("no input from CLI returns an error", func(t *testing.T) {
		id := "cli-arg-types" //nolint:goconst
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Empty(t, req)
		require.ErrorContains(t, err,
			"cannot find command-line argument number '@1'")
	})
	t.Run("string input via CLI is injected", func(t *testing.T) {
		id := "cli-arg-types"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req", "foobar"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/foobar", req.URL())
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)
		input := js.Get("json.input").String()
		require.Equal(t, "foobar", input)
	})
	t.Run("referencing $0 errors", func(t *testing.T) {
		id := "bad-cli-arg"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Empty(t, req)
		require.NotNil(t, err)
		require.ErrorContains(t, err,
			"positional argument must be greater than 0")
	})
	t.Run("referencing $ errors", func(t *testing.T) {
		req, err := e.BuildRequest("invalid-ref", &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Empty(t, req)
		require.NotNil(t, err)
		require.ErrorContains(t, err,
			"invalid reference '@'")
	})
	t.Run("referencing a request ID only errors (path required)", func(t *testing.T) {
		req, err := e.BuildRequest("invalid-req-ref", &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Empty(t, req)
		require.NotNil(t, err)
		require.ErrorContains(t, err,
			"invalid reference: '@redirect'")
	})
	t.Run("number input via CLI is injected", func(t *testing.T) {
		id := "cli-arg-types"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req", "42"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/42", req.URL())
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)
		input := js.Get("json.input").Int()
		require.Equal(t, int64(42), input)
	})
	t.Run("float input via CLI is injected", func(t *testing.T) {
		id := "cli-arg-types"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req", "42.2442"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/42.2442", req.URL())
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)
		input := js.Get("json.input").Float()
		require.Equal(t, 42.2442, input)
	})
	t.Run("bool true input via CLI is injected", func(t *testing.T) {
		id := "cli-arg-types"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req", "true"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/true", req.URL())
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)
		input := js.Get("json.input").Bool()
		require.Equal(t, true, input)
	})
	t.Run("bool false input via CLI is injected", func(t *testing.T) {
		id := "cli-arg-types"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req", "false"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/false", req.URL())
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.Response.Code)
		js := gjsonBody(t, res.Response.Body)
		input := js.Get("json.input").Bool()
		require.Equal(t, false, input)
	})
	t.Run("redirects are not followed", func(t *testing.T) {
		id := "redirect"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/status/302", req.URL())
		res, err := e.Execute(context.Background(), id, req)
		require.Nil(t, err)
		require.Equal(t, http.StatusFound, res.Response.Code)
		require.Equal(t, res.Response.Header.Get("location"), "/redirect/1")
	})
	t.Run("an explicit host header is not overwritten", func(t *testing.T) {
		id := "request-with-host-header"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Nil(t, err)
		require.Equal(t, "foo.com", req.Header.Get("host"))
	})
	t.Run("no body encoding requests are sent as is", func(t *testing.T) {
		id := "no-body-encoding"
		req, err := e.BuildRequest(id, &executor.RequestOpts{
			Params: []string{"@req"},
		})
		require.Nil(t, err)
		require.Emptyf(t, req.Header.Get("content-type"),
			"no content-type header is set")
		require.Equal(t, "plain-text body", string(req.Body))
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

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
	t.Run("content-type header is set with static body", func(t *testing.T) {
		req, err := e.BuildRequest("post-with-static-body", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		body := gjsonBody(t, res)
		contentTypeHeaderValue := body.Get("headers.Content-Type").String()
		require.Equal(t, "application/json", contentTypeHeaderValue)
	})
	t.Run("populate cache", func(t *testing.T) {
		req, err := e.BuildRequest("populate-cache", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		body := gjsonBody(t, res)
		contentTypeHeaderValue := body.Get("headers.Content-Type").String()
		require.Equal(t, "application/json", contentTypeHeaderValue)
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
	t.Run("request with path referencing cache", func(t *testing.T) {
		req, err := e.BuildRequest("get-cache-ref-in-path", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything/foobar",
			req.HTTPRequest.URL.String())

		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)

		url := js.Get("url").String()
		require.Equal(t, "https://httpbin.org/anything/foobar", url)
	})
	t.Run("request with query param referencing cache", func(t *testing.T) {
		req, err := e.BuildRequest("get-cache-ref-in-query-param", nil)
		require.Nil(t, err)
		require.NotNil(t, req)
		require.Equal(t, "https://httpbin.org/anything/qp?foo=42",
			req.HTTPRequest.URL.String())

		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)

		url := js.Get("url").String()
		require.Equal(t, "https://httpbin.org/anything/qp?foo=42", url)
	})
	t.Run("non existent request errors", func(t *testing.T) {
		req, err := e.BuildRequest("get-does-not-exist", nil)
		require.NotNil(t, err)
		require.Equal(t, "request 'get-does-not-exist' not found", err.Error())
		require.Nil(t, req)
	})
	t.Run("string input via CLI is injected", func(t *testing.T) {
		req, err := e.BuildRequest("cli-arg-types", &executor.RequestOpts{
			Params: []string{"hit-test", "@req", "foobar"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/foobar",
			req.HTTPRequest.URL.String())
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)
		input := js.Get("json.input").String()
		require.Equal(t, "foobar", input)
	})
	t.Run("number input via CLI is injected", func(t *testing.T) {
		req, err := e.BuildRequest("cli-arg-types", &executor.RequestOpts{
			Params: []string{"hit-test", "@req", "42"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/42",
			req.HTTPRequest.URL.String())
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)
		input := js.Get("json.input").Int()
		require.Equal(t, int64(42), input)
	})
	t.Run("float input via CLI is injected", func(t *testing.T) {
		req, err := e.BuildRequest("cli-arg-types", &executor.RequestOpts{
			Params: []string{"hit-test", "@req", "42.2442"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/42.2442",
			req.HTTPRequest.URL.String())
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)
		input := js.Get("json.input").Float()
		require.Equal(t, 42.2442, input)
	})
	t.Run("bool true input via CLI is injected", func(t *testing.T) {
		req, err := e.BuildRequest("cli-arg-types", &executor.RequestOpts{
			Params: []string{"hit-test", "@req", "true"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/true",
			req.HTTPRequest.URL.String())
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)
		input := js.Get("json.input").Bool()
		require.Equal(t, true, input)
	})
	t.Run("bool false input via CLI is injected", func(t *testing.T) {
		req, err := e.BuildRequest("cli-arg-types", &executor.RequestOpts{
			Params: []string{"hit-test", "@req", "false"},
		})
		require.Nil(t, err)
		require.Equal(t, "https://httpbin.org/anything/false",
			req.HTTPRequest.URL.String())
		res, err := e.Execute(context.Background(), req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		js := gjsonBody(t, res)
		input := js.Get("json.input").Bool()
		require.Equal(t, false, input)
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

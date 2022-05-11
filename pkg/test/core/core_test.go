package core

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/executor"
	"github.com/stretchr/testify/require"
)

var c = cache.Get()

func testExecutor(t *testing.T) *executor.Executor {
	t.Helper()
	e, err := executor.NewExecutor(&executor.Opts{Cache: c})
	require.Nil(t, err)
	return e
}

func TestGET(t *testing.T) {
	e := testExecutor(t)
	defer e.Close()

	require.Nil(t, e.LoadFiles())
	req, err := e.BuildRequest("get-headers", nil)
	require.Nil(t, err)
	require.NotNil(t, req)
	require.Equal(t, "https://httpbin.org/headers",
		req.HTTPRequest.URL.String())

	res, err := e.Execute(context.Background(), req)
	require.Nil(t, err)
	require.NotNil(t, res)
	require.Equal(t, http.StatusOK, res.StatusCode)
	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)
	require.Nil(t, err)
	var m map[string]interface{}
	require.Nil(t, json.Unmarshal(bodyBytes, &m))
	h, ok := m["headers"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "bar", h["Foo"])
}

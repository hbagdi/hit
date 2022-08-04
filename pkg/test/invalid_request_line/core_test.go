package core

import (
	"fmt"
	"testing"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/db"
	"github.com/hbagdi/hit/pkg/executor"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/stretchr/testify/require"
)

var c cache.Cache

func init() {
	store, err := db.NewStore(db.StoreOpts{Logger: log.Logger})
	if err != nil {
		panic(fmt.Errorf("init test db: %v", err))
	}
	c = cache.GetDBCache(store)
}

func TestInvalidRequestLine(t *testing.T) {
	e, err := executor.NewExecutor(&executor.Opts{Cache: c})
	require.Nil(t, err)
	err = e.LoadFiles()
	require.ErrorContains(t, err, "invalid request line")
}

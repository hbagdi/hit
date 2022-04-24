package version

import (
	"context"
	"testing"

	"github.com/hbagdi/hit/pkg/cmd"
	"github.com/hbagdi/hit/pkg/test/util"
)

func TestVersion(t *testing.T) {
	c := util.NewStdCapture()
	defer c.Cleanup()
	err := cmd.Run(context.Background(), "test-binary-name", "version")
	if err != nil {
		t.Errorf("expected err to be nil but got %v", err)
	}
	c.Stop()

	if string(c.Stdout()) != "dev (commit: dev)\n" {
		t.Errorf("unexpected output")
	}
}

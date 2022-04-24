package version

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/hbagdi/hit/pkg/cmd"
	"github.com/hbagdi/hit/pkg/test/util"
)

func TestCompletion(t *testing.T) {
	c := util.NewStdCapture()
	defer c.Cleanup()
	err := cmd.Run(context.Background(), "test-binary-name", "completion")
	if err != nil {
		t.Errorf("expected err to be nil but got %v", err)
	}
	c.Stop()
	out := c.Stdout()

	expected, err := os.ReadFile("../../completion/hit-completion.bash")
	if err != nil {
		t.Errorf("not expected err: %v", err)
	}
	if !bytes.Equal(expected, out) {
		t.Errorf("unexpected output")
	}
}

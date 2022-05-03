package cmd

import (
	"fmt"

	"github.com/hbagdi/hit/pkg/version"
)

func executeVersion() error {
	fmt.Printf("%s (commit: %s)\n", version.Version, version.CommitHash)
	return nil
}

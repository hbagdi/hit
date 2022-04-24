package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hbagdi/hit/pkg/cmd"
)

func main() {
	exitCode := 0
	defer func() { os.Exit(exitCode) }()
	ctx := context.Background()
	ctx, done := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer done()
	if err := cmd.Run(ctx, os.Args...); err != nil {
		exitCode = 1
		fmt.Fprintln(os.Stderr, err)
	}
}

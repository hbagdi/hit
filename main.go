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
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	if err := cmd.Run(ctx, os.Args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

import (
	"bambuscribe/desktop/internal/cli"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := cli.New().Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"ok": false, "error": err.Error()})
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		os.Exit(1)
	}
}

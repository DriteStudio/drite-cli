package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/DriteStudio/drite-cli/internal/cli"
)

var version = "dev"

func main() {
	cli.Version = version
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	app := cli.NewApp(os.Stdin, os.Stdout, os.Stderr)
	os.Exit(app.Run(ctx, os.Args[1:]))
}

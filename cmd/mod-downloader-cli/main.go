package main

import (
	"context"
	"fmt"
	"os"

	"mod-downloader/cliapp"
)

func main() {
	app := cliapp.New(os.Stdout, os.Stderr)
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

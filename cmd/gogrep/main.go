package main

import (
	"io"
	"os"

	"github.com/winezer0/ripgrep/internal/app"
)

func main() {
	os.Exit(execute(os.Args[1:]))
}

func execute(args []string) int {
	stdin := io.Reader(os.Stdin)
	stdout := io.Writer(os.Stdout)
	stderr := io.Writer(os.Stderr)
	return app.Run(args, stdin, stdout, stderr)
}

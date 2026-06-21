package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rei0721/go-scaffold/internal/app/cliapp"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// main 是编译后二进制的进程入口。
func main() {
	if err := cliapp.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.GetExitCode(err))
	}
}

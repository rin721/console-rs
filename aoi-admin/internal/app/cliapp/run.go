package cliapp

import (
	"context"
	"io"
)

// Run 创建 CLI 应用并使用传入的 IO 运行。
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	app, err := NewApp(args...)
	if err != nil {
		return err
	}
	return app.RunWithIO(ctx, args, stdin, stdout, stderr)
}

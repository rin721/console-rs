package validators

import (
	"fmt"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// ValidateOptionalServerArg 限定 service 子命令当前只接受可选的 server 参数。
func ValidateOptionalServerArg(ctx *cli.Context) error {
	if len(ctx.Args) == 0 {
		return nil
	}
	if len(ctx.Args) == 1 && ctx.Args[0] == managed.ServiceServer {
		return nil
	}
	return &cli.UsageError{Command: ctx.CommandPath, Message: fmt.Sprintf("expected optional service name %q", managed.ServiceServer)}
}

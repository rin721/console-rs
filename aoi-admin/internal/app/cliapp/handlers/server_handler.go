package handlers

import (
	"fmt"

	cliconfig "github.com/rei0721/go-scaffold/internal/app/cliapp/config"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/server"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// ServerHandler 处理 server 命令入口。
type ServerHandler struct{}

func NewServerHandler() *ServerHandler {
	return &ServerHandler{}
}

func (h *ServerHandler) Execute(ctx *cli.Context) error {
	configPath := appconfig.ResolveConfigPath(ctx.GetString("config"), ctx.IsFlagChanged("config"))
	if _, diagnostics, err := appconfig.LoadDiagnostics(configPath); err != nil {
		return cliconfig.ActionableConfigLoadError(configPath, err)
	} else if len(diagnostics) > 0 {
		return cliconfig.ActionableConfigLoadError(configPath, fmt.Errorf("configuration diagnostics failed"))
	}
	return server.Run(configPath)
}

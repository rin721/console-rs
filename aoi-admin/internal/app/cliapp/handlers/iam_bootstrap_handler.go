package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// IAMBootstrapHandler 处理 iam bootstrap-admin 子命令。
type IAMBootstrapHandler struct{}

func NewIAMBootstrapHandler() *IAMBootstrapHandler {
	return &IAMBootstrapHandler{}
}

func (h *IAMBootstrapHandler) Execute(ctx *cli.Context) error {
	password := ctx.GetString("password")
	if ctx.GetBool("password-stdin") {
		raw, err := io.ReadAll(ctx.Stdin)
		if err != nil {
			return err
		}
		password = strings.TrimSpace(string(raw))
	}
	if password == "" {
		return &cli.UsageError{Command: ctx.CommandPath, Message: "password is required; pass --password or --password-stdin"}
	}

	core, err := initapp.NewCore(ctx.GetString("config"))
	if err != nil {
		return fmt.Errorf("initialize core: %w", err)
	}
	defer func() {
		if core.Logger != nil {
			_ = core.Logger.Sync()
		}
	}()
	infra, err := initapp.NewInfrastructure(core)
	if err != nil {
		return err
	}
	defer func() {
		if infra.Database != nil {
			_ = infra.Database.Close()
		}
	}()
	if err := initapp.ApplyConfiguredMigrations(core, infra); err != nil {
		return err
	}
	module, err := initapp.NewIAMModule(core, infra)
	if err != nil {
		return err
	}
	principal, err := module.Service.BootstrapAdmin(ctx.Context, iamservice.BootstrapAdminInput{
		OrgCode:     ctx.GetString("org-code"),
		OrgName:     ctx.GetString("org-name"),
		Username:    ctx.GetString("username"),
		Email:       ctx.GetString("email"),
		DisplayName: ctx.GetString("display-name"),
		Password:    password,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(ctx.Stdout).Encode(principal)
}
